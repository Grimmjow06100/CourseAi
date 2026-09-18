package service

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

// ListGenerationRequests returns the authenticated user's resumable generation history.
func (s *CourseGeneratorService) ListGenerationRequests(ctx context.Context, filters contract.GenerationHistoryFilters) (contract.Page[contract.GenerationSummary], error) {
	if err := s.validateDependencies(); err != nil {
		return contract.Page[contract.GenerationSummary]{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.Page[contract.GenerationSummary]{}, err
	}
	if filters.PipelineStatus != nil {
		if err := filters.PipelineStatus.Validate(); err != nil {
			return contract.Page[contract.GenerationSummary]{}, err
		}
	}
	filters.ClerkUserID = owner
	filters.Pagination = filters.Pagination.Normalize()

	var page contract.Page[contract.GenerationSummary]
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		var err error
		page, err = repositories.GenerationRequests().ListGenerationRequests(ctx, filters)
		return err
	})
	return page, err
}
