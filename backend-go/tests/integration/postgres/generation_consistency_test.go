//go:build integration

package postgresintegration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/Grimmjow06100/course-ai/backend-go/tests/testkit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type consistencyAI struct{ contract.CourseAIGenerator }
type consistencyClock struct{}

func (consistencyClock) Now() time.Time { return time.Now() }

func seedCompleteGeneration(t *testing.T) (context.Context, *pgxpool.Pool, domain.GenerationJob) {
	t.Helper()
	ctx, pool := testkit.OpenPostgres(t, 40*time.Second)
	rid, cid, mid, lid := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO generation_requests(id,clerk_user_id,initial_user_prompt,pipeline_status,started_at,updated_at) VALUES($1,'user_consistency','test','running',now(),now())`, rid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM generation_requests WHERE id=$1`, rid); err != nil {
			t.Error(err)
		}
	})
	_, err = pool.Exec(ctx, `INSERT INTO courses(id,request_id,clerk_user_id,language,status,initial_user_prompt,title,synopsis,current_level,target_level,updated_at) VALUES($1,$2,'user_consistency','en','content_generating','test','Test','Test course','beginner','intermediate',now())`, cid, rid)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	_, err = postgres.NewModuleRepository(pool).SaveModules(ctx, []domain.Module{{ID: mid, CourseID: cid, Order: 1, Title: "Module", Description: "Test", CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		t.Fatal(err)
	}
	content := "# Complete content"
	_, err = postgres.NewLessonRepository(pool).SaveLessons(ctx, []domain.Lesson{{ID: lid, ModuleID: mid, Order: 1, Title: "Lesson", Type: domain.LessonTypeTheory, LearningGoal: "Learn", EstimatedDurationMinutes: 10, ContentMarkdown: &content, CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		t.Fatal(err)
	}
	job, err := domain.NewGenerationJob(domain.NewGenerationJobParams{RequestID: rid, TargetID: &cid, Kind: domain.GenerationJobKindFinalizeCourse, IdempotencyKey: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	repo := postgres.NewGenerationJobRepository(pool)
	if _, err := repo.Enqueue(ctx, job); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE generation_jobs SET status='running',locked_by='consistency',locked_until=clock_timestamp()+interval '1 minute',started_at=now(),attempt_count=1 WHERE id=$1`, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	job, err = repo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool, job
}

type completionFaultUOW struct {
	base         contract.UnitOfWork
	beforeUpdate func() error
}

func (u completionFaultUOW) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return u.base.WithinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		return fn(ctx, completionFaultRepos{r, u.beforeUpdate})
	})
}

type completionFaultRepos struct {
	contract.TransactionalRepositories
	beforeUpdate func() error
}

func (r completionFaultRepos) GenerationRequests() contract.GenerationRequestRepository {
	return completionFaultRequests{r.TransactionalRepositories.GenerationRequests(), r.beforeUpdate}
}

type completionFaultRequests struct {
	contract.GenerationRequestRepository
	beforeUpdate func() error
}

func (r completionFaultRequests) UpdateGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		if err := r.beforeUpdate(); err != nil {
			return domain.GenerationRequest{}, err
		}
	}
	return r.GenerationRequestRepository.UpdateGenerationRequest(ctx, request)
}

func TestFinalizationRollbackAndConcurrentFailure(t *testing.T) {
	ctx, pool, job := seedCompleteGeneration(t)
	uow := postgres.NewUnitOfWork(pool)
	fault := errors.New("injected request update failure")
	broken := service.NewCourseGeneratorService(consistencyAI{}, completionFaultUOW{uow, func() error { return fault }}, consistencyClock{}, service.CourseGeneratorConfig{})
	if err := service.NewGenerationJobExecutor(broken).Execute(ctx, job); !errors.Is(err, fault) {
		t.Fatalf("expected injected failure: %v", err)
	}
	assertConsistencyPair(t, ctx, pool, job.RequestID, "running", "content_generating")
	generator := service.NewCourseGeneratorService(consistencyAI{}, uow, consistencyClock{}, service.CourseGeneratorConfig{})
	executor := service.NewGenerationJobExecutor(generator)
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 6; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); results <- executor.Execute(ctx, job) }()
		go func() { defer wg.Done(); results <- executor.HandleTerminalFailure(ctx, job, errors.New("late error")) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertConsistencyPair(t, ctx, pool, job.RequestID, "completed", "completed")
}

func TestLeaseExpiryRollsBackBusinessCompletion(t *testing.T) {
	ctx, pool, job := seedCompleteGeneration(t)
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET locked_until=clock_timestamp()+interval '1 second' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	entered := false
	uow := completionFaultUOW{postgres.NewUnitOfWork(pool), func() error { entered = true; time.Sleep(1100 * time.Millisecond); return nil }}
	generator := service.NewCourseGeneratorService(consistencyAI{}, uow, consistencyClock{}, service.CourseGeneratorConfig{})
	if err := service.NewGenerationJobExecutor(generator).Execute(ctx, job); !errors.Is(err, contract.ErrGenerationJobClaimLost) {
		t.Fatalf("expected expired claim: %v", err)
	}
	if !entered {
		t.Fatal("claim expired before the write; test did not exercise commit fencing")
	}
	assertConsistencyPair(t, ctx, pool, job.RequestID, "running", "content_generating")
}

func TestCompletionReconcilerWaitsForCurrentJobs(t *testing.T) {
	ctx, pool, job := seedCompleteGeneration(t)
	generator := service.NewCourseGeneratorService(consistencyAI{}, postgres.NewUnitOfWork(pool), consistencyClock{}, service.CourseGeneratorConfig{})
	if err := generator.ReconcileCompletedGenerations(ctx, 100); err != nil {
		t.Fatal(err)
	}
	assertConsistencyPair(t, ctx, pool, job.RequestID, "running", "content_generating")
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET status='failed',locked_by=NULL,locked_until=NULL,completed_at=now(),last_error_code='test',last_error_message='test' WHERE id=$1`, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := generator.ReconcileCompletedGenerations(ctx, 100); err != nil {
		t.Fatal(err)
	}
	assertConsistencyPair(t, ctx, pool, job.RequestID, "completed", "completed")
}

func assertConsistencyPair(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, wantRequest, wantCourse string) {
	t.Helper()
	var request, course string
	if err := pool.QueryRow(ctx, `SELECT r.pipeline_status::text,c.status::text FROM generation_requests r JOIN courses c ON c.request_id=r.id WHERE r.id=$1`, id).Scan(&request, &course); err != nil {
		t.Fatal(err)
	}
	if request != wantRequest || course != wantCourse {
		t.Fatalf("pair=%s/%s, want %s/%s", request, course, wantRequest, wantCourse)
	}
}

type parallelContentAI struct {
	contract.CourseAIGenerator
	entered chan struct{}
	release chan struct{}
}

func (a parallelContentAI) GenerateLessonContent(ctx context.Context, _ contract.LessonContentInput) (contract.LessonContentOutput, error) {
	a.entered <- struct{}{}
	select {
	case <-a.release:
		return contract.LessonContentOutput{ContentMarkdown: "# Generated content"}, nil
	case <-ctx.Done():
		return contract.LessonContentOutput{}, ctx.Err()
	}
}

func TestParallelLastLessonsEnqueueOneFinalizerAndReplaySafely(t *testing.T) {
	ctx, pool, initial := seedCompleteGeneration(t)
	if _, err := pool.Exec(ctx, `DELETE FROM generation_jobs WHERE id=$1`, initial.ID); err != nil {
		t.Fatal(err)
	}
	course, err := postgres.NewCourseRepository(pool).FindCourseByRequestID(ctx, initial.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	first := course.Modules[0].Lessons[0]
	first.ContentMarkdown = nil
	second := first
	second.ID, second.Order = uuid.New(), 2
	lessons := []domain.Lesson{first, second}
	if _, err := postgres.NewLessonRepository(pool).UpdateLesson(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := postgres.NewLessonRepository(pool).SaveLesson(ctx, second); err != nil {
		t.Fatal(err)
	}
	ai := parallelContentAI{entered: make(chan struct{}, 2), release: make(chan struct{})}
	executor := service.NewGenerationJobExecutor(service.NewCourseGeneratorService(ai, postgres.NewUnitOfWork(pool), consistencyClock{}, service.CourseGeneratorConfig{}))
	repo := postgres.NewGenerationJobRepository(pool)
	results := make(chan error, 2)
	for _, lesson := range lessons {
		job, err := domain.NewGenerationJob(domain.NewGenerationJobParams{RequestID: initial.RequestID, TargetID: &lesson.ID, Kind: domain.GenerationJobKindLessonContent, IdempotencyKey: uuid.NewString()})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Enqueue(ctx, job); err != nil {
			t.Fatal(err)
		}
		job = claimConsistencyJob(t, ctx, pool, job.ID)
		go func(job domain.GenerationJob) { results <- executor.Execute(ctx, job) }(job)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-ai.entered:
		case err := <-results:
			t.Fatalf("worker stopped before simultaneous generation: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	close(ai.release)
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := repo.ListByRequestID(ctx, initial.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	var finalizers []domain.GenerationJob
	for _, job := range jobs {
		if job.Kind == domain.GenerationJobKindFinalizeCourse {
			finalizers = append(finalizers, job)
		}
	}
	if len(finalizers) != 1 {
		t.Fatalf("finalizers=%d, want exactly one", len(finalizers))
	}
	finalizer := claimConsistencyJob(t, ctx, pool, finalizers[0].ID)
	if err := executor.Execute(ctx, finalizer); err != nil {
		t.Fatal(err)
	}
	// Replay after a lost acknowledgement must not degrade either lifecycle.
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET attempt_count=attempt_count+1 WHERE id=$1`, finalizer.ID); err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(ctx, finalizer); !errors.Is(err, contract.ErrGenerationJobClaimLost) {
		t.Fatalf("old claim was not rejected: %v", err)
	}
	finalizer, err = repo.FindByID(ctx, finalizer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := executor.Execute(ctx, finalizer); err != nil {
		t.Fatalf("replay after lost acknowledgement: %v", err)
	}
	assertConsistencyPair(t, ctx, pool, initial.RequestID, "completed", "completed")
}

func claimConsistencyJob(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) domain.GenerationJob {
	t.Helper()
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET status='running',locked_by='consistency',locked_until=clock_timestamp()+interval '1 minute',started_at=now(),attempt_count=1 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	job, err := postgres.NewGenerationJobRepository(pool).FindByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return job
}
