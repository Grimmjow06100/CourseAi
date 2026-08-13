package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) enqueueFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	prompt := strings.TrimSpace(params.Prompt)
	if prompt == "" {
		return contract.GenerationStarted{}, ErrPromptRequired
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		idempotencyKey := fullCourseIdempotencyKey(params.IdempotencyKey)
		if idempotencyKey != "" {
			existing, err := repositories.GenerationJobs().FindByIdempotencyKey(ctx, idempotencyKey)
			if err == nil {
				existingRequest, findErr := repositories.GenerationRequests().FindGenerationRequestByID(ctx, existing.RequestID)
				if findErr != nil {
					return findErr
				}
				if existingRequest.InitialUserPrompt != prompt {
					return contract.ErrGenerationJobIdempotencyConflict
				}
				request = existingRequest
				job = existing
				return nil
			}
			if !errors.Is(err, contract.ErrGenerationJobNotFound) {
				return err
			}
		}

		now := s.now()
		createdRequest, err := domain.NewGenerationRequestAt(prompt, now)
		if err != nil {
			return err
		}
		if idempotencyKey == "" {
			idempotencyKey = "full_course:" + createdRequest.ID.String() + ":v1"
		}
		createdJob, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
			RequestID:      createdRequest.ID,
			Kind:           domain.GenerationJobKindFullCourse,
			IdempotencyKey: idempotencyKey,
			Payload:        json.RawMessage(`{}`),
			AvailableAt:    now,
		}, now)
		if err != nil {
			return err
		}

		request, err = repositories.GenerationRequests().SaveGenerationRequest(ctx, createdRequest)
		if err != nil {
			return err
		}
		job, err = repositories.GenerationJobs().Enqueue(ctx, createdJob)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) EnqueueCourseStructure(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	params, err := normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	request, err := s.loadGenerationRequest(ctx, params.RequestID)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	if request.IsOutOfScope {
		return contract.GenerationStarted{}, ErrGenerationOutOfScope
	}
	if !requestHasAnalysis(request) {
		return contract.GenerationStarted{}, ErrGenerationAnalysisRequired
	}
	if request.PipelineStatus == domain.PipelineStatusFailed {
		return contract.GenerationStarted{}, ErrGenerationStructureRetryNotAllowed
	}

	payload, err := architectureJobPayload(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	job, err := s.enqueueJob(ctx, domain.NewGenerationJobParams{
		RequestID:      request.ID,
		Kind:           domain.GenerationJobKindArchitecture,
		IdempotencyKey: deterministicJobKey("architecture", request.ID, payload),
		Payload:        payload,
		AvailableAt:    s.now(),
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) EnqueueStructureRetry(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	params, err := normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	payload, err := architectureJobPayload(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedRequest, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, params.RequestID)
		if err != nil {
			return err
		}
		if loadedRequest.PipelineStatus != domain.PipelineStatusFailed {
			return ErrGenerationStructureRetryNotAllowed
		}
		if !isStructureRetryableStep(loadedRequest.CurrentStep) {
			return ErrGenerationStructureRetryStepMismatch
		}
		if loadedRequest.IsOutOfScope {
			return ErrGenerationOutOfScope
		}
		if !requestHasAnalysis(loadedRequest) {
			return ErrGenerationAnalysisRequired
		}
		if err := repositories.Courses().DeleteCourseByRequestID(ctx, loadedRequest.ID); err != nil && !errors.Is(err, contract.ErrCourseNotFound) {
			return err
		}
		if err := loadedRequest.RestartFromFailure(stepAnalysisCompleted, 25, s.now()); err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, loadedRequest)
		if err != nil {
			return err
		}
		createdJob, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
			RequestID:      request.ID,
			Kind:           domain.GenerationJobKindArchitecture,
			IdempotencyKey: "architecture:" + request.ID.String() + ":retry:" + uuid.NewString(),
			Payload:        payload,
			AvailableAt:    s.now(),
		}, s.now())
		if err != nil {
			return err
		}
		job, err = repositories.GenerationJobs().Enqueue(ctx, createdJob)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) EnqueueLessonContentGeneration(ctx context.Context, lessonID uuid.UUID) (contract.GenerationStarted, error) {
	return s.enqueueTargetedContentJob(ctx, domain.GenerationJobKindLessonContent, lessonID)
}

