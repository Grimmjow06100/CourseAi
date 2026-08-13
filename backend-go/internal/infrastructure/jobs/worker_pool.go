package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

var (
	ErrMissingWorkerQueue    = errors.New("generation worker queue is missing")
	ErrMissingWorkerExecutor = errors.New("generation job executor is missing")
	ErrMissingWorkerClock    = errors.New("generation worker clock is missing")
	ErrWorkerShutdownTimeout = errors.New("generation worker shutdown timed out")
	ErrJobExecutionPanic     = errors.New("generation job executor panicked")
)

type WorkerPool struct {
	queue       contract.GenerationJobQueue
	executor    contract.GenerationJobExecutor
	clock       contract.Clock
	config      WorkerConfig
	logger      *slog.Logger
	heartbeat   *Heartbeat
	retryPolicy *RetryPolicy
}

func NewWorkerPool(
	queue contract.GenerationJobQueue,
	executor contract.GenerationJobExecutor,
	clock contract.Clock,
	config WorkerConfig,
	logger *slog.Logger,
) (*WorkerPool, error) {
	if queue == nil {
		return nil, ErrMissingWorkerQueue
	}
	if executor == nil {
		return nil, ErrMissingWorkerExecutor
	}
	if clock == nil {
		return nil, ErrMissingWorkerClock
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	heartbeat, err := NewHeartbeat(queue, clock, config.HeartbeatInterval, config.LeaseDuration)
	if err != nil {
		return nil, err
	}
	retryPolicy, err := NewRetryPolicy(clock, config)
	if err != nil {
		return nil, err
	}
	return &WorkerPool{
		queue:       queue,
		executor:    executor,
		clock:       clock,
		config:      config,
		logger:      logger,
		heartbeat:   heartbeat,
		retryPolicy: retryPolicy,
	}, nil
}

// Run starts a fixed number of workers and a lease reaper until ctx is cancelled.
func (p *WorkerPool) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("generation worker context is nil")
	}
	if !p.config.Enabled {
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var workers sync.WaitGroup
	for index := 0; index < p.config.Concurrency; index++ {
		workerID := p.newWorkerID(index)
		workers.Add(1)
		go func(id string) {
			defer workers.Done()
			p.runWorker(runCtx, id)
		}(workerID)
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		p.runReaper(runCtx)
	}()

	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		cancel()
	}

	timer := time.NewTimer(p.config.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		return ErrWorkerShutdownTimeout
	}
}

func (p *WorkerPool) runWorker(ctx context.Context, workerID string) {
	for {
		if ctx.Err() != nil {
			return
		}

		job, err := p.queue.ClaimNext(ctx, workerID, p.clock.Now().Add(p.config.LeaseDuration))
		switch {
		case err == nil:
			p.executeClaimedJob(ctx, job)
		case errors.Is(err, contract.ErrGenerationJobUnavailable):
			if !waitForContext(ctx, p.config.PollInterval) {
				return
			}
		default:
			p.logger.ErrorContext(ctx, "claim generation job", "worker_id", workerID, "error", err)
			if !waitForContext(ctx, p.config.PollInterval) {
				return
			}
		}
	}
}

