package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const defaultGenerationJobErrorCode = "generation_job_failed"

type GenerationJobRepository struct {
	queries *dbsqlc.Queries
}

func NewGenerationJobRepository(db DBTX) *GenerationJobRepository {
	return &GenerationJobRepository{queries: dbsqlc.New(db)}
}

func (r *GenerationJobRepository) Enqueue(ctx context.Context, job domain.GenerationJob) (domain.GenerationJob, error) {
	if err := validateQueuedGenerationJob(job); err != nil {
		return domain.GenerationJob{}, err
	}

	row, err := r.queries.EnqueueGenerationJob(ctx, enqueueGenerationJobParams(job))
	if err == nil {
		return generationJobFromSQLC(row)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.GenerationJob{}, err
	}

	existing, findErr := r.FindByIdempotencyKey(ctx, job.IdempotencyKey)
	if findErr != nil {
		return domain.GenerationJob{}, findErr
	}
	if !sameGenerationJobOperation(existing, job) {
		return domain.GenerationJob{}, contract.ErrGenerationJobIdempotencyConflict
	}
	return existing, nil
}

func (r *GenerationJobRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.GenerationJob, error) {
	if id == uuid.Nil {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id", domain.ErrBlankField)
	}
	row, err := r.queries.GetGenerationJobByID(ctx, id)
	if err != nil {
		return domain.GenerationJob{}, mapNoRows(err, contract.ErrGenerationJobNotFound)
	}
	return generationJobFromSQLC(row)
}

func (r *GenerationJobRepository) FindByIdempotencyKey(ctx context.Context, key string) (domain.GenerationJob, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: idempotency key", domain.ErrBlankField)
	}
	row, err := r.queries.GetGenerationJobByIdempotencyKey(ctx, key)
	if err != nil {
		return domain.GenerationJob{}, mapNoRows(err, contract.ErrGenerationJobNotFound)
	}
	return generationJobFromSQLC(row)
}

func (r *GenerationJobRepository) ListByRequestID(ctx context.Context, requestID uuid.UUID) ([]domain.GenerationJob, error) {
	if requestID == uuid.Nil {
		return nil, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}
	rows, err := r.queries.ListGenerationJobsByRequestID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	return generationJobsFromSQLC(rows)
}

func (r *GenerationJobRepository) ClaimNext(ctx context.Context, workerID string, lockedUntil time.Time) (domain.GenerationJob, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: worker id", domain.ErrBlankField)
	}
	if lockedUntil.IsZero() {
		return domain.GenerationJob{}, fmt.Errorf("%w: locked until", domain.ErrBlankField)
	}

	row, err := r.queries.ClaimNextGenerationJob(ctx, dbsqlc.ClaimNextGenerationJobParams{
		WorkerID:    &workerID,
		LockedUntil: pointer.To(lockedUntil),
	})
	if err != nil {
		return domain.GenerationJob{}, mapNoRows(err, contract.ErrGenerationJobUnavailable)
	}
	return generationJobFromSQLC(row)
}

func (r *GenerationJobRepository) RenewLease(ctx context.Context, claim domain.JobClaim, lockedUntil time.Time) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	if lockedUntil.IsZero() {
		return fmt.Errorf("%w: locked until", domain.ErrBlankField)
	}
	rows, err := r.queries.RenewGenerationJobLease(ctx, dbsqlc.RenewGenerationJobLeaseParams{
		LockedUntil:  pointer.To(lockedUntil),
		ID:           claim.JobID,
		WorkerID:     textutil.TrimmedPointer(claim.WorkerID),
		AttemptCount: int32(claim.AttemptCount),
	})
	return requireGenerationJobClaim(rows, err)
}

func (r *GenerationJobRepository) Complete(ctx context.Context, claim domain.JobClaim, completedAt time.Time) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	if completedAt.IsZero() {
		return fmt.Errorf("%w: completed at", domain.ErrBlankField)
	}
	rows, err := r.queries.CompleteGenerationJob(ctx, dbsqlc.CompleteGenerationJobParams{
		CompletedAt:  pointer.To(completedAt),
		ID:           claim.JobID,
		WorkerID:     textutil.TrimmedPointer(claim.WorkerID),
		AttemptCount: int32(claim.AttemptCount),
	})
	return requireGenerationJobClaim(rows, err)
}

func (r *GenerationJobRepository) Retry(ctx context.Context, claim domain.JobClaim, retryAt time.Time, cause error) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	if retryAt.IsZero() {
		return fmt.Errorf("%w: retry at", domain.ErrBlankField)
	}
	errorCode, errorMessage, err := generationJobFailure(cause)
	if err != nil {
		return err
	}
	rows, err := r.queries.RetryGenerationJob(ctx, dbsqlc.RetryGenerationJobParams{
		RetryAt:      retryAt,
		ErrorCode:    &errorCode,
		ErrorMessage: &errorMessage,
		ID:           claim.JobID,
		WorkerID:     textutil.TrimmedPointer(claim.WorkerID),
		AttemptCount: int32(claim.AttemptCount),
	})
	return requireGenerationJobClaim(rows, err)
}

