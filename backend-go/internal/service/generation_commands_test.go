package service

import (
	"context"
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

	started, err := service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{
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
	job, err := queue.FindByID(authenticatedTestContext(), started.JobID)
	if err != nil {
		t.Fatalf("find queued job: %v", err)
	}
	if job.Kind != domain.GenerationJobKindAnalysis || job.RequestID != started.RequestID {
		t.Fatalf("unexpected queued job: %+v", job)
	}
}

func TestStartFullCourseGenerationEnforcesPersistentAdmissionLimits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		usage contract.GenerationAdmissionUsage
		want  error
	}{
		{name: "active", usage: contract.GenerationAdmissionUsage{ActiveRequests: 1}, want: contract.ErrGenerationActiveLimitExceeded},
		{name: "daily", usage: contract.GenerationAdmissionUsage{DailyRequests: 1}, want: contract.ErrGenerationDailyLimitExceeded},
		{name: "queue", usage: contract.GenerationAdmissionUsage{PendingJobs: 1}, want: contract.ErrGenerationQueueSaturated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewCourseGeneratorService(
				fakeCourseAI{},
				&fakeUnitOfWork{requests: make(map[uuid.UUID]domain.GenerationRequest), jobs: newMemoryGenerationJobQueue(), admissionUsage: test.usage},
				fixedClock{now: time.Now()},
				CourseGeneratorConfig{MaxActivePerUser: 1, MaxDailyPerUser: 1, MaxPendingJobs: 1},
			)
			_, err := service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Linux"})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
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
	params := contract.StartGenerationParams{Prompt: "  Build a\n Docker\t course  ", IdempotencyKey: "same-command"}

	first, err := service.StartFullCourseGeneration(authenticatedTestContext(), params)
	if err != nil {
		t.Fatalf("first command: %v", err)
	}
	second, err := service.StartFullCourseGeneration(authenticatedTestContext(), params)
	if err != nil {
		t.Fatalf("idempotent command: %v", err)
	}
	if first.JobID != second.JobID || first.RequestID != second.RequestID || queue.count() != 1 || len(requests) != 1 {
		t.Fatalf("idempotency failed: first=%+v second=%+v requests=%d jobs=%d", first, second, len(requests), queue.count())
	}
	params.Prompt = "Build a Docker course"
	canonical, err := service.StartFullCourseGeneration(authenticatedTestContext(), params)
	if err != nil || canonical.JobID != first.JobID || queue.count() != 1 || len(requests) != 1 {
		t.Fatalf("normalized replay: job=%+v error=%v", canonical, err)
	}

	_, err = service.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{
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

	started, err := service.EnqueueCourseStructure(authenticatedTestContext(), validStructureParams(request.ID))
	if err != nil {
		t.Fatalf("EnqueueCourseStructure() error = %v", err)
	}
	job, err := queue.FindByID(authenticatedTestContext(), started.JobID)
	if err != nil {
		t.Fatalf("find architecture job: %v", err)
	}
	if job.Kind != domain.GenerationJobKindArchitecture || job.RequestID != request.ID {
		t.Fatalf("unexpected architecture job: %+v", job)
	}
	if string(job.Payload) != `{}` || requests[request.ID].ConfirmedBrief == nil {
		t.Fatalf("architecture should use the persisted brief: job=%+v request=%+v", job, requests[request.ID])
	}

	failed := requests[request.ID]
	if err := failed.MarkFailed("lesson plan failed", now); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	step := stepLessonPlan
	failed.CurrentStep = &step
	requests[request.ID] = failed
	retried, err := service.EnqueueStructureRetry(authenticatedTestContext(), validStructureParams(request.ID))
	if err != nil {
		t.Fatalf("EnqueueStructureRetry() error = %v", err)
	}
	if retried.JobID == started.JobID || courses.deletedRequestID != request.ID || requests[request.ID].PipelineStatus != domain.PipelineStatusRunning {
		t.Fatalf("unexpected retry state: retry=%+v deleted=%s request=%+v", retried, courses.deletedRequestID, requests[request.ID])
	}
}

func TestSubmitClarificationsPersistsBriefAndEnqueuesArchitecture(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 13, 0, 0, 0, time.UTC)
	request, err := domain.NewGenerationRequestAt("Build a Linux course", "user_test", now)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := request.MarkRunning(stepAnalysis, now); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	unknown := domain.LevelUnknown
	advanced := domain.LevelAdvanced
	language := domain.CourseLanguageEN
	title := "Linux"
	synopsis := "Linux administration"
	goal := "Administer Linux"
	if err := request.ApplyAnalysis(domain.AnalysisSummary{
		SuggestedTitle: &title, ShortSynopsis: &synopsis,
		DetectedCurrentLevel: &unknown, DetectedTargetLevel: &advanced,
		DetectedGoal: &goal, DetectedLanguage: &language,
		ClarificationQuestions: []domain.ClarificationQuestion{{
			ID: domain.ClarificationIDCurrentLevel, Question: "Current level?",
			Options: []domain.ClarificationOption{{Value: "beginner", Label: "Beginner"}, {Value: "intermediate", Label: "Intermediate"}},
		}},
	}, now); err != nil {
		t.Fatalf("apply analysis: %v", err)
	}
	if err := request.MarkAwaitingClarification(now); err != nil {
		t.Fatalf("mark awaiting clarification: %v", err)
	}

	requests := map[uuid.UUID]domain.GenerationRequest{request.ID: request}
	queue := newMemoryGenerationJobQueue()
	service := NewCourseGeneratorService(fakeCourseAI{}, &fakeUnitOfWork{requests: requests, jobs: queue}, fixedClock{now: now}, CourseGeneratorConfig{})
	params := contract.SubmitClarificationsParams{
		RequestID: request.ID,
		Answers: []domain.ClarificationAnswer{{
			QuestionID: domain.ClarificationIDCurrentLevel, SelectedValues: []string{"beginner"},
		}},
		Title: "Linux administration", Synopsis: "Production Linux", Language: domain.CourseLanguageEN,
	}
	started, err := service.SubmitClarifications(authenticatedTestContext(), params)
	if err != nil {
		t.Fatalf("SubmitClarifications() error = %v", err)
	}
	persisted := requests[request.ID]
	if persisted.PipelineStatus != domain.PipelineStatusQueued || persisted.ConfirmedBrief == nil || persisted.ConfirmedBrief.CurrentLevel != domain.LevelBeginner {
		t.Fatalf("unexpected persisted clarification state: %+v", persisted)
	}
	job, err := queue.FindByID(authenticatedTestContext(), started.JobID)
	if err != nil || job.Kind != domain.GenerationJobKindArchitecture {
		t.Fatalf("architecture job = %+v, %v", job, err)
	}

	repeated, err := service.SubmitClarifications(authenticatedTestContext(), params)
	if err != nil || repeated.JobID != started.JobID || queue.count() != 1 {
		t.Fatalf("idempotent submission = %+v, %v, jobs=%d", repeated, err, queue.count())
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

	lessonStarted, err := service.EnqueueLessonContentGeneration(authenticatedTestContext(), lessonID)
	if err != nil {
		t.Fatalf("EnqueueLessonContentGeneration() error = %v", err)
	}
	moduleStarted, err := service.EnqueueModuleContentGeneration(authenticatedTestContext(), moduleID)
	if err != nil {
		t.Fatalf("EnqueueModuleContentGeneration() error = %v", err)
	}
	if lessonStarted.JobID == moduleStarted.JobID || queue.count() != 2 {
		t.Fatalf("unexpected targeted jobs: lesson=%+v module=%+v count=%d", lessonStarted, moduleStarted, queue.count())
	}
	job, err := service.GetGenerationJob(authenticatedTestContext(), lessonStarted.JobID)
	if err != nil || job.TargetID == nil || *job.TargetID != lessonID {
		t.Fatalf("GetGenerationJob() = %+v, %v", job, err)
	}
	if _, err := service.GetGenerationJob(authenticatedTestContext(), uuid.Nil); !errors.Is(err, domain.ErrBlankField) {
		t.Fatalf("nil job id error = %v", err)
	}
}

func TestGenerationCommandHelpers(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	brief := generationBriefFromStructureParams(validStructureParams(requestID))
	first, err := architectureJobKey(requestID, 1, brief)
	if err != nil {
		t.Fatalf("architectureJobKey() error = %v", err)
	}
	second, err := architectureJobKey(requestID, 1, brief)
	if err != nil {
		t.Fatalf("architectureJobKey() second error = %v", err)
	}
	third, err := architectureJobKey(requestID, 2, brief)
	if err != nil {
		t.Fatalf("architectureJobKey() version error = %v", err)
	}
	if first != second || first == third {
		t.Fatal("deterministic job key is not stable or payload-sensitive")
	}
	if generationRequestIdempotencyKey("user_test", "  client-key ") != "analysis:clerk_user:user_test:client:client-key" || generationRequestIdempotencyKey("user_test", " ") != "" {
		t.Fatal("generation request idempotency key normalization is incorrect")
	}
	if _, err := requestIDForTarget(authenticatedTestContext(), fakeRepositories{}, domain.GenerationJobKindAnalysis, uuid.New()); !errors.Is(err, domain.ErrInvalidGenerationJobKind) {
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

func (q *memoryGenerationJobQueue) CountPendingWithAdmissionLock(context.Context) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return int64(len(q.jobs)), nil
}

func (q *memoryGenerationJobQueue) count() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.jobs)
}

var _ contract.GenerationJobQueue = (*memoryGenerationJobQueue)(nil)
