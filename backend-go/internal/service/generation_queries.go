package service

import (
	"context"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (contract.GenerationStatus, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStatus{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStatus{}, err
	}

	var status contract.GenerationStatus
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}
		var err error
		status, err = repositories.GenerationRequests().FindGenerationStatusByID(ctx, requestID)
		return err
	})
	if err != nil {
		return contract.GenerationStatus{}, err
	}
	if status.PipelineStatus == domain.PipelineStatusAwaitingClarification {
		status.ActionRequired = &contract.GenerationActionRequired{
			Type: "submit_clarifications",
			URL:  formatGenerationURL(s.config.ClarificationURLFormat, "/api/generations/%s/clarifications", requestID),
		}
	}
	return status, nil
}

func (s *CourseGeneratorService) GetGenerationResult(ctx context.Context, requestID uuid.UUID) (contract.GenerationResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationResult{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	var result contract.GenerationResult
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
		if err != nil {
			return err
		}
		if request.PipelineStatus != domain.PipelineStatusCompleted {
			return ErrGenerationNotCompleted
		}

		course, err := repositories.Courses().FindCourseByRequestID(ctx, request.ID)
		if err != nil {
			return err
		}

		result = contract.GenerationResult{
			Request: request,
			Course:  course,
		}
		return nil
	})
	if err != nil {
		return contract.GenerationResult{}, err
	}
	return result, nil
}

func (s *CourseGeneratorService) GetGenerationJob(ctx context.Context, jobID uuid.UUID) (domain.GenerationJob, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.GenerationJob{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if jobID == uuid.Nil {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id", domain.ErrBlankField)
	}
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationJob, jobID, owner); err != nil {
			return err
		}
		loadedJob, err := repositories.GenerationJobs().FindByID(ctx, jobID)
		if err != nil {
			return err
		}
		job = loadedJob
		return nil
	})
	return job, err
}
