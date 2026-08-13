package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestStartFullCourseGenerationOnlyPersistsCommand(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	requests := make(map[uuid.UUID]domain.GenerationRequest)
	queue := newMemoryGenerationJobQueue()
	service := NewCourseGeneratorService(
		fakeCourseAI{},
		&fakeUnitOfWork{requests: requests, jobs: queue},
		fixedClock{now: now},
		CourseGeneratorConfig{},
	)

	started, err := service.StartFullCourseGeneration(context.Background(), contract.StartGenerationParams{
		Prompt:         "Build a Linux course",
		IdempotencyKey: "request-42",
	})
	if err != nil {
		t.Fatalf("start full course generation: %v", err)
	}
	if started.JobID == uuid.Nil || started.RequestID == uuid.Nil {
		t.Fatalf("missing accepted identifiers: %+v", started)
	}
	if started.Status != domain.PipelineStatusQueued || started.JobStatus != domain.GenerationJobStatusQueued {
		t.Fatalf("unexpected accepted status: %+v", started)
	}
	if len(requests) != 1 || queue.count() != 1 {
		t.Fatalf("requests=%d jobs=%d, want one atomic command", len(requests), queue.count())
	}
	job, err := queue.FindByID(context.Background(), started.JobID)
	if err != nil {
		t.Fatalf("find queued job: %v", err)
	}
	if job.Kind != domain.GenerationJobKindFullCourse || job.RequestID != started.RequestID {
		t.Fatalf("unexpected queued job: %+v", job)
	}
}