func (p *WorkerPool) executeClaimedJob(processCtx context.Context, job domain.GenerationJob) {
	claim, err := job.Claim()
	if err != nil {
		p.logger.ErrorContext(processCtx, "invalid claimed generation job", "job_id", job.ID, "error", err)
		return
	}

	jobCtx, cancelJob := context.WithTimeout(processCtx, p.config.JobTimeout)
	heartbeatResult := make(chan error, 1)
	go func() {
		heartbeatErr := p.heartbeat.Run(jobCtx, claim)
		if heartbeatErr != nil {
			cancelJob()
		}
		heartbeatResult <- heartbeatErr
	}()

	executionErr := p.executeSafely(jobCtx, job)
	jobContextErr := jobCtx.Err()
	cancelJob()
	heartbeatErr := <-heartbeatResult

	if processCtx.Err() != nil {
		p.logger.Info("generation job interrupted by worker shutdown", "job_id", job.ID, "worker_id", claim.WorkerID)
		return
	}
	if errors.Is(heartbeatErr, contract.ErrGenerationJobClaimLost) {
		p.logger.Warn("generation job claim lost", "job_id", job.ID, "worker_id", claim.WorkerID, "attempt", claim.AttemptCount)
		return
	}
	if heartbeatErr != nil {
		if executionErr == nil || errors.Is(executionErr, context.Canceled) {
			executionErr = heartbeatErr
		} else {
			executionErr = errors.Join(executionErr, heartbeatErr)
		}
	}
	if errors.Is(jobContextErr, context.DeadlineExceeded) {
		timeoutErr := fmt.Errorf("generation job execution timeout: %w", context.DeadlineExceeded)
		if executionErr == nil || errors.Is(executionErr, context.Canceled) {
			executionErr = timeoutErr
		} else {
			executionErr = errors.Join(executionErr, timeoutErr)
		}
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(processCtx), p.config.CleanupTimeout)
	defer cleanupCancel()

	if executionErr == nil {
		if err := p.queue.Complete(cleanupCtx, claim, p.clock.Now()); err != nil {
			p.logger.ErrorContext(cleanupCtx, "complete generation job", "job_id", job.ID, "error", err)
		}
		return
	}

	decision := p.retryPolicy.Decide(job, executionErr)
	if decision.Retry {
		if err := p.queue.Retry(cleanupCtx, claim, decision.AvailableAt, executionErr); err != nil {
			p.logger.ErrorContext(cleanupCtx, "schedule generation job retry", "job_id", job.ID, "error", err)
			return
		}
		p.logger.Warn("generation job retry scheduled", "job_id", job.ID, "attempt", claim.AttemptCount, "delay", decision.Delay, "reason", decision.Reason)
		return
	}

	if err := p.queue.Fail(cleanupCtx, claim, executionErr, p.clock.Now()); err != nil {
		p.logger.ErrorContext(cleanupCtx, "fail generation job", "job_id", job.ID, "error", err)
		return
	}
	if failureHandler, ok := p.executor.(contract.GenerationJobFailureHandler); ok {
		if err := failureHandler.HandleTerminalFailure(cleanupCtx, job, executionErr); err != nil {
			p.logger.ErrorContext(cleanupCtx, "synchronize terminal generation job failure", "job_id", job.ID, "request_id", job.RequestID, "error", err)
		}
	}
	p.logger.Error("generation job failed", "job_id", job.ID, "attempt", claim.AttemptCount, "reason", decision.Reason, "error", executionErr)
}

func (p *WorkerPool) executeSafely(ctx context.Context, job domain.GenerationJob) (executionErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			executionErr = permanentJobError{
				code: "executor_panic",
				err:  fmt.Errorf("%w: %v\n%s", ErrJobExecutionPanic, recovered, debug.Stack()),
			}
		}
	}()
	return p.executor.Execute(ctx, job)
}

func (p *WorkerPool) runReaper(ctx context.Context) {
	p.requeueExpired(ctx)
	ticker := time.NewTicker(p.config.ReaperInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.requeueExpired(ctx)
		}
	}
}

func (p *WorkerPool) requeueExpired(ctx context.Context) {
	count, err := p.queue.RequeueExpired(ctx, p.clock.Now())
	if err != nil {
		if ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "requeue expired generation jobs", "error", err)
		}
		return
	}
	if count > 0 {
		p.logger.Warn("expired generation job leases requeued", "count", count)
	}
}

func (p *WorkerPool) newWorkerID(index int) string {
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = "unknown-host"
	}
	return fmt.Sprintf("%s:%s:%d:%d:%s", p.config.WorkerIDPrefix, hostname, os.Getpid(), index, uuid.NewString())
}

func waitForContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type permanentJobError struct {
	code string
	err  error
}

func (e permanentJobError) Error() string {
	return e.err.Error()
}

func (e permanentJobError) Unwrap() error {
	return e.err
}

func (e permanentJobError) ErrorCode() string {
	return e.code
}

func (e permanentJobError) Retryable() bool {
	return false
}
