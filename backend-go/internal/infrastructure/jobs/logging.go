package jobs

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
	"github.com/google/uuid"
)

const workerLogComponent = "generation_worker"

type errorCoder interface {
	ErrorCode() string
}

func (p *WorkerPool) workerLogger(workerID string) *slog.Logger {
	return p.logger.With(
		"component", workerLogComponent,
		"worker_id", workerID,
	)
}

func (p *WorkerPool) claimedJobLogger(job domain.GenerationJob, claim domain.JobClaim) *slog.Logger {
	return p.workerLogger(claim.WorkerID).With(
		"job_id", job.ID.String(),
		"request_id", job.RequestID.String(),
		"parent_job_id", optionalUUIDLogValue(job.ParentJobID),
		"job_kind", string(job.Kind),
		"target_id", optionalUUIDLogValue(job.TargetID),
		"attempt", claim.AttemptCount,
		"max_attempts", job.MaxAttempts,
		"priority", job.Priority,
	)
}

func jobFinishedLogArgs(startedAt, finishedAt time.Time, outcome string, status domain.GenerationJobStatus) []any {
	return []any{
		"outcome", outcome,
		"job_status", string(status),
		"finished_at", finishedAt.UTC(),
		"duration_ms", finishedAt.Sub(startedAt).Milliseconds(),
	}
}

func errorLogArgs(err error) []any {
	if err == nil {
		return nil
	}
	var coder errorCoder
	var errorCode any
	if errors.As(err, &coder) {
		errorCode = coder.ErrorCode()
	}
	traced := errtrace.Capture(err)
	return []any{
		"error", err,
		"error_type", fmt.Sprintf("%T", err),
		"error_code", errorCode,
		"stack", errtrace.Stack(traced),
	}
}

func appendLogArgs(groups ...[]any) []any {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	attributes := make([]any, 0, total)
	for _, group := range groups {
		attributes = append(attributes, group...)
	}
	return attributes
}

func optionalUUIDLogValue(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return id.String()
}

func optionalTimeLogValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func optionalStringLogValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nonNegativeDurationMilliseconds(duration time.Duration) int64 {
	if duration < 0 {
		return 0
	}
	return duration.Milliseconds()
}
