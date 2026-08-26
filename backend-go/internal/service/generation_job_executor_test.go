package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestGenerationJobExecutorDispatchesEveryJobKind(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	targetID := uuid.New()
	tests := []struct {
		name   string
		kind   domain.GenerationJobKind
		target *uuid.UUID
	}{
		{name: "analysis", kind: domain.GenerationJobKindAnalysis},
		{name: "architecture", kind: domain.GenerationJobKindArchitecture},
		{name: "lesson plan", kind: domain.GenerationJobKindLessonPlan, target: &targetID},
		{name: "lesson content", kind: domain.GenerationJobKindLessonContent, target: &targetID},
		{name: "module content", kind: domain.GenerationJobKindModuleContent, target: &targetID},
		{name: "finalize course", kind: domain.GenerationJobKindFinalizeCourse, target: &targetID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner := &recordingGenerationJobRunner{}
			executor := newGenerationJobExecutor(runner)
			job := executableGenerationJob(t, requestID, test.kind, test.target, json.RawMessage(`{}`))
			if err := executor.Execute(authenticatedTestContext(), job); err != nil {
				t.Fatalf("execute job: %v", err)
			}
			if runner.kind != test.kind || runner.requestID != requestID {
				t.Fatalf("dispatch kind=%s request=%s, want kind=%s request=%s", runner.kind, runner.requestID, test.kind, requestID)
			}
			if test.target != nil && runner.targetID != *test.target {
				t.Fatalf("target = %s, want %s", runner.targetID, *test.target)
			}
		})
	}
}

func TestGenerationJobExecutorRejectsInvalidPayloads(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	for _, payload := range []json.RawMessage{json.RawMessage(`{"unexpected":true}`), json.RawMessage(`{"title":`)} {
		job := executableGenerationJob(t, requestID, domain.GenerationJobKindArchitecture, nil, payload)
		if err := newGenerationJobExecutor(&recordingGenerationJobRunner{}).Execute(authenticatedTestContext(), job); !errors.Is(err, domain.ErrInvalidGenerationJobPayload) {
			t.Fatalf("error = %v, want ErrInvalidGenerationJobPayload", err)
		}
	}
}

func TestGenerationJobExecutorRequiresClaimedJob(t *testing.T) {
	t.Parallel()

	now := time.Now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID: uuid.New(), Kind: domain.GenerationJobKindAnalysis,
		IdempotencyKey: "analysis:not-claimed", Payload: json.RawMessage(`{}`), AvailableAt: now,
	}, now)
	if err != nil {
		t.Fatalf("new job: %v", err)
	}
	if err := newGenerationJobExecutor(&recordingGenerationJobRunner{}).Execute(authenticatedTestContext(), job); !errors.Is(err, ErrGenerationJobNotExecutable) {
		t.Fatalf("error = %v, want ErrGenerationJobNotExecutable", err)
	}
}

func TestGenerationJobExecutorRequiresRunner(t *testing.T) {
	t.Parallel()

	job := executableGenerationJob(t, uuid.New(), domain.GenerationJobKindAnalysis, nil, json.RawMessage(`{}`))
	if err := NewGenerationJobExecutor(nil).Execute(authenticatedTestContext(), job); !errors.Is(err, ErrGenerationJobExecutorDependency) {
		t.Fatalf("error = %v, want ErrGenerationJobExecutorDependency", err)
	}
}

func executableGenerationJob(t *testing.T, requestID uuid.UUID, kind domain.GenerationJobKind, targetID *uuid.UUID, payload json.RawMessage) domain.GenerationJob {
	t.Helper()
	now := time.Now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		RequestID: requestID, Kind: kind, TargetID: targetID,
		IdempotencyKey: string(kind) + ":" + uuid.NewString(), Payload: payload, AvailableAt: now,
	}, now)
	if err != nil {
		if !errors.Is(err, domain.ErrInvalidGenerationJobPayload) {
			t.Fatalf("new job: %v", err)
		}
		job = domain.GenerationJob{
			ID: uuid.New(), RequestID: requestID, Kind: kind, TargetID: targetID,
			IdempotencyKey: string(kind) + ":" + uuid.NewString(), Payload: payload,
			MaxAttempts: 3, AvailableAt: now, CreatedAt: now, UpdatedAt: now,
		}
	}
	workerID := "test-worker"
	lockedUntil := now.Add(time.Minute)
	job.Status = domain.GenerationJobStatusRunning
	job.AttemptCount = 1
	job.LockedBy = &workerID
	job.LockedUntil = &lockedUntil
	job.StartedAt = &now
	return job
}

type recordingGenerationJobRunner struct {
	kind      domain.GenerationJobKind
	requestID uuid.UUID
	targetID  uuid.UUID
}

func (r *recordingGenerationJobRunner) record(job domain.GenerationJob) {
	r.kind = job.Kind
	r.requestID = job.RequestID
	if job.TargetID != nil {
		r.targetID = *job.TargetID
	}
}

func (r *recordingGenerationJobRunner) runAnalysisJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) runArchitectureJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) runLessonPlanJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) runLessonContentJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) runModuleContentJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) runFinalizeCourseJob(_ context.Context, job domain.GenerationJob) error {
	r.record(job)
	return nil
}
func (r *recordingGenerationJobRunner) handleTerminalJobFailure(_ context.Context, job domain.GenerationJob, _ error) error {
	r.requestID = job.RequestID
	return nil
}
