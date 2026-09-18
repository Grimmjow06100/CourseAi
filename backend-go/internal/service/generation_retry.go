package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type retryContinuation int

const (
	retryFromAnalysis retryContinuation = iota
	retryToClarification
	retryFromArchitecture
)

func (s *CourseGeneratorService) RetryFullCourseGeneration(ctx context.Context, requestID uuid.UUID) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}

		lockedRequest, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
		if err != nil {
			return err
		}
		continuation, err := s.prepareFailedRequestRetry(&lockedRequest)
		if err != nil {
			return err
		}
		if err := s.cancelSupersededJobs(ctx, repositories, lockedRequest); err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, lockedRequest)
		if err != nil {
			return err
		}

		if continuation != retryFromArchitecture {
			job, err = s.enqueueAnalysisRetry(ctx, repositories, request, continuation)
			return err
		}
		if err := s.recoverFailedCourse(ctx, repositories, request.ID); err != nil {
			return err
		}
		job, err = s.enqueueArchitectureJobWithRepositories(ctx, repositories, request, nil)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) prepareFailedRequestRetry(request *domain.GenerationRequest) (retryContinuation, error) {
	if request.PipelineStatus != domain.PipelineStatusFailed {
		return retryFromAnalysis, ErrGenerationNotRetryable
	}

	now := s.now()
	if request.AnalysisCompletedAt == nil {
		return retryFromAnalysis, request.RestartFromFailure(stepAnalysis, 0, now)
	}
	if request.ConfirmedBrief == nil && request.NeedsClarification() {
		return retryToClarification, request.RestartFromFailure(stepAnalysisCompleted, 25, now)
	}

	brief := request.ConfirmedBrief
	if err := request.RestartFromFailure(stepAnalysisCompleted, 25, now); err != nil {
		return retryFromArchitecture, err
	}
	if brief == nil {
		return retryFromArchitecture, request.ConfirmDetectedBrief(now)
	}
	return retryFromArchitecture, request.ConfirmBrief(*brief, now)
}

func (s *CourseGeneratorService) enqueueAnalysisRetry(
	ctx context.Context,
	repositories contract.TransactionalRepositories,
	request domain.GenerationRequest,
	continuation retryContinuation,
) (domain.GenerationJob, error) {
	reason := "retry"
	if continuation == retryToClarification {
		reason = "clarification-resume"
	}
	now := s.now()
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		GenerationAttempt: request.GenerationAttempt,
		RequestID:         request.ID,
		Kind:              domain.GenerationJobKindAnalysis,
		IdempotencyKey:    "analysis:" + request.ID.String() + ":" + reason + ":" + uuid.NewString(),
		Payload:           json.RawMessage(`{}`),
		AvailableAt:       now,
	}, now)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	return s.enqueueWithCapacity(ctx, repositories, job)
}

func (s *CourseGeneratorService) recoverFailedCourse(
	ctx context.Context,
	repositories contract.TransactionalRepositories,
	requestID uuid.UUID,
) error {
	course, err := repositories.Courses().FindCourseByRequestID(ctx, requestID)
	if errors.Is(err, contract.ErrCourseNotFound) {
		return nil
	}
	if err != nil || course.Status != domain.CourseStatusFailed {
		return err
	}
	if err := course.RestartGenerationFromFailure(courseRecoveryStatus(course)); err != nil {
		return err
	}
	course.UpdatedAt = s.now()
	_, err = repositories.Courses().UpdateCourse(ctx, course)
	return err
}