func TestStartFullCourseGenerationIsIdempotent(t *testing.T) {
	t.Parallel()

	requests := make(map[uuid.UUID]domain.GenerationRequest)
	queue := newMemoryGenerationJobQueue()
	service := NewCourseGeneratorService(
		fakeCourseAI{},
		&fakeUnitOfWork{requests: requests, jobs: queue},
		fixedClock{now: time.Now()},
		CourseGeneratorConfig{},
	)
	params := contract.StartGenerationParams{Prompt: "Build a Docker course", IdempotencyKey: "same-command"}

	first, err := service.StartFullCourseGeneration(context.Background(), params)
	if err != nil {
		t.Fatalf("first command: %v", err)
	}
	second, err := service.StartFullCourseGeneration(context.Background(), params)
	if err != nil {
		t.Fatalf("idempotent command: %v", err)
	}
	if first.JobID != second.JobID || first.RequestID != second.RequestID || queue.count() != 1 || len(requests) != 1 {
		t.Fatalf("idempotency failed: first=%+v second=%+v requests=%d jobs=%d", first, second, len(requests), queue.count())
	}

	_, err = service.StartFullCourseGeneration(context.Background(), contract.StartGenerationParams{
		Prompt:         "A different prompt",
		IdempotencyKey: params.IdempotencyKey,
	})
	if !errors.Is(err, contract.ErrGenerationJobIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
}

func TestEnqueueCourseStructureAndRetry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	request := analyzedGenerationRequest(t, now)
	requests := map[uuid.UUID]domain.GenerationRequest{request.ID: request}
	queue := newMemoryGenerationJobQueue()
	courses := &fakeCourseRepository{}
	service := NewCourseGeneratorService(fakeCourseAI{}, &fakeUnitOfWork{
		requests: requests, courses: courses, jobs: queue,
	}, fixedClock{now: now}, CourseGeneratorConfig{})

	started, err := service.EnqueueCourseStructure(context.Background(), validStructureParams(request.ID))
	if err != nil {
		t.Fatalf("EnqueueCourseStructure() error = %v", err)
	}
	job, err := queue.FindByID(context.Background(), started.JobID)
	if err != nil {
		t.Fatalf("find architecture job: %v", err)
	}
	if job.Kind != domain.GenerationJobKindArchitecture || job.RequestID != request.ID {
		t.Fatalf("unexpected architecture job: %+v", job)
	}
	var payload contract.ArchitectureJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.Title != "Formation Linux" {
		t.Fatalf("unexpected architecture payload: %+v, %v", payload, err)
	}

	failed := requests[request.ID]
	if err := failed.MarkFailed("lesson plan failed", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	step := stepLessonPlan
	failed.CurrentStep = &step
	requests[request.ID] = failed
	retried, err := service.EnqueueStructureRetry(context.Background(), validStructureParams(request.ID))
	if err != nil {
		t.Fatalf("EnqueueStructureRetry() error = %v", err)
	}
	if retried.JobID == started.JobID || courses.deletedRequestID != request.ID || requests[request.ID].PipelineStatus != domain.PipelineStatusRunning {
		t.Fatalf("unexpected retry state: retry=%+v deleted=%s request=%+v", retried, courses.deletedRequestID, requests[request.ID])
	}
}

func TestEnqueueTargetedContentJobsAndGetJob(t *testing.T) {
	t.Parallel()

	now := time.Now()
	request := analyzedGenerationRequest(t, now)
	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	course := domain.Course{ID: courseID, RequestID: request.ID}
	module := domain.Module{ID: moduleID, CourseID: courseID}
	lesson := domain.Lesson{ID: lessonID, ModuleID: moduleID}
	queue := newMemoryGenerationJobQueue()
	service := NewCourseGeneratorService(fakeCourseAI{}, &fakeUnitOfWork{
		requests: map[uuid.UUID]domain.GenerationRequest{request.ID: request},
		courses:  &fakeCourseRepository{courseByID: map[uuid.UUID]domain.Course{courseID: course}},
		modules:  &generationModuleRepository{module: module},
		lessons:  &generationLessonRepository{lesson: lesson},
		jobs:     queue,
	}, fixedClock{now: now}, CourseGeneratorConfig{})

	lessonStarted, err := service.EnqueueLessonContentGeneration(context.Background(), lessonID)
	if err != nil {
		t.Fatalf("EnqueueLessonContentGeneration() error = %v", err)
	}
	moduleStarted, err := service.EnqueueModuleContentGeneration(context.Background(), moduleID)
	if err != nil {
		t.Fatalf("EnqueueModuleContentGeneration() error = %v", err)
	}
	if lessonStarted.JobID == moduleStarted.JobID || queue.count() != 2 {
		t.Fatalf("unexpected targeted jobs: lesson=%+v module=%+v count=%d", lessonStarted, moduleStarted, queue.count())
	}
	job, err := service.GetGenerationJob(context.Background(), lessonStarted.JobID)
	if err != nil || job.TargetID == nil || *job.TargetID != lessonID {
		t.Fatalf("GetGenerationJob() = %+v, %v", job, err)
	}
	if _, err := service.GetGenerationJob(context.Background(), uuid.Nil); !errors.Is(err, domain.ErrBlankField) {
		t.Fatalf("nil job id error = %v", err)
	}
}

func TestGenerationCommandHelpers(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	payload, err := architectureJobPayload(validStructureParams(requestID))
	if err != nil {
		t.Fatalf("architectureJobPayload() error = %v", err)
	}
	first := deterministicJobKey("architecture", requestID, payload)
	second := deterministicJobKey("architecture", requestID, payload)
	if first != second || first == deterministicJobKey("architecture", requestID, json.RawMessage(`{"different":true}`)) {
		t.Fatal("deterministic job key is not stable or payload-sensitive")
	}
	if fullCourseIdempotencyKey("  client-key ") != "full_course:client:client-key" || fullCourseIdempotencyKey(" ") != "" {
		t.Fatal("full-course idempotency key normalization is incorrect")
	}
	if _, err := requestIDForTarget(context.Background(), fakeRepositories{}, domain.GenerationJobKindAnalysis, uuid.New()); !errors.Is(err, domain.ErrInvalidGenerationJobKind) {
		t.Fatalf("unsupported target kind error = %v", err)
	}
}

type generationModuleRepository struct {
	contract.ModuleRepository
	module domain.Module
	err    error
}

func (r *generationModuleRepository) FindModuleByID(context.Context, uuid.UUID) (domain.Module, error) {
	return r.module, r.err
}

type generationLessonRepository struct {
	contract.LessonRepository
	lesson domain.Lesson
	err    error
}

func (r *generationLessonRepository) FindLessonByID(context.Context, uuid.UUID) (domain.Lesson, error) {
	return r.lesson, r.err
}

type memoryGenerationJobQueue struct {
	mu    sync.Mutex
	jobs  map[uuid.UUID]domain.GenerationJob
	byKey map[string]uuid.UUID
}

func newMemoryGenerationJobQueue() *memoryGenerationJobQueue {
	return &memoryGenerationJobQueue{jobs: make(map[uuid.UUID]domain.GenerationJob), byKey: make(map[string]uuid.UUID)}
}

func (q *memoryGenerationJobQueue) Enqueue(_ context.Context, job domain.GenerationJob) (domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if existingID, ok := q.byKey[job.IdempotencyKey]; ok {
		return q.jobs[existingID], nil
	}
	q.jobs[job.ID] = job
	q.byKey[job.IdempotencyKey] = job.ID
	return job, nil
}

func (q *memoryGenerationJobQueue) FindByID(_ context.Context, id uuid.UUID) (domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	job, ok := q.jobs[id]
	if !ok {
		return domain.GenerationJob{}, contract.ErrGenerationJobNotFound
	}
	return job, nil
}

func (q *memoryGenerationJobQueue) FindByIdempotencyKey(_ context.Context, key string) (domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	id, ok := q.byKey[key]
	if !ok {
		return domain.GenerationJob{}, contract.ErrGenerationJobNotFound
	}
	return q.jobs[id], nil
}

func (q *memoryGenerationJobQueue) ListByRequestID(_ context.Context, requestID uuid.UUID) ([]domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	jobs := make([]domain.GenerationJob, 0)
	for _, job := range q.jobs {
		if job.RequestID == requestID {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (q *memoryGenerationJobQueue) ClaimNext(context.Context, string, time.Time) (domain.GenerationJob, error) {
	return domain.GenerationJob{}, contract.ErrGenerationJobUnavailable
}

func (q *memoryGenerationJobQueue) RenewLease(context.Context, domain.JobClaim, time.Time) error {
	return nil
}

func (q *memoryGenerationJobQueue) Complete(context.Context, domain.JobClaim, time.Time) error {
	return nil
}

func (q *memoryGenerationJobQueue) Retry(context.Context, domain.JobClaim, time.Time, error) error {
	return nil
}

func (q *memoryGenerationJobQueue) Fail(context.Context, domain.JobClaim, error, time.Time) error {
	return nil
}

func (q *memoryGenerationJobQueue) Cancel(context.Context, uuid.UUID, time.Time) error {
	return nil
}

func (q *memoryGenerationJobQueue) RequeueExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (q *memoryGenerationJobQueue) count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

var _ contract.GenerationJobQueue = (*memoryGenerationJobQueue)(nil)
