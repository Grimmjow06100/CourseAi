package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestWorkerPoolBoundsConcurrencyAndCompletesJobs(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now), testJob(t, now), testJob(t, now), testJob(t, now))
	var active atomic.Int32
	var maximum atomic.Int32
	executor := executorFunc(func(context.Context, domain.GenerationJob) error {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		return nil
	})
	cfg := testWorkerConfig()
	cfg.Concurrency = 2
	pool := mustWorkerPool(t, queue, executor, &fixedClock{now: now}, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()

	for index := 0; index < 4; index++ {
		select {
		case <-queue.completedChan:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for completed jobs")
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run worker pool: %v", err)
	}
	if got := maximum.Load(); got != 2 {
		t.Fatalf("maximum concurrency = %d, want 2", got)
	}
	_, completed, _, _, requeues := queue.snapshot()
	if completed != 4 || requeues == 0 {
		t.Fatalf("completed=%d requeues=%d", completed, requeues)
	}
}

func TestWorkerPoolSchedulesRetryForTransientError(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now))
	pool := mustWorkerPool(t, queue, executorFunc(func(context.Context, domain.GenerationJob) error {
		return context.DeadlineExceeded
	}), &fixedClock{now: now}, testWorkerConfig())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()

	select {
	case retry := <-queue.retryChan:
		if retry.retryAt.Before(now) || !errors.Is(retry.cause, context.DeadlineExceeded) {
			t.Fatalf("unexpected retry: %+v", retry)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for retry")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run worker pool: %v", err)
	}
}

func TestWorkerPoolFailsPermanentError(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now))
	pool := mustWorkerPool(t, queue, executorFunc(func(context.Context, domain.GenerationJob) error {
		return domain.ErrBlankField
	}), &fixedClock{now: now}, testWorkerConfig())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()

	select {
	case failure := <-queue.failChan:
		if !errors.Is(failure.cause, domain.ErrBlankField) {
			t.Fatalf("unexpected failure: %+v", failure)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for failure")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run worker pool: %v", err)
	}
}

func TestWorkerPoolSynchronizesTerminalFailureWithApplication(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now))
	executor := &failureAwareExecutor{handled: make(chan domain.GenerationJob, 1)}
	pool := mustWorkerPool(t, queue, executor, &fixedClock{now: now}, testWorkerConfig())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()

	select {
	case job := <-executor.handled:
		if job.RequestID == uuid.Nil {
			t.Fatal("terminal failure handler received an empty request id")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal failure synchronization")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run worker pool: %v", err)
	}
}

func TestWorkerPoolStopsObsoleteExecutionAfterClaimLoss(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now))
	queue.renewErr = contract.ErrGenerationJobClaimLost
	executorStopped := make(chan struct{})
	pool := mustWorkerPool(t, queue, executorFunc(func(ctx context.Context, _ domain.GenerationJob) error {
		<-ctx.Done()
		close(executorStopped)
		return ctx.Err()
	}), &fixedClock{now: now}, testWorkerConfig())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()

	select {
	case <-executorStopped:
	case <-time.After(time.Second):
		t.Fatal("heartbeat did not stop obsolete execution")
	}
	time.Sleep(20 * time.Millisecond)
	_, completed, retries, failures, _ := queue.snapshot()
	if completed != 0 || retries != 0 || failures != 0 {
		t.Fatalf("obsolete worker finalized job: completed=%d retries=%d failures=%d", completed, retries, failures)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("run worker pool: %v", err)
	}
}

func TestWorkerPoolReturnsShutdownTimeoutWhenExecutorIgnoresContext(t *testing.T) {
	t.Parallel()
	now := time.Now()
	queue := newFakeJobQueue(testJob(t, now))
	started := make(chan struct{})
	release := make(chan struct{})
	cfg := testWorkerConfig()
	cfg.ShutdownTimeout = 30 * time.Millisecond
	cfg.CleanupTimeout = 10 * time.Millisecond
	pool := mustWorkerPool(t, queue, executorFunc(func(context.Context, domain.GenerationJob) error {
		close(started)
		<-release
		return nil
	}), &fixedClock{now: now}, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pool.Run(ctx) }()
	<-started
	cancel()

	if err := <-done; !errors.Is(err, ErrWorkerShutdownTimeout) {
		t.Fatalf("error = %v, want ErrWorkerShutdownTimeout", err)
	}
	close(release)
}

func testWorkerConfig() WorkerConfig {
	cfg := DefaultWorkerConfig()
	cfg.PollInterval = 5 * time.Millisecond
	cfg.LeaseDuration = 100 * time.Millisecond
	cfg.HeartbeatInterval = 10 * time.Millisecond
	cfg.JobTimeout = 500 * time.Millisecond
	cfg.ShutdownTimeout = 200 * time.Millisecond
	cfg.CleanupTimeout = 20 * time.Millisecond
	cfg.ReaperInterval = 20 * time.Millisecond
	cfg.RetryBaseDelay = 10 * time.Millisecond
	cfg.RetryMaxDelay = 100 * time.Millisecond
	cfg.RetryJitterFraction = 0
	return cfg
}

func mustWorkerPool(t *testing.T, queue contract.GenerationJobQueue, executor contract.GenerationJobExecutor, clock contract.Clock, cfg WorkerConfig) *WorkerPool {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pool, err := NewWorkerPool(queue, executor, clock, cfg, logger)
	if err != nil {
		t.Fatalf("new worker pool: %v", err)
	}
	return pool
}

type failureAwareExecutor struct {
	handled chan domain.GenerationJob
}

func (e *failureAwareExecutor) Execute(context.Context, domain.GenerationJob) error {
	return domain.ErrBlankField
}

func (e *failureAwareExecutor) HandleTerminalFailure(_ context.Context, job domain.GenerationJob, cause error) error {
	if !errors.Is(cause, domain.ErrBlankField) {
		return errors.New("unexpected terminal failure cause")
	}
	e.handled <- job
	return nil
}
