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
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/correlation"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
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
	if p.config.RetentionEnabled {
		if _, ok := p.queue.(contract.GenerationJobRetention); ok {
			workers.Add(1)
			go func() {
				defer workers.Done()
				p.runRetention(runCtx)
			}()
		}
	}
	if _, ok := p.queue.(contract.GenerationQueueMetricsProvider); ok {
		workers.Add(1)
		go func() {
			defer workers.Done()
			p.runMetrics(runCtx)
		}()
	}

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
	workerStartedAt := time.Now()
	workerLogger := p.workerLogger(workerID)
	workerLogger.InfoContext(ctx, "generation worker started",
		"event", "generation_worker_started",
		"started_at", workerStartedAt.UTC(),
		"poll_interval_ms", p.config.PollInterval.Milliseconds(),
		"lease_duration_ms", p.config.LeaseDuration.Milliseconds(),
		"heartbeat_interval_ms", p.config.HeartbeatInterval.Milliseconds(),
		"job_timeout_ms", p.config.JobTimeout.Milliseconds(),
	)
	defer func() {
		finishedAt := time.Now()
		stopReason := "worker_loop_completed"
		if ctx.Err() != nil {
			stopReason = ctx.Err().Error()
		}
		workerLogger.Info("generation worker stopped",
			"event", "generation_worker_stopped",
			"outcome", "stopped",
			"stop_reason", stopReason,
			"finished_at", finishedAt.UTC(),
			"duration_ms", finishedAt.Sub(workerStartedAt).Milliseconds(),
		)
	}()

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
			workerLogger.ErrorContext(ctx, "claim generation job failed",
				appendLogArgs(
					[]any{"event", "generation_job_claim_failed", "outcome", "error"},
					errorLogArgs(err),
				)...,
			)
			if !waitForContext(ctx, p.config.PollInterval) {
				return
			}
		}
	}
}

