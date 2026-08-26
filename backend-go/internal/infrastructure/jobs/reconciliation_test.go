package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type reconciliationQueue struct {
	*fakeJobQueue
	mu      sync.Mutex
	failed  []domain.GenerationJob
	handled []uuid.UUID
}

func (q *reconciliationQueue) ListUnreconciledFailures(context.Context, int) ([]domain.GenerationJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]domain.GenerationJob(nil), q.failed...), nil
}

func (q *reconciliationQueue) MarkFailureHandled(_ context.Context, id uuid.UUID, _ time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handled = append(q.handled, id)
	return nil
}

type reconciliationExecutor struct {
	mu        sync.Mutex
	handled   []uuid.UUID
	handleErr error
}

func (*reconciliationExecutor) Execute(context.Context, domain.GenerationJob) error { return nil }

func (e *reconciliationExecutor) HandleTerminalFailure(_ context.Context, job domain.GenerationJob, _ error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handled = append(e.handled, job.ID)
	return e.handleErr
}

func TestWorkerReconcilesUnhandledTerminalFailure(t *testing.T) {
	now := time.Unix(1000, 0).UTC()
	job := failedTestJob(t, now)
	queue := &reconciliationQueue{fakeJobQueue: newFakeJobQueue(), failed: []domain.GenerationJob{job}}
	executor := &reconciliationExecutor{}
	pool, err := NewWorkerPool(queue, executor, &fixedClock{now: now}, DefaultWorkerConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	pool.reconcileTerminalFailures(context.Background())
	if len(executor.handled) != 1 || executor.handled[0] != job.ID {
		t.Fatalf("handled failures = %v", executor.handled)
	}
	if len(queue.handled) != 1 || queue.handled[0] != job.ID {
		t.Fatalf("reconciled markers = %v", queue.handled)
	}
}

func TestWorkerLeavesFailureUnmarkedWhenStateSynchronizationFails(t *testing.T) {
	now := time.Unix(1000, 0).UTC()
	job := failedTestJob(t, now)
	queue := &reconciliationQueue{fakeJobQueue: newFakeJobQueue(), failed: []domain.GenerationJob{job}}
	executor := &reconciliationExecutor{handleErr: errors.New("database unavailable")}
	pool, err := NewWorkerPool(queue, executor, &fixedClock{now: now}, DefaultWorkerConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	pool.reconcileTerminalFailures(context.Background())
	if len(queue.handled) != 0 {
		t.Fatalf("failure must remain available for retry, markers = %v", queue.handled)
	}
}

func failedTestJob(t *testing.T, now time.Time) domain.GenerationJob {
	t.Helper()
	job := testJob(t, now)
	message := "worker lease expired before the job completed"
	code := "max_attempts_exhausted"
	job.Status = domain.GenerationJobStatusFailed
	job.AttemptCount = job.MaxAttempts
	job.StartedAt = &now
	job.CompletedAt = &now
	job.LastErrorCode = &code
	job.LastErrorMessage = &message
	job.UpdatedAt = now
	if err := job.Validate(); err != nil {
		t.Fatalf("failed test job: %v", err)
	}
	return job
}

var _ contract.GenerationJobFailureReconciler = (*reconciliationQueue)(nil)
var _ contract.GenerationJobFailureHandler = (*reconciliationExecutor)(nil)