func (s *CourseGeneratorService) EnqueueModuleContentGeneration(ctx context.Context, moduleID uuid.UUID) (contract.GenerationStarted, error) {
	return s.enqueueTargetedContentJob(ctx, domain.GenerationJobKindModuleContent, moduleID)
}

func (s *CourseGeneratorService) GetGenerationJob(ctx context.Context, jobID uuid.UUID) (domain.GenerationJob, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.GenerationJob{}, err
	}
	if jobID == uuid.Nil {
		return domain.GenerationJob{}, fmt.Errorf("%w: generation job id", domain.ErrBlankField)
	}
	var job domain.GenerationJob
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedJob, err := repositories.GenerationJobs().FindByID(ctx, jobID)
		if err != nil {
			return err
		}
		job = loadedJob
		return nil
	})
	return job, err
}

func (s *CourseGeneratorService) enqueueTargetedContentJob(ctx context.Context, kind domain.GenerationJobKind, targetID uuid.UUID) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	if targetID == uuid.Nil {
		return contract.GenerationStarted{}, fmt.Errorf("%w: generation job target id", domain.ErrBlankField)
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		requestID, err := requestIDForTarget(ctx, repositories, kind, targetID)
		if err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		idempotencyKey := string(kind) + ":" + targetID.String() + ":v1"
		existing, findErr := repositories.GenerationJobs().FindByIdempotencyKey(ctx, idempotencyKey)
		if findErr == nil && existing.Status != domain.GenerationJobStatusFailed && existing.Status != domain.GenerationJobStatusCancelled {
			job = existing
			return nil
		}
		if findErr != nil && !errors.Is(findErr, contract.ErrGenerationJobNotFound) {
			return findErr
		}
		if findErr == nil {
			idempotencyKey += ":retry:" + uuid.NewString()
		}
		createdJob, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
			RequestID:      request.ID,
			Kind:           kind,
			TargetID:       &targetID,
			IdempotencyKey: idempotencyKey,
			Payload:        json.RawMessage(`{}`),
			AvailableAt:    s.now(),
		}, s.now())
		if err != nil {
			return err
		}
		job, err = repositories.GenerationJobs().Enqueue(ctx, createdJob)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) enqueueJob(ctx context.Context, params domain.NewGenerationJobParams) (domain.GenerationJob, error) {
	job, err := domain.NewGenerationJobAt(params, s.now())
	if err != nil {
		return domain.GenerationJob{}, err
	}
	var saved domain.GenerationJob
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		var err error
		saved, err = repositories.GenerationJobs().Enqueue(ctx, job)
		return err
	})
	return saved, err
}

func requestIDForTarget(ctx context.Context, repositories contract.TransactionalRepositories, kind domain.GenerationJobKind, targetID uuid.UUID) (uuid.UUID, error) {
	switch kind {
	case domain.GenerationJobKindLessonContent:
		lesson, err := repositories.Lessons().FindLessonByID(ctx, targetID)
		if err != nil {
			return uuid.Nil, err
		}
		module, err := repositories.Modules().FindModuleByID(ctx, lesson.ModuleID)
		if err != nil {
			return uuid.Nil, err
		}
		course, err := repositories.Courses().FindCourseStateByID(ctx, module.CourseID)
		if err != nil {
			return uuid.Nil, err
		}
		return course.RequestID, nil
	case domain.GenerationJobKindModuleContent:
		module, err := repositories.Modules().FindModuleByID(ctx, targetID)
		if err != nil {
			return uuid.Nil, err
		}
		course, err := repositories.Courses().FindCourseStateByID(ctx, module.CourseID)
		if err != nil {
			return uuid.Nil, err
		}
		return course.RequestID, nil
	default:
		return uuid.Nil, fmt.Errorf("%w: unsupported targeted command %s", domain.ErrInvalidGenerationJobKind, kind)
	}
}

func architectureJobPayload(params contract.GenerateStructureParams) (json.RawMessage, error) {
	payload, err := json.Marshal(contract.ArchitectureJobPayload{
		Title:        params.Title,
		Synopsis:     params.Synopsis,
		CurrentLevel: params.CurrentLevel,
		TargetLevel:  params.TargetLevel,
		Goals:        params.Goals,
		Language:     params.Language,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal architecture job payload: %w", err)
	}
	return payload, nil
}

func deterministicJobKey(prefix string, requestID uuid.UUID, payload json.RawMessage) string {
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%s:%s:%x", prefix, requestID, digest[:12])
}

func fullCourseIdempotencyKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "full_course:client:" + value
}