func (p *WorkerPool) executeClaimedJob(processCtx context.Context, job domain.GenerationJob) {
	claim, err := job.Claim()
	if err != nil {
		p.logger.ErrorContext(processCtx, "invalid claimed generation job",
			appendLogArgs(
				[]any{
					"event", "generation_job_claim_invalid",
					"component", workerLogComponent,
					"worker_id", optionalStringLogValue(job.LockedBy),
					"job_id", job.ID.String(),
					"request_id", job.RequestID.String(),
					"parent_job_id", optionalUUIDLogValue(job.ParentJobID),
					"job_kind", string(job.Kind),
					"target_id", optionalUUIDLogValue(job.TargetID),
					"outcome", "error",
				},
				errorLogArgs(err),
			)...,
		)
		return
	}
	jobLogger := p.claimedJobLogger(job, claim)
	executionStartedAt := time.Now()
	jobLogger.InfoContext(processCtx, "generation job started",
		"event", "generation_job_started",
		"outcome", "running",
		"job_status", string(domain.GenerationJobStatusRunning),
		"started_at", executionStartedAt.UTC(),
		"claim_started_at", optionalTimeLogValue(job.StartedAt),
		"locked_until", optionalTimeLogValue(job.LockedUntil),
		"available_at", job.AvailableAt.UTC(),
		"queue_delay_ms", nonNegativeDurationMilliseconds(executionStartedAt.Sub(job.AvailableAt)),
		"job_timeout_ms", p.config.JobTimeout.Milliseconds(),
	)

	jobCtx, cancelJob := context.WithTimeout(processCtx, p.config.JobTimeout)
	jobCtx = correlation.WithJob(jobCtx, correlation.Job{
		JobID: job.ID.String(), RequestID: job.RequestID.String(), Kind: string(job.Kind), Attempt: claim.AttemptCount,
	})
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
		finishedAt := time.Now()
		jobLogger.Info("generation job interrupted by worker shutdown",
			appendLogArgs(
				[]any{"event", "generation_job_interrupted", "reason", "worker_shutdown"},
				jobFinishedLogArgs(executionStartedAt, finishedAt, "interrupted", domain.GenerationJobStatusRunning),
			)...,
		)
		return
	}
	if errors.Is(heartbeatErr, contract.ErrGenerationJobClaimLost) {
		finishedAt := time.Now()
		jobLogger.Warn("generation job claim lost",
			appendLogArgs(
				[]any{"event", "generation_job_claim_lost"},
				jobFinishedLogArgs(executionStartedAt, finishedAt, "claim_lost", domain.GenerationJobStatusRunning),
				errorLogArgs(heartbeatErr),
			)...,
		)
		return
	}
	if heartbeatErr != nil {
		jobLogger.Warn("generation job heartbeat failed",
			appendLogArgs(
				[]any{"event", "generation_job_heartbeat_failed", "elapsed_ms", time.Since(executionStartedAt).Milliseconds()},
				errorLogArgs(heartbeatErr),
			)...,
		)
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
		completedAt := p.clock.Now()
		if err := p.queue.Complete(cleanupCtx, claim, completedAt); err != nil {
			finishedAt := time.Now()
			jobLogger.ErrorContext(cleanupCtx, "persist generation job completion failed",
				appendLogArgs(
					[]any{"event", "generation_job_completion_failed", "completed_at", completedAt.UTC()},
					jobFinishedLogArgs(executionStartedAt, finishedAt, "completion_persistence_failed", domain.GenerationJobStatusRunning),
					errorLogArgs(err),
				)...,
			)
			return
		}
		finishedAt := time.Now()
		jobLogger.InfoContext(cleanupCtx, "generation job completed",
			appendLogArgs(
				[]any{"event", "generation_job_completed", "completed_at", completedAt.UTC()},
				jobFinishedLogArgs(executionStartedAt, finishedAt, "success", domain.GenerationJobStatusCompleted),
			)...,
		)
		return
	}

	decision := p.retryPolicy.Decide(job, executionErr)
	if decision.Retry {
		if err := p.queue.Retry(cleanupCtx, claim, decision.AvailableAt, executionErr); err != nil {
			finishedAt := time.Now()
			jobLogger.ErrorContext(cleanupCtx, "persist generation job retry failed",
				appendLogArgs(
					[]any{
						"event", "generation_job_retry_failed",
						"retry_reason", decision.Reason,
						"retry_at", decision.AvailableAt.UTC(),
						"retry_delay_ms", decision.Delay.Milliseconds(),
						"execution_error", executionErr,
					},
					jobFinishedLogArgs(executionStartedAt, finishedAt, "retry_persistence_failed", domain.GenerationJobStatusRunning),
					errorLogArgs(err),
				)...,
			)
			return
		}
		finishedAt := time.Now()
		jobLogger.Warn("generation job retry scheduled",
			appendLogArgs(
				[]any{
					"event", "generation_job_retry_scheduled",
					"retry_reason", decision.Reason,
					"retry_at", decision.AvailableAt.UTC(),
					"retry_delay_ms", decision.Delay.Milliseconds(),
				},
				jobFinishedLogArgs(executionStartedAt, finishedAt, "retry_scheduled", domain.GenerationJobStatusRetryScheduled),
				errorLogArgs(executionErr),
			)...,
		)
		return
	}

	if err := p.queue.Fail(cleanupCtx, claim, executionErr, p.clock.Now()); err != nil {
		finishedAt := time.Now()
		jobLogger.ErrorContext(cleanupCtx, "persist terminal generation job failure failed",
			appendLogArgs(
				[]any{
					"event", "generation_job_failure_persistence_failed",
					"failure_reason", decision.Reason,
					"execution_error", executionErr,
				},
				jobFinishedLogArgs(executionStartedAt, finishedAt, "failure_persistence_failed", domain.GenerationJobStatusRunning),
				errorLogArgs(err),
			)...,
		)
		return
	}
	if failureHandler, ok := p.executor.(contract.GenerationJobFailureHandler); ok {
		if err := failureHandler.HandleTerminalFailure(cleanupCtx, job, executionErr); err != nil {
			jobLogger.ErrorContext(cleanupCtx, "synchronize terminal generation job failure",
				appendLogArgs(
					[]any{"event", "generation_job_failure_sync_failed", "execution_error", executionErr},
					errorLogArgs(err),
				)...,
			)
		} else if reconciler, ok := p.queue.(contract.GenerationJobFailureReconciler); ok {
			if err := reconciler.MarkFailureHandled(cleanupCtx, job.ID, p.clock.Now()); err != nil {
				jobLogger.ErrorContext(cleanupCtx, "mark terminal generation job failure handled",
					appendLogArgs([]any{"event", "generation_job_failure_mark_failed"}, errorLogArgs(err))...,
				)
			}
		}
	}
	finishedAt := time.Now()
	jobLogger.Error("generation job failed",
		appendLogArgs(
			[]any{"event", "generation_job_failed", "failure_reason", decision.Reason},
			jobFinishedLogArgs(executionStartedAt, finishedAt, "failed", domain.GenerationJobStatusFailed),
			errorLogArgs(executionErr),
		)...,
	)
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
	return errtrace.Capture(p.executor.Execute(ctx, job))
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
