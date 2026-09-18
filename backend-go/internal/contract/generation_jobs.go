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
	CountPendingWithAdmissionLock(ctx context.Context) (int64, error)
}

type GenerationJobExecutor interface {
	Execute(ctx context.Context, job domain.GenerationJob) error
}

// GenerationJobClaimGuard locks and checks a live claim within the business transaction.
type GenerationJobClaimGuard interface {
	LockClaim(ctx context.Context, claim domain.JobClaim) error
}

// GenerationSuccessReconciler completes persisted content whose finalizer was interrupted.
type GenerationSuccessReconciler interface {
	ReconcileCompletedGenerations(ctx context.Context, limit int) error
}

// GenerationJobFailureHandler synchronizes application state after a terminal job failure.
type GenerationJobFailureHandler interface {
	HandleTerminalFailure(ctx context.Context, job domain.GenerationJob, cause error) error
}

// GenerationJobFailureReconciler repairs terminal job failures whose request/course transition was interrupted.
type GenerationJobFailureReconciler interface {
	ListUnreconciledFailures(ctx context.Context, limit int) ([]domain.GenerationJob, error)
	MarkFailureHandled(ctx context.Context, jobID uuid.UUID, handledAt time.Time) error
}

type GenerationJobRetention interface {
	PurgeTerminalBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
	PurgeRawOutputsBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}

type GenerationQueueMetrics struct {
	Queued               int64
	RetryScheduled       int64
	Running              int64
	Completed            int64
	Failed               int64
	Cancelled            int64
	ExpiredLeases        int64
	UnreconciledFailures int64
}

type GenerationQueueMetricsProvider interface {
	GetQueueMetrics(ctx context.Context) (GenerationQueueMetrics, error)
}
