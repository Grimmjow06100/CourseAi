package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

type GenerationJobStatus string

const (
	GenerationJobStatusQueued         GenerationJobStatus = "queued"
	GenerationJobStatusRunning        GenerationJobStatus = "running"
	GenerationJobStatusRetryScheduled GenerationJobStatus = "retry_scheduled"
	GenerationJobStatusCompleted      GenerationJobStatus = "completed"
	GenerationJobStatusFailed         GenerationJobStatus = "failed"
	GenerationJobStatusCancelled      GenerationJobStatus = "cancelled"
)

func ParseGenerationJobStatus(value string) (GenerationJobStatus, error) {
	status := GenerationJobStatus(strings.ToLower(strings.TrimSpace(value)))
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

func (s GenerationJobStatus) Validate() error {
	switch s {
	case GenerationJobStatusQueued,
		GenerationJobStatusRunning,
		GenerationJobStatusRetryScheduled,
		GenerationJobStatusCompleted,
		GenerationJobStatusFailed,
		GenerationJobStatusCancelled:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidGenerationJobStatus, s)
	}
}

func (s GenerationJobStatus) IsTerminal() bool {
	return s == GenerationJobStatusCompleted ||
		s == GenerationJobStatusFailed ||
		s == GenerationJobStatusCancelled
}

func (s GenerationJobStatus) CanTransitionTo(next GenerationJobStatus) bool {
	if s == next {
		return true
	}
	if s.IsTerminal() {
		return false
	}

	switch s {
	case GenerationJobStatusQueued:
		return next == GenerationJobStatusRunning || next == GenerationJobStatusCancelled
	case GenerationJobStatusRunning:
		return next == GenerationJobStatusCompleted ||
			next == GenerationJobStatusRetryScheduled ||
			next == GenerationJobStatusFailed
	case GenerationJobStatusRetryScheduled:
		return next == GenerationJobStatusRunning ||
			next == GenerationJobStatusFailed ||
			next == GenerationJobStatusCancelled
	default:
		return false
	}
}

type GenerationJobKind string

const (
	GenerationJobKindAnalysis       GenerationJobKind = "analysis"
	GenerationJobKindArchitecture   GenerationJobKind = "architecture"
	GenerationJobKindLessonPlan     GenerationJobKind = "lesson_plan"
	GenerationJobKindLessonContent  GenerationJobKind = "lesson_content"
	GenerationJobKindModuleContent  GenerationJobKind = "module_content"
	GenerationJobKindFinalizeCourse GenerationJobKind = "finalize_course"
)

func ParseGenerationJobKind(value string) (GenerationJobKind, error) {
	kind := GenerationJobKind(strings.ToLower(strings.TrimSpace(value)))
	if err := kind.Validate(); err != nil {
		return "", err
	}
	return kind, nil
}

func (k GenerationJobKind) Validate() error {
	switch k {
	case GenerationJobKindAnalysis,
		GenerationJobKindArchitecture,
		GenerationJobKindLessonPlan,
		GenerationJobKindLessonContent,
		GenerationJobKindModuleContent,
		GenerationJobKindFinalizeCourse:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidGenerationJobKind, k)
	}
}

func (k GenerationJobKind) RequiresTarget() bool {
	switch k {
	case GenerationJobKindLessonPlan,
		GenerationJobKindLessonContent,
		GenerationJobKindModuleContent,
		GenerationJobKindFinalizeCourse:
		return true
	default:
		return false
	}
}

type NewGenerationJobParams struct {
	RequestID      uuid.UUID
	ParentJobID    *uuid.UUID
	Kind           GenerationJobKind
	TargetID       *uuid.UUID
	IdempotencyKey string
	Payload        json.RawMessage
	Priority       int
	MaxAttempts    int
	AvailableAt    time.Time
}

type JobClaim struct {
	JobID        uuid.UUID
	WorkerID     string
	AttemptCount int
}

func (c JobClaim) Validate() error {
	if c.JobID == uuid.Nil {
		return fmt.Errorf("%w: generation job id", ErrBlankField)
	}
	if strings.TrimSpace(c.WorkerID) == "" {
		return fmt.Errorf("%w: worker id", ErrBlankField)
	}
	if c.AttemptCount <= 0 {
		return ErrInvalidGenerationJobAttempts
	}
	return nil
}

