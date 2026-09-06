package service

import (
	"context"
	"fmt"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

// ListGenerationJobs returns public job state after checking ownership of the request.
func (s *CourseGeneratorService) ListGenerationJobs(ctx context.Context, requestID uuid.UUID) ([]domain.GenerationJob, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return nil, err
	}
	if requestID == uuid.Nil {
		return nil, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}
	var jobs []domain.GenerationJob
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}
		var err error
		jobs, err = repositories.GenerationJobs().ListByRequestID(ctx, requestID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list generation jobs: %w", err)
	}
	return jobs, nil
}
