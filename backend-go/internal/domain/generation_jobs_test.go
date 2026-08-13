package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewGenerationJobAppliesDefaults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
	job, err := NewGenerationJobAt(NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           GenerationJobKindAnalysis,
		IdempotencyKey: " request:analysis ",
	}, now)
	if err != nil {
		t.Fatalf("new generation job: %v", err)
	}

	if job.Status != GenerationJobStatusQueued {
		t.Fatalf("status = %q, want %q", job.Status, GenerationJobStatusQueued)
	}
	if job.MaxAttempts != 3 {
		t.Fatalf("max attempts = %d, want 3", job.MaxAttempts)
	}
	if job.IdempotencyKey != "request:analysis" {
		t.Fatalf("idempotency key = %q", job.IdempotencyKey)
	}
	if string(job.Payload) != "{}" {
		t.Fatalf("payload = %s, want {}", job.Payload)
	}
	if !job.AvailableAt.Equal(now) || !job.CreatedAt.Equal(now) || !job.UpdatedAt.Equal(now) {
		t.Fatal("constructor did not use the supplied clock value")
	}
}

func TestNewGenerationJobRequiresTargetForTargetedKinds(t *testing.T) {
	t.Parallel()

	_, err := NewGenerationJobAt(NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           GenerationJobKindLessonContent,
		IdempotencyKey: "lesson-content",
	}, time.Now())
	if !errors.Is(err, ErrBlankField) {
		t.Fatalf("error = %v, want ErrBlankField", err)
	}
}

func TestNewGenerationJobRejectsNonObjectPayload(t *testing.T) {
	t.Parallel()

	_, err := NewGenerationJobAt(NewGenerationJobParams{
		RequestID:      uuid.New(),
		Kind:           GenerationJobKindAnalysis,
		IdempotencyKey: "analysis",
		Payload:        json.RawMessage(`["invalid"]`),
	}, time.Now())
	if !errors.Is(err, ErrInvalidGenerationJobPayload) {
		t.Fatalf("error = %v, want ErrInvalidGenerationJobPayload", err)
	}
}

func TestGenerationJobStatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current GenerationJobStatus
		next    GenerationJobStatus
		want    bool
	}{
		{name: "queued to running", current: GenerationJobStatusQueued, next: GenerationJobStatusRunning, want: true},
		{name: "running to retry", current: GenerationJobStatusRunning, next: GenerationJobStatusRetryScheduled, want: true},
		{name: "retry to running", current: GenerationJobStatusRetryScheduled, next: GenerationJobStatusRunning, want: true},
		{name: "running to completed", current: GenerationJobStatusRunning, next: GenerationJobStatusCompleted, want: true},
		{name: "queued cannot complete", current: GenerationJobStatusQueued, next: GenerationJobStatusCompleted, want: false},
		{name: "completed is terminal", current: GenerationJobStatusCompleted, next: GenerationJobStatusRunning, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.current.CanTransitionTo(test.next); got != test.want {
				t.Fatalf("CanTransitionTo(%q) = %t, want %t", test.next, got, test.want)
			}
		})
	}
}

func TestGenerationJobClaimUsesFencingAttempt(t *testing.T) {
	t.Parallel()

	now := time.Now()
	workerID := "worker-1"
	lockedUntil := now.Add(time.Minute)
	startedAt := now
	job := GenerationJob{
		ID:             uuid.New(),
		RequestID:      uuid.New(),
		Kind:           GenerationJobKindAnalysis,
		Status:         GenerationJobStatusRunning,
		IdempotencyKey: "analysis",
		Payload:        json.RawMessage(`{}`),
		AttemptCount:   2,
		MaxAttempts:    3,
		AvailableAt:    now,
		LockedBy:       &workerID,
		LockedUntil:    &lockedUntil,
		StartedAt:      &startedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	claim, err := job.Claim()
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claim.JobID != job.ID || claim.WorkerID != workerID || claim.AttemptCount != 2 {
		t.Fatalf("unexpected claim: %+v", claim)
	}
}

func TestGenerationJobParsersAndRetryBudget(t *testing.T) {
	t.Parallel()

	status, err := ParseGenerationJobStatus(" RETRY_SCHEDULED ")
	if err != nil || status != GenerationJobStatusRetryScheduled {
		t.Fatalf("ParseGenerationJobStatus() = %q, %v", status, err)
	}
	if _, err := ParseGenerationJobStatus("paused"); !errors.Is(err, ErrInvalidGenerationJobStatus) {
		t.Fatalf("invalid status error = %v", err)
	}
	kind, err := ParseGenerationJobKind(" LESSON_CONTENT ")
	if err != nil || kind != GenerationJobKindLessonContent || !kind.RequiresTarget() {
		t.Fatalf("ParseGenerationJobKind() = %q, %v", kind, err)
	}
	if _, err := ParseGenerationJobKind("video"); !errors.Is(err, ErrInvalidGenerationJobKind) {
		t.Fatalf("invalid kind error = %v", err)
	}

	job, err := NewGenerationJob(NewGenerationJobParams{
		RequestID: uuid.New(), Kind: GenerationJobKindAnalysis, IdempotencyKey: "analysis",
	})
	if err != nil {
		t.Fatalf("NewGenerationJob() error = %v", err)
	}
	job.AttemptCount = 2
	job.MaxAttempts = 3
	if !job.CanRetry() {
		t.Fatal("job should have one retry remaining")
	}
	job.AttemptCount = 3
	if job.CanRetry() {
		t.Fatal("job should have exhausted its retry budget")
	}
}

func TestJobClaimValidate(t *testing.T) {
	t.Parallel()

	if err := (JobClaim{JobID: uuid.New(), WorkerID: "worker", AttemptCount: 1}).Validate(); err != nil {
		t.Fatalf("valid claim rejected: %v", err)
	}
	if err := (JobClaim{WorkerID: "worker", AttemptCount: 1}).Validate(); !errors.Is(err, ErrBlankField) {
		t.Fatalf("nil job id error = %v", err)
	}
	if err := (JobClaim{JobID: uuid.New(), WorkerID: " ", AttemptCount: 1}).Validate(); !errors.Is(err, ErrBlankField) {
		t.Fatalf("blank worker error = %v", err)
	}
	if err := (JobClaim{JobID: uuid.New(), WorkerID: "worker"}).Validate(); !errors.Is(err, ErrInvalidGenerationJobAttempts) {
		t.Fatalf("invalid attempts error = %v", err)
	}
}