func (r *GenerationJobRepository) Fail(ctx context.Context, claim domain.JobClaim, cause error, failedAt time.Time) error {
	if err := claim.Validate(); err != nil {
		return err
	}
	if failedAt.IsZero() {
		return fmt.Errorf("%w: failed at", domain.ErrBlankField)
	}
	errorCode, errorMessage, err := generationJobFailure(cause)
	if err != nil {
		return err
	}
	rows, err := r.queries.FailGenerationJob(ctx, dbsqlc.FailGenerationJobParams{
		FailedAt:     pointer.To(failedAt),
		ErrorCode:    &errorCode,
		ErrorMessage: &errorMessage,
		ID:           claim.JobID,
		WorkerID:     textutil.TrimmedPointer(claim.WorkerID),
		AttemptCount: int32(claim.AttemptCount),
	})
	return requireGenerationJobClaim(rows, err)
}

func (r *GenerationJobRepository) Cancel(ctx context.Context, id uuid.UUID, cancelledAt time.Time) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: generation job id", domain.ErrBlankField)
	}
	if cancelledAt.IsZero() {
		return fmt.Errorf("%w: cancelled at", domain.ErrBlankField)
	}
	rows, err := r.queries.CancelGenerationJob(ctx, dbsqlc.CancelGenerationJobParams{
		CancelledAt: pointer.To(cancelledAt),
		ID:          id,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return contract.ErrGenerationJobNotCancellable
	}
	return nil
}

func (r *GenerationJobRepository) RequeueExpired(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		return 0, fmt.Errorf("%w: requeued at", domain.ErrBlankField)
	}
	return r.queries.RequeueExpiredGenerationJobs(ctx, now)
}

func validateQueuedGenerationJob(job domain.GenerationJob) error {
	if err := job.Validate(); err != nil {
		return err
	}
	if job.Status != domain.GenerationJobStatusQueued || job.AttemptCount != 0 || job.LockedBy != nil || job.LockedUntil != nil || job.StartedAt != nil || job.CompletedAt != nil || job.LastErrorCode != nil || job.LastErrorMessage != nil {
		return domain.ErrInvalidGenerationJobState
	}
	return nil
}

func enqueueGenerationJobParams(job domain.GenerationJob) dbsqlc.EnqueueGenerationJobParams {
	return dbsqlc.EnqueueGenerationJobParams{
		ID:               job.ID,
		RequestID:        job.RequestID,
		ParentJobID:      nullableUUID(job.ParentJobID),
		Kind:             dbsqlc.GenerationJobKind(job.Kind),
		Status:           dbsqlc.GenerationJobStatus(job.Status),
		TargetID:         nullableUUID(job.TargetID),
		IdempotencyKey:   job.IdempotencyKey,
		Payload:          jsonutil.Clone(job.Payload),
		Priority:         int32(job.Priority),
		AttemptCount:     int32(job.AttemptCount),
		MaxAttempts:      int32(job.MaxAttempts),
		AvailableAt:      job.AvailableAt,
		LockedBy:         pointer.Clone(job.LockedBy),
		LockedUntil:      pointer.Clone(job.LockedUntil),
		StartedAt:        pointer.Clone(job.StartedAt),
		CompletedAt:      pointer.Clone(job.CompletedAt),
		LastErrorCode:    pointer.Clone(job.LastErrorCode),
		LastErrorMessage: pointer.Clone(job.LastErrorMessage),
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}
}

func sameGenerationJobOperation(existing, candidate domain.GenerationJob) bool {
	return existing.RequestID == candidate.RequestID &&
		existing.Kind == candidate.Kind &&
		pointer.Equal(existing.ParentJobID, candidate.ParentJobID) &&
		pointer.Equal(existing.TargetID, candidate.TargetID) &&
		jsonutil.EqualObjects(existing.Payload, candidate.Payload)
}

func requireGenerationJobClaim(rows int64, err error) error {
	if err != nil {
		return err
	}
	if rows != 1 {
		return contract.ErrGenerationJobClaimLost
	}
	return nil
}

type generationJobErrorCoder interface {
	ErrorCode() string
}

func generationJobFailure(cause error) (string, string, error) {
	if cause == nil {
		return "", "", fmt.Errorf("%w: generation job failure", domain.ErrBlankField)
	}
	code := defaultGenerationJobErrorCode
	var coded generationJobErrorCoder
	if errors.As(cause, &coded) {
		if candidate := strings.TrimSpace(coded.ErrorCode()); candidate != "" {
			code = candidate
		}
	}
	message := strings.TrimSpace(cause.Error())
	if message == "" {
		message = "generation job failed without an error message"
	}
	return code, message, nil
}

func nullableUUID(value *uuid.UUID) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *value, Valid: true}
}
