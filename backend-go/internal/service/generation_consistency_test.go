package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func prepareConsistencyPipeline(t *testing.T) (*CourseGeneratorService, *pipelineMemoryStore, domain.GenerationJob) {
	t.Helper()
	store := newPipelineMemoryStore()
	s := NewCourseGeneratorService(&pipelineAIStub{}, pipelineMemoryUnitOfWork{store}, fixedClock{time.Now()}, CourseGeneratorConfig{})
	started, err := s.StartFullCourseGeneration(authenticatedTestContext(), contract.StartGenerationParams{Prompt: "Linux course"})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []domain.GenerationJobKind{domain.GenerationJobKindAnalysis, domain.GenerationJobKindArchitecture, domain.GenerationJobKindLessonPlan} {
		job := pipelineJobByKind(t, store, started.RequestID, kind)
		switch kind {
		case domain.GenerationJobKindAnalysis:
			err = s.runAnalysisJob(authenticatedTestContext(), job)
		case domain.GenerationJobKindArchitecture:
			err = s.runArchitectureJob(authenticatedTestContext(), job)
		case domain.GenerationJobKindLessonPlan:
			err = s.runLessonPlanJob(authenticatedTestContext(), job)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, store, pipelineJobByKind(t, store, started.RequestID, domain.GenerationJobKindLessonContent)
}

func TestTerminalCoordinationFailurePreservesCompleteContent(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	if err := s.runLessonContentJob(authenticatedTestContext(), job); err != nil {
		t.Fatal(err)
	}
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), job, errors.New("finalizer enqueue unavailable")); err != nil {
		t.Fatal(err)
	}
	course, _ := store.courseByRequestID(job.RequestID)
	request := store.requests[job.RequestID]
	if course.Status != domain.CourseStatusCompleted || request.PipelineStatus != domain.PipelineStatusCompleted || request.FailureMessage != nil {
		t.Fatalf("inconsistent completion: %s/%s", course.Status, request.PipelineStatus)
	}
	finalize := pipelineJobByKind(t, store, job.RequestID, domain.GenerationJobKindFinalizeCourse)
	if err := s.runFinalizeCourseJob(authenticatedTestContext(), finalize); err != nil {
		t.Fatalf("idempotent finalizer: %v", err)
	}
}

func TestOldFailureCannotPoisonRetriedGeneration(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), job, errors.New("old failure")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RetryFullCourseGeneration(authenticatedTestContext(), job.RequestID); err != nil {
		t.Fatal(err)
	}
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), job, errors.New("late callback")); err != nil {
		t.Fatal(err)
	}
	request := store.requests[job.RequestID]
	if request.PipelineStatus != domain.PipelineStatusRunning || request.GenerationAttempt != job.GenerationAttempt+1 {
		t.Fatal("old callback changed new attempt")
	}
}

func TestFailedRequestRejectsTargetedJobsBeforeEnqueue(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	if err := s.handleTerminalJobFailure(authenticatedTestContext(), job, errors.New("content failure")); err != nil {
		t.Fatal(err)
	}
	count := store.jobs.count()
	if _, err := s.EnqueueLessonContentGeneration(authenticatedTestContext(), *job.TargetID); !errors.Is(err, ErrGenerationNotRetryable) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if store.jobs.count() != count {
		t.Fatal("inexecutable job enqueued")
	}
}

func TestProgressAndStaleCourseCannotRegress(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	old, _ := store.courseByRequestID(job.RequestID)
	if err := s.runLessonContentJob(authenticatedTestContext(), job); err != nil {
		t.Fatal(err)
	}
	if err := s.runFinalizeCourseJob(authenticatedTestContext(), pipelineJobByKind(t, store, job.RequestID, domain.GenerationJobKindFinalizeCourse)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.updateRequestProgress(authenticatedTestContext(), job.RequestID, stepLessonPlan, 60); err != nil {
		t.Fatal(err)
	}
	if _, err := s.transitionCourse(authenticatedTestContext(), old, func(c *domain.Course) error { return c.MarkContentGenerating() }); err != nil {
		t.Fatal(err)
	}
	course, _ := store.courseByRequestID(job.RequestID)
	if course.Status != domain.CourseStatusCompleted || store.requests[job.RequestID].ProgressPercent != 100 {
		t.Fatal("terminal state regressed")
	}
}

func TestExecutorFencesOldAttemptBeforeAnyWork(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	request := store.requests[job.RequestID]
	request.GenerationAttempt++
	store.requests[request.ID] = request
	worker := "old-worker"
	until := time.Now().Add(time.Minute)
	job.Status, job.LockedBy, job.LockedUntil, job.AttemptCount = domain.GenerationJobStatusRunning, &worker, &until, 1
	started := time.Now()
	job.StartedAt = &started
	if err := NewGenerationJobExecutor(s).Execute(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	course, _ := store.courseByRequestID(request.ID)
	if course.HasCompleteContent() {
		t.Fatal("superseded worker wrote content")
	}
}

func (r pipelineRequestRepository) ListCompletionCandidates(context.Context, int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for id := range r.store.requests {
		ids = append(ids, id)
	}
	return ids, nil
}

func TestReconcilerRepairsLegacyPairWithoutNewAttempt(t *testing.T) {
	s, store, job := prepareConsistencyPipeline(t)
	if err := s.runLessonContentJob(authenticatedTestContext(), job); err != nil {
		t.Fatal(err)
	}
	request := store.requests[job.RequestID]
	if err := request.MarkFailed("legacy failure", time.Now()); err != nil {
		t.Fatal(err)
	}
	store.requests[request.ID] = request
	for id, j := range store.jobs.jobs {
		j.Status = domain.GenerationJobStatusCompleted
		store.jobs.jobs[id] = j
	}
	if err := s.ReconcileCompletedGenerations(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
	repaired := store.requests[request.ID]
	if repaired.PipelineStatus != domain.PipelineStatusCompleted || repaired.GenerationAttempt != request.GenerationAttempt {
		t.Fatal("legacy pair not repaired")
	}
}
