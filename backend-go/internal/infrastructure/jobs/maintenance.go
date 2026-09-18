package jobs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

func (p *WorkerPool) runReaper(ctx context.Context) {
	p.requeueExpired(ctx)
	p.reconcileTerminalFailures(ctx)
	ticker := time.NewTicker(p.config.ReaperInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.requeueExpired(ctx)
			p.reconcileTerminalFailures(ctx)
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
		p.logger.Warn("expired generation job leases processed", "event", "generation_job_leases_reaped", "count", count)
	}
}

func (p *WorkerPool) reconcileTerminalFailures(ctx context.Context) {
	defer func() {
		if reconciler, ok := p.executor.(contract.GenerationSuccessReconciler); ok {
			if err := reconciler.ReconcileCompletedGenerations(ctx, p.config.ReconciliationBatch); err != nil {
				p.logger.ErrorContext(ctx, "reconcile completed generation content", "event", "generation_completion_reconciliation_failed", "error", err)
			}
		}
	}()
	reconciler, ok := p.queue.(contract.GenerationJobFailureReconciler)
	if !ok {
		return
	}
	failureHandler, ok := p.executor.(contract.GenerationJobFailureHandler)
	if !ok {
		return
	}

	failedJobs, err := reconciler.ListUnreconciledFailures(ctx, p.config.ReconciliationBatch)
	if err != nil {
		if ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "list unreconciled generation failures",
				appendLogArgs([]any{"event", "generation_job_failure_reconciliation_list_failed"}, errorLogArgs(err))...,
			)
		}
		return
	}
	for _, job := range failedJobs {
		message := "generation job failed"
		if job.LastErrorMessage != nil && strings.TrimSpace(*job.LastErrorMessage) != "" {
			message = strings.TrimSpace(*job.LastErrorMessage)
		}
		cause := errors.New(message)
		if err := failureHandler.HandleTerminalFailure(ctx, job, cause); err != nil {
			p.logger.ErrorContext(ctx, "reconcile terminal generation failure",
				appendLogArgs(
					[]any{"event", "generation_job_failure_reconciliation_failed", "job_id", job.ID.String(), "request_id", job.RequestID.String(), "job_kind", string(job.Kind)},
					errorLogArgs(err),
				)...,
			)
			continue
		}
		if err := reconciler.MarkFailureHandled(ctx, job.ID, p.clock.Now()); err != nil {
			p.logger.ErrorContext(ctx, "mark reconciled generation failure",
				appendLogArgs(
					[]any{"event", "generation_job_failure_reconciliation_mark_failed", "job_id", job.ID.String(), "request_id", job.RequestID.String(), "job_kind", string(job.Kind)},
					errorLogArgs(err),
				)...,
			)
			continue
		}
		p.logger.InfoContext(ctx, "terminal generation failure reconciled",
			"event", "generation_job_failure_reconciled",
			"job_id", job.ID.String(),
			"request_id", job.RequestID.String(),
			"job_kind", string(job.Kind),
		)
	}
}

func (p *WorkerPool) runRetention(ctx context.Context) {
	p.purgeExpiredOperationalData(ctx)
	ticker := time.NewTicker(p.config.RetentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.purgeExpiredOperationalData(ctx)
		}
	}
}

func (p *WorkerPool) purgeExpiredOperationalData(ctx context.Context) {
	retention, ok := p.queue.(contract.GenerationJobRetention)
	if !ok {
		return
	}
	cutoff := p.clock.Now().Add(-p.config.RetentionPeriod)
	jobs, err := retention.PurgeTerminalBefore(ctx, cutoff, p.config.RetentionBatch)
	if err != nil {
		p.logger.ErrorContext(ctx, "purge terminal generation jobs", appendLogArgs([]any{"event", "generation_retention_jobs_failed"}, errorLogArgs(err))...)
		return
	}
	rawOutputs, err := retention.PurgeRawOutputsBefore(ctx, cutoff, p.config.RetentionBatch)
	if err != nil {
		p.logger.ErrorContext(ctx, "purge raw generation outputs", appendLogArgs([]any{"event", "generation_retention_raw_outputs_failed"}, errorLogArgs(err))...)
		return
	}
	if jobs > 0 || rawOutputs > 0 {
		p.logger.InfoContext(ctx, "generation retention completed",
			"event", "generation_retention_completed",
			"purged_root_jobs", jobs,
			"purged_raw_output_requests", rawOutputs,
			"cutoff", cutoff.UTC(),
		)
	}
}

func (p *WorkerPool) runMetrics(ctx context.Context) {
	p.logQueueMetrics(ctx)
	ticker := time.NewTicker(p.config.MetricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logQueueMetrics(ctx)
		}
	}
}

func (p *WorkerPool) logQueueMetrics(ctx context.Context) {
	provider, ok := p.queue.(contract.GenerationQueueMetricsProvider)
	if !ok {
		return
	}
	metrics, err := provider.GetQueueMetrics(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, "load generation queue metrics", appendLogArgs([]any{"event", "generation_queue_metrics_failed"}, errorLogArgs(err))...)
		return
	}
	p.logger.InfoContext(ctx, "generation queue metrics",
		"event", "generation_queue_metrics",
		"queued", metrics.Queued,
		"retry_scheduled", metrics.RetryScheduled,
		"running", metrics.Running,
		"completed", metrics.Completed,
		"failed", metrics.Failed,
		"cancelled", metrics.Cancelled,
		"expired_leases", metrics.ExpiredLeases,
		"unreconciled_failures", metrics.UnreconciledFailures,
	)
}
