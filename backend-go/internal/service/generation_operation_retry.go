package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/google/uuid"
)

// RetryGenerationOperations admits the entire selection under the request lock.
// The receipt is committed with the replacements, including on lost HTTP replies.
func (s *CourseGeneratorService) RetryGenerationOperations(ctx context.Context, params contract.RetryOperationsParams) (contract.RetryOperationsResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.RetryOperationsResult{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.RetryOperationsResult{}, err
	}
	params.IdempotencyKey = strings.TrimSpace(params.IdempotencyKey)
	if len(params.IdempotencyKey) < 8 || len(params.IdempotencyKey) > 200 || len(params.Operations) == 0 || len(params.Operations) > 500 {
		return contract.RetryOperationsResult{}, fmt.Errorf("%w: retry key and 1–500 operations required", domain.ErrBlankField)
	}
	operations := append([]contract.RetryOperation(nil), params.Operations...)
	sort.Slice(operations, func(i, j int) bool { return operations[i].JobID.String() < operations[j].JobID.String() })
	for i, op := range operations {
		if op.JobID == uuid.Nil || op.OperationVersion < 1 || (i > 0 && operations[i-1].JobID == op.JobID) {
			return contract.RetryOperationsResult{}, fmt.Errorf("%w: retry selection", domain.ErrBlankField)
		}
	}
	payload, err := json.Marshal(struct {
		Operations []contract.RetryOperation `json:"operations"`
	}{operations})
	if err != nil {
		return contract.RetryOperationsResult{}, err
	}
	result := contract.RetryOperationsResult{RequestID: params.RequestID, Replacements: []contract.OperationReplacement{}}
	err = s.withinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, r.Ownership(), ownedGenerationRequest, params.RequestID, owner); err != nil {
			return err
		}
		request, err := r.GenerationRequests().FindGenerationRequestForUpdate(ctx, params.RequestID)
		if err != nil {
			return err
		}
		previousPayload, previousResult, err := r.Tracking().RetryReceipt(ctx, params.RequestID, params.IdempotencyKey)
		if err == nil {
			if !jsonutil.EqualObjects(previousPayload, payload) {
				return contract.ErrGenerationJobIdempotencyConflict
			}
			return json.Unmarshal(previousResult, &result)
		}
		if !errors.Is(err, contract.ErrGenerationJobNotFound) {
			return err
		}
		snapshot, err := r.Tracking().Snapshot(ctx, params.RequestID)
		if err != nil {
			return err
		}
		enrichTracking(&snapshot)
		current := make(map[uuid.UUID]contract.TrackingOperation, len(snapshot.Operations))
		for _, op := range snapshot.Operations {
			current[op.ID] = op
		}
		for _, selected := range operations {
			op, found := current[selected.JobID]
			if !found || !op.Retryable || op.OperationVersion != selected.OperationVersion {
				return contract.ErrGenerationJobIdempotencyConflict
			}
		}
		if err := s.enforceGenerationRetryAdmission(ctx, r.GenerationRequests(), owner, !request.PipelineStatus.IsTerminal()); err != nil {
			return err
		}
		if err := request.ResumeOperation(s.now()); err != nil {
			return ErrGenerationNotRetryable
		}
		if _, err := r.GenerationRequests().UpdateGenerationRequest(ctx, request); err != nil {
			return err
		}
		if err := s.recoverFailedCourse(ctx, r, request.ID); err != nil {
			return err
		}
		// Reserve enough queue capacity before changing any operation. The shared
		// admission lock is held until the receipt and all replacements commit.
		pending, err := r.GenerationJobs().CountPendingWithAdmissionLock(ctx)
		if err != nil {
			return err
		}
		if s.config.MaxPendingJobs > 0 && pending+int64(len(operations)) > s.config.MaxPendingJobs {
			return contract.ErrGenerationQueueSaturated
		}
		for _, selected := range operations {
			old, err := r.GenerationJobs().FindByID(ctx, selected.JobID)
			if err != nil {
				return err
			}
			if err := r.Tracking().Supersede(ctx, old.ID, selected.OperationVersion); err != nil {
				return err
			}
			job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
				GenerationAttempt: request.GenerationAttempt, RequestID: request.ID, ParentJobID: old.ParentJobID,
				Kind: old.Kind, TargetID: old.TargetID, Payload: old.Payload, Priority: old.Priority,
				MaxAttempts: old.MaxAttempts, IdempotencyKey: "operation-retry:" + old.ID.String() + ":" + params.IdempotencyKey,
			}, s.now())
			if err != nil {
				return err
			}
			job.OperationVersion = old.OperationVersion + 1
			job.SupersedesJobID = &old.ID
			job, err = r.GenerationJobs().Enqueue(ctx, job)
			if err != nil {
				return err
			}
			result.Replacements = append(result.Replacements, contract.OperationReplacement{PreviousJobID: old.ID, JobID: job.ID, OperationVersion: job.OperationVersion})
		}
		after, err := r.Tracking().Snapshot(ctx, request.ID)
		if err != nil {
			return err
		}
		result.Revision, result.GenerationAttempt = after.Revision, request.GenerationAttempt
		encoded, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return r.Tracking().SaveRetryReceipt(ctx, request.ID, params.IdempotencyKey, payload, encoded)
	})
	return result, err
}