type GenerationJob struct {
	ID               uuid.UUID
	RequestID        uuid.UUID
	ParentJobID      *uuid.UUID
	Kind             GenerationJobKind
	Status           GenerationJobStatus
	TargetID         *uuid.UUID
	IdempotencyKey   string
	Payload          json.RawMessage
	Priority         int
	AttemptCount     int
	MaxAttempts      int
	AvailableAt      time.Time
	LockedBy         *string
	LockedUntil      *time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
	LastErrorCode    *string
	LastErrorMessage *string
	FailureHandledAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewGenerationJob(params NewGenerationJobParams) (GenerationJob, error) {
	return NewGenerationJobAt(params, time.Now())
}

func NewGenerationJobAt(params NewGenerationJobParams, now time.Time) (GenerationJob, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return GenerationJob{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	maxAttempts := params.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	availableAt := params.AvailableAt
	if availableAt.IsZero() {
		availableAt = now
	}
	payload, err := normalizeGenerationJobPayload(params.Payload)
	if err != nil {
		return GenerationJob{}, err
	}

	job := GenerationJob{
		ID:             id,
		RequestID:      params.RequestID,
		ParentJobID:    pointer.Clone(params.ParentJobID),
		Kind:           params.Kind,
		Status:         GenerationJobStatusQueued,
		TargetID:       pointer.Clone(params.TargetID),
		IdempotencyKey: strings.TrimSpace(params.IdempotencyKey),
		Payload:        payload,
		Priority:       params.Priority,
		MaxAttempts:    maxAttempts,
		AvailableAt:    availableAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := job.Validate(); err != nil {
		return GenerationJob{}, err
	}
	return job, nil
}

func (j GenerationJob) Claim() (JobClaim, error) {
	if j.Status != GenerationJobStatusRunning || j.LockedBy == nil {
		return JobClaim{}, ErrGenerationJobNotClaimed
	}
	claim := JobClaim{JobID: j.ID, WorkerID: *j.LockedBy, AttemptCount: j.AttemptCount}
	if err := claim.Validate(); err != nil {
		return JobClaim{}, err
	}
	return claim, nil
}

func (j GenerationJob) CanRetry() bool {
	return j.AttemptCount < j.MaxAttempts
}

func (j GenerationJob) Validate() error {
	if err := j.validateIdentityAndPayload(); err != nil {
		return err
	}
	if err := j.validateScheduling(); err != nil {
		return err
	}
	if err := j.validateExecutionState(); err != nil {
		return err
	}
	return j.validateOutcomeState()
}

func (j GenerationJob) validateIdentityAndPayload() error {
	if j.ID == uuid.Nil {
		return fmt.Errorf("%w: generation job id", ErrBlankField)
	}
	if j.RequestID == uuid.Nil {
		return fmt.Errorf("%w: generation request id", ErrBlankField)
	}
	if j.ParentJobID != nil {
		if *j.ParentJobID == uuid.Nil {
			return fmt.Errorf("%w: parent job id", ErrBlankField)
		}
		if *j.ParentJobID == j.ID {
			return ErrGenerationJobSelfParent
		}
	}
	if err := j.Kind.Validate(); err != nil {
		return err
	}
	if err := j.Status.Validate(); err != nil {
		return err
	}
	if j.Kind.RequiresTarget() && (j.TargetID == nil || *j.TargetID == uuid.Nil) {
		return fmt.Errorf("%w: generation job target id", ErrBlankField)
	}
	if j.TargetID != nil && *j.TargetID == uuid.Nil {
		return fmt.Errorf("%w: generation job target id", ErrBlankField)
	}
	if strings.TrimSpace(j.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key", ErrBlankField)
	}
	if _, err := normalizeGenerationJobPayload(j.Payload); err != nil {
		return err
	}
	return nil
}

func (j GenerationJob) validateScheduling() error {
	if j.AttemptCount < 0 || j.MaxAttempts <= 0 || j.AttemptCount > j.MaxAttempts {
		return ErrInvalidGenerationJobAttempts
	}
	if j.AvailableAt.IsZero() {
		return fmt.Errorf("%w: available at", ErrBlankField)
	}
	if j.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created at", ErrBlankField)
	}
	if j.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: updated at", ErrBlankField)
	}
	return nil
}

func (j GenerationJob) validateExecutionState() error {
	if j.AttemptCount == 0 {
		if j.StartedAt != nil || (j.Status != GenerationJobStatusQueued && j.Status != GenerationJobStatusCancelled) {
			return ErrInvalidGenerationJobState
		}
	} else if j.StartedAt == nil || j.Status == GenerationJobStatusQueued {
		return ErrInvalidGenerationJobState
	}

	if j.Status == GenerationJobStatusRunning {
		if j.AttemptCount == 0 || j.LockedBy == nil || strings.TrimSpace(*j.LockedBy) == "" || j.LockedUntil == nil || j.LockedUntil.IsZero() || j.StartedAt == nil {
			return ErrGenerationJobNotClaimed
		}
		if j.CompletedAt != nil {
			return ErrInvalidGenerationJobState
		}
	} else if j.LockedBy != nil || j.LockedUntil != nil {
		return ErrInvalidGenerationJobState
	}
	return nil
}

func (j GenerationJob) validateOutcomeState() error {
	if j.Status.IsTerminal() {
		if j.CompletedAt == nil {
			return fmt.Errorf("%w: completed at", ErrBlankField)
		}
	} else if j.CompletedAt != nil {
		return ErrInvalidGenerationJobState
	}

	if (j.Status == GenerationJobStatusRetryScheduled || j.Status == GenerationJobStatusFailed) &&
		(j.LastErrorMessage == nil || strings.TrimSpace(*j.LastErrorMessage) == "") {
		return fmt.Errorf("%w: last error message", ErrBlankField)
	}
	if (j.Status == GenerationJobStatusQueued || j.Status == GenerationJobStatusRunning || j.Status == GenerationJobStatusCompleted) &&
		(j.LastErrorCode != nil || j.LastErrorMessage != nil) {
		return ErrInvalidGenerationJobState
	}
	if j.FailureHandledAt != nil && j.Status != GenerationJobStatusFailed {
		return ErrInvalidGenerationJobState
	}
	return nil
}

func normalizeGenerationJobPayload(payload json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(trimmed) || trimmed[0] != '{' {
		return nil, ErrInvalidGenerationJobPayload
	}
	return jsonutil.Clone(json.RawMessage(trimmed)), nil
}
