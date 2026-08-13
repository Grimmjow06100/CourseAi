package contract

import (
	"context"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type GenerationJobQueue interface {
	Enqueue(ctx context.Context, job domain.GenerationJob) (domain.GenerationJob, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.GenerationJob, error)
	FindByIdempotencyKey(ctx context.Context, key string) (domain.GenerationJob, error)
	ListByRequestID(ctx context.Context, requestID uuid.UUID) ([]domain.GenerationJob, error)
	ClaimNext(ctx context.Context, workerID string, lockedUntil time.Time) (domain.GenerationJob, error)
	RenewLease(ctx context.Context, claim domain.JobClaim, lockedUntil time.Time) error
	Complete(ctx context.Context, claim domain.JobClaim, completedAt time.Time) error
	Retry(ctx context.Context, claim domain.JobClaim, retryAt time.Time, cause error) error
	Fail(ctx context.Context, claim domain.JobClaim, cause error, failedAt time.Time) error
	Cancel(ctx context.Context, id uuid.UUID, cancelledAt time.Time) error
	RequeueExpired(ctx context.Context, now time.Time) (int64, error)
}

type GenerationJobExecutor interface {
	Execute(ctx context.Context, job domain.GenerationJob) error
}

// GenerationJobFailureHandler synchronizes application state after a terminal job failure.
type GenerationJobFailureHandler interface {
	HandleTerminalFailure(ctx context.Context, job domain.GenerationJob, cause error) error
}

// ArchitectureJobPayload carries user-confirmed inputs for an architecture job.
// Empty fields are completed from the persisted prompt analysis.
type ArchitectureJobPayload struct {
	Title        string                `json:"title"`
	Synopsis     string                `json:"synopsis"`
	CurrentLevel domain.Level          `json:"currentLevel"`
	TargetLevel  domain.Level          `json:"targetLevel"`
	Goals        []string              `json:"goals"`
	Language     domain.CourseLanguage `json:"language"`
}
