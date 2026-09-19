//go:build integration

package postgresintegration_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func trackingFixture(t *testing.T) (context.Context, *pgxpool.Pool, *service.CourseGeneratorService, []domain.GenerationJob) {
	t.Helper()
	ctx, pool, initial := seedCompleteGeneration(t)
	ctx = contract.ContextWithPrincipal(ctx, contract.Principal{UserID: "user_consistency"})
	if _, err := pool.Exec(ctx, `DELETE FROM generation_jobs WHERE id=$1`, initial.ID); err != nil {
		t.Fatal(err)
	}
	course, err := postgres.NewCourseRepository(pool).FindCourseByRequestID(ctx, initial.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	jobs := []domain.GenerationJob{}
	for i := 0; i < 2; i++ {
		lesson := course.Modules[0].Lessons[0]
		lesson.ID, lesson.Order, lesson.ContentMarkdown = uuid.New(), i+2, nil
		if _, err := postgres.NewLessonRepository(pool).SaveLesson(ctx, lesson); err != nil {
			t.Fatal(err)
		}
		job, err := domain.NewGenerationJob(domain.NewGenerationJobParams{RequestID: initial.RequestID, Kind: domain.GenerationJobKindLessonContent, TargetID: &lesson.ID, IdempotencyKey: uuid.NewString()})
		if err != nil {
			t.Fatal(err)
		}
		job, err = postgres.NewGenerationJobRepository(pool).Enqueue(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, job)
	}
	s := service.NewCourseGeneratorService(consistencyAI{}, postgres.NewUnitOfWork(pool), consistencyClock{}, service.CourseGeneratorConfig{})
	return ctx, pool, s, jobs
}

func failTrackingJob(t *testing.T, ctx context.Context, pool *pgxpool.Pool, job domain.GenerationJob) domain.GenerationJob {
	t.Helper()
	claimed := claimConsistencyJob(t, ctx, pool, job.ID)
	claim, err := claimed.Claim()
	if err != nil {
		t.Fatal(err)
	}
	queue := postgres.NewGenerationJobRepository(pool)
	if err := queue.Fail(ctx, claim, errors.New("PRIVATE PROVIDER DIAGNOSTIC"), time.Now()); err != nil {
		t.Fatal(err)
	}
	failed, err := queue.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	return failed
}

func TestTrackingLocalFailureWaitsForSiblingsThenSettlesPartial(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	executor := service.NewGenerationJobExecutor(s)
	first := failTrackingJob(t, ctx, pool, jobs[0])
	if err := executor.HandleTerminalFailure(ctx, first, errors.New("failed")); err != nil {
		t.Fatal(err)
	}
	before, err := s.GetGenerationTracking(ctx, first.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if before.PipelineStatus != domain.PipelineStatusRunning || !before.HasActiveWork || before.Counts.JobsFailed != 1 || before.Counts.LessonsAvailable != 1 {
		t.Fatalf("local failure poisoned progress: %+v", before)
	}
	second := failTrackingJob(t, ctx, pool, jobs[1])
	if err := executor.HandleTerminalFailure(ctx, second, errors.New("failed")); err != nil {
		t.Fatal(err)
	}
	after, err := s.GetGenerationTracking(ctx, first.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if after.PipelineStatus != domain.PipelineStatusPartial || after.HasActiveWork || after.ContentAvailability != "partial" || after.Counts.LessonsExpected == nil || *after.Counts.LessonsExpected != 3 {
		t.Fatalf("incorrect partial outcome: %+v", after)
	}
	a, _ := new(big.Int).SetString(before.Revision, 10)
	b, _ := new(big.Int).SetString(after.Revision, 10)
	if b.Cmp(a) <= 0 {
		t.Fatal("revision did not advance")
	}
	encoded, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PRIVATE") || strings.Contains(string(encoded), "# Complete content") {
		t.Fatal("tracking leaked content or internal diagnostics")
	}
	page, err := s.GetGenerationEvents(ctx, first.RequestID, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatal("missing event pagination")
	}
	cursor, _ := new(big.Int).SetString(*page.NextCursor, 10)
	next, err := s.GetGenerationEvents(ctx, first.RequestID, cursor.Int64(), 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range next.Items {
		if event.ID == page.Items[0].ID || event.ID == page.Items[1].ID {
			t.Fatal("event cursor repeated rows")
		}
	}
	other := contract.ContextWithPrincipal(ctx, contract.Principal{UserID: "different_user"})
	if _, err := s.GetGenerationTracking(other, first.RequestID); !errors.Is(err, contract.ErrGenerationRequestNotFound) {
		t.Fatalf("ownership check: %v", err)
	}
}

func TestConcurrentBulkRetryIsAtomicIdempotentAndFencesOldFailures(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	params := contract.RetryOperationsParams{RequestID: jobs[0].RequestID, IdempotencyKey: uuid.NewString()}
	for i, job := range jobs {
		jobs[i] = failTrackingJob(t, ctx, pool, job)
		params.Operations = append(params.Operations, contract.RetryOperation{JobID: job.ID, OperationVersion: 1})
	}
	if _, err := s.ReconcileCompletedGeneration(ctx, params.RequestID); err != nil {
		t.Fatal(err)
	}
	limited := service.NewCourseGeneratorService(consistencyAI{}, postgres.NewUnitOfWork(pool), consistencyClock{}, service.CourseGeneratorConfig{MaxPendingJobs: 1})
	if _, err := limited.RetryGenerationOperations(ctx, params); !errors.Is(err, contract.ErrGenerationQueueSaturated) {
		t.Fatalf("expected atomic capacity rejection: %v", err)
	}
	assertConsistencyPair(t, ctx, pool, params.RequestID, "partial", "partial")
	var wg sync.WaitGroup
	results := make(chan contract.RetryOperationsResult, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := s.RetryGenerationOperations(ctx, params)
			results <- result
			failures <- err
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first *contract.RetryOperationsResult
	for result := range results {
		if first == nil {
			first = &result
		} else if !reflect.DeepEqual(*first, result) {
			t.Fatal("idempotent replay returned different replacements")
		}
	}
	if len(first.Replacements) != 2 {
		t.Fatal("bulk retry omitted an operation")
	}
	for _, old := range jobs {
		if err := service.NewGenerationJobExecutor(s).HandleTerminalFailure(ctx, old, errors.New("late old callback")); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := s.GetGenerationTracking(ctx, params.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.GenerationAttempt != 1 || snapshot.PipelineStatus != domain.PipelineStatusRunning || snapshot.Counts.JobsActive != 2 || snapshot.Counts.JobsFailed != 0 || snapshot.Counts.LessonsAvailable != 1 {
		t.Fatalf("retry disturbed sibling/content state: %+v", snapshot)
	}
	for _, op := range snapshot.Operations {
		if op.OperationVersion != 2 || op.SupersedesJobID == nil {
			t.Fatalf("operation has no explicit predecessor: %+v", op)
		}
	}
	stale := params
	stale.IdempotencyKey = uuid.NewString()
	if _, err := s.RetryGenerationOperations(ctx, stale); !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("obsolete selection accepted: %v", err)
	}
	changed := params
	changed.Operations = changed.Operations[:1]
	if _, err := s.RetryGenerationOperations(ctx, changed); !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("key reused with changed selection: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM generation_retry_commands WHERE request_id=$1`, params.RequestID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("receipt count=%d: %v", count, err)
	}
}

func TestTrackingLargeCourseStaysMetadataOnly(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	course, err := postgres.NewCourseRepository(pool).FindCourseByRequestID(ctx, jobs[0].RequestID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO lessons(id,module_id,lesson_order,title,type,estimated_duration_minutes,learning_goal,updated_at,content_markdown)
 SELECT gen_random_uuid(), $1, n, 'Lesson ' || n, 'theory', 10, 'Learn', now(), repeat('PRIVATE LESSON BODY', 1000) FROM generate_series(4,500) n`, course.Modules[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	snapshot, err := s.GetGenerationTracking(ctx, jobs[0].RequestID)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Counts.LessonsExpected == nil || *snapshot.Counts.LessonsExpected != 500 || snapshot.Counts.LessonsAvailable != 498 {
		t.Fatalf("counts: %+v", snapshot.Counts)
	}
	if len(encoded) > 150000 || strings.Contains(string(encoded), "PRIVATE LESSON BODY") {
		t.Fatalf("oversized or sensitive tracking: %d bytes", len(encoded))
	}
	t.Logf("500 lessons: tracking=%d bytes, query+authorization=%s", len(encoded), elapsed)
}

func TestOperationRetryAdmissionAlsoAppliesToActiveGeneration(t *testing.T) {
	ctx, pool, _, jobs := trackingFixture(t)
	failed := failTrackingJob(t, ctx, pool, jobs[0])
	limited := service.NewCourseGeneratorService(consistencyAI{}, postgres.NewUnitOfWork(pool), consistencyClock{}, service.CourseGeneratorConfig{MaxDailyPerUser: 1})
	_, err := limited.RetryGenerationOperations(ctx, contract.RetryOperationsParams{RequestID: failed.RequestID, IdempotencyKey: uuid.NewString(), Operations: []contract.RetryOperation{{JobID: failed.ID, OperationVersion: 1}}})
	if !errors.Is(err, contract.ErrGenerationDailyLimitExceeded) {
		t.Fatalf("daily admission: %v", err)
	}
	current, err := postgres.NewGenerationJobRepository(pool).FindByID(ctx, failed.ID)
	if err != nil || !current.IsCurrent || current.Status != domain.GenerationJobStatusFailed {
		t.Fatalf("rejected retry mutated operation: %+v %v", current, err)
	}
}

func TestOperationRetryRejectsPersistedContentAndIncompletePlanBarrier(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	failed := failTrackingJob(t, ctx, pool, jobs[0])
	params := contract.RetryOperationsParams{RequestID: failed.RequestID, IdempotencyKey: uuid.NewString(), Operations: []contract.RetryOperation{{JobID: failed.ID, OperationVersion: 1}}}
	if _, err := pool.Exec(ctx, `UPDATE lessons SET content_markdown='already available' WHERE id=$1`, failed.TargetID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RetryGenerationOperations(ctx, params); !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("must preserve available content: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE lessons SET content_markdown=NULL WHERE id=$1`, failed.TargetID); err != nil {
		t.Fatal(err)
	}
	course, err := postgres.NewCourseRepository(pool).FindCourseByRequestID(ctx, failed.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := postgres.NewModuleRepository(pool).SaveModules(ctx, []domain.Module{{ID: uuid.New(), CourseID: course.ID, Order: 2, Title: "Missing plan", Description: "Test", CreatedAt: now, UpdatedAt: now}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RetryGenerationOperations(ctx, params); !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("must respect plan barrier: %v", err)
	}
	snapshot, err := s.GetGenerationTracking(ctx, failed.RequestID)
	if err != nil || snapshot.Counts.LessonsExpected != nil || snapshot.Counts.PlansReady != 1 {
		t.Fatalf("unknown lesson count: %+v %v", snapshot.Counts, err)
	}
}

func TestTrackingPermanentConfigurationFailureRequiresOperator(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	failed := failTrackingJob(t, ctx, pool, jobs[0])
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET last_error_code='invalid_api_key' WHERE id=$1`, failed.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.GetGenerationTracking(ctx, failed.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range snapshot.Operations {
		if op.ID == failed.ID && (op.Retryable || op.FailureCode == nil || *op.FailureCode != "operator_action_required") {
			t.Fatalf("unsafe manual retry: %+v", op)
		}
	}
	_, err = s.RetryGenerationOperations(ctx, contract.RetryOperationsParams{RequestID: failed.RequestID, IdempotencyKey: uuid.NewString(), Operations: []contract.RetryOperation{{JobID: failed.ID, OperationVersion: 1}}})
	if !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("configuration retry allowed: %v", err)
	}
}

func TestTrackingReconcilesLegacyPartialAfterJobsWerePurged(t *testing.T) {
	ctx, pool, s, jobs := trackingFixture(t)
	requestID := jobs[0].RequestID
	for _, statement := range []string{
		`DELETE FROM generation_jobs WHERE request_id=$1`,
		`UPDATE generation_requests SET pipeline_status='failed', failure_message='Legacy failure', completed_at=now(), history_complete=false WHERE id=$1`,
		`UPDATE courses SET status='failed' WHERE request_id=$1`,
	} {
		if _, err := pool.Exec(ctx, statement, requestID); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ReconcileCompletedGenerations(ctx, 100); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.GetGenerationTracking(ctx, requestID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.PipelineStatus != domain.PipelineStatusPartial || snapshot.Counts.LessonsAvailable != 1 || snapshot.HistoryComplete || len(snapshot.Operations) != 0 {
		t.Fatalf("legacy failure was not requalified from preserved content: %+v", snapshot)
	}
}
