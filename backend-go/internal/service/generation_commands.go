package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) enqueueFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	prompt := strings.TrimSpace(params.Prompt)
	if prompt == "" {
		return contract.GenerationStarted{}, ErrPromptRequired
	}
	if utf8.RuneCountInString(prompt) > 4000 {
		return contract.GenerationStarted{}, ErrPromptTooLong
	}
	if utf8.RuneCountInString(strings.TrimSpace(params.IdempotencyKey)) > 200 {
		return contract.GenerationStarted{}, fmt.Errorf("%w: idempotency key exceeds 200 characters", domain.ErrInvalidCollection)
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		idempotencyKey := generationRequestIdempotencyKey(owner, params.IdempotencyKey)
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
				if existingRequest.ClerkUserID != owner {
					return contract.ErrGenerationJobNotFound
				}
				request = existingRequest
				job = existing
				return nil
			}
			if !errors.Is(err, contract.ErrGenerationJobNotFound) {
				return err
			}
		}
		if err := s.enforceGenerationAdmission(ctx, repositories.GenerationRequests(), owner); err != nil {
			return err
		}

		now := s.now()
		createdRequest, err := domain.NewGenerationRequestAt(prompt, owner, now)
		if err != nil {
			return err
		}
		if idempotencyKey == "" {
			idempotencyKey = "analysis:" + createdRequest.ID.String() + ":v1"
		}
		createdJob, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
			GenerationAttempt: createdRequest.GenerationAttempt,
			RequestID:         createdRequest.ID,
			Kind:              domain.GenerationJobKindAnalysis,
			IdempotencyKey:    idempotencyKey,
			Payload:           json.RawMessage(`{}`),
			AvailableAt:       now,
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
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	params, err = normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	brief := generationBriefFromStructureParams(params)
	if err := brief.Validate(); err != nil {
		return contract.GenerationStarted{}, err
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, params.RequestID, owner); err != nil {
			return err
		}
		locked, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, params.RequestID)
		if err != nil {
			return err
		}
		if locked.IsOutOfScope {
			return ErrGenerationOutOfScope
		}
		if locked.AnalysisCompletedAt == nil {
			return ErrGenerationAnalysisRequired
		}
		if locked.PipelineStatus == domain.PipelineStatusFailed {
			return ErrGenerationStructureRetryNotAllowed
		}
		if locked.PipelineStatus == domain.PipelineStatusAwaitingClarification {
			return ErrGenerationAwaitingClarification
		}
		if locked.ConfirmedBrief == nil {
			if err := locked.ConfirmBrief(brief, s.now()); err != nil {
				return err
			}
			locked, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, locked)
			if err != nil {
				return err
			}
		} else if !generationBriefEqual(*locked.ConfirmedBrief, brief) {
			return ErrClarificationAlreadySubmitted
		}
		request = locked
		job, err = s.enqueueArchitectureJobWithRepositories(ctx, repositories, request, nil)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) SubmitClarifications(ctx context.Context, params contract.SubmitClarificationsParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	if params.RequestID == uuid.Nil {
		return contract.GenerationStarted{}, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}
	if err := params.Language.Validate(); err != nil {
		return contract.GenerationStarted{}, err
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, params.RequestID, owner); err != nil {
			return err
		}
		locked, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, params.RequestID)
		if err != nil {
			return err
		}
		if locked.PipelineStatus != domain.PipelineStatusAwaitingClarification {
			if clarificationSubmissionMatches(locked, params) {
				request = locked
				job, err = s.enqueueArchitectureJobWithRepositories(ctx, repositories, locked, nil)
				return err
			}
			return ErrGenerationNotAwaitingClarification
		}
		if err := locked.SubmitClarifications(params.Answers, params.Title, params.Synopsis, params.Language, s.now()); err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, locked)
		if err != nil {
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

func (s *CourseGeneratorService) EnqueueStructureRetry(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	params, err = normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	brief := generationBriefFromStructureParams(params)
	if err := brief.Validate(); err != nil {
		return contract.GenerationStarted{}, err
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, params.RequestID, owner); err != nil {
			return err
		}
		loadedRequest, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, params.RequestID)
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
		if err := s.cancelSupersededJobs(ctx, repositories, loadedRequest); err != nil {
			return err
		}
		if err := loadedRequest.ConfirmBrief(brief, s.now()); err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, loadedRequest)
		if err != nil {
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

func (s *CourseGeneratorService) DeleteGenerationRequest(ctx context.Context, requestID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return err
	}
	if requestID == uuid.Nil {
		return fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}
		if _, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID); err != nil {
			return err
		}
		return repositories.GenerationRequests().DeleteGenerationRequest(ctx, requestID)
	})
}

func (s *CourseGeneratorService) enqueueTargetedContentJob(ctx context.Context, kind domain.GenerationJobKind, targetID uuid.UUID) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	if targetID == uuid.Nil {
		return contract.GenerationStarted{}, fmt.Errorf("%w: generation job target id", domain.ErrBlankField)
	}

	var request domain.GenerationRequest
	var job domain.GenerationJob
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		resource := ownedModule
		if kind == domain.GenerationJobKindLessonContent {
			resource = ownedLesson
		}
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), resource, targetID, owner); err != nil {
			return err
		}
		requestID, err := requestIDForTarget(ctx, repositories, kind, targetID)
		if err != nil {
			return err
		}
		request, err = repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
		if err != nil {
			return err
		}
		if request.PipelineStatus == domain.PipelineStatusFailed || request.PipelineStatus == domain.PipelineStatusPartial {
			return ErrGenerationNotRetryable
		}
		idempotencyKey := fmt.Sprintf("%s:%s:v%d:%s", kind, request.ID, request.ClarificationVersion, targetID)
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
			GenerationAttempt: request.GenerationAttempt,
			RequestID:         request.ID,
			Kind:              kind,
			TargetID:          &targetID,
			IdempotencyKey:    idempotencyKey,
			Payload:           json.RawMessage(`{}`),
			AvailableAt:       s.now(),
		}, s.now())
		if err != nil {
			return err
		}
		job, err = s.enqueueWithCapacity(ctx, repositories, createdJob)
		return err
	})
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	return s.generationStarted(request, job), nil
}

func (s *CourseGeneratorService) enqueueWithCapacity(ctx context.Context, repositories contract.TransactionalRepositories, job domain.GenerationJob) (domain.GenerationJob, error) {
	if job.GenerationAttempt > 1 {
		job.IdempotencyKey += fmt.Sprintf(":attempt:%d", job.GenerationAttempt)
	}
	jobs, err := repositories.GenerationJobs().ListByRequestID(ctx, job.RequestID)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	for _, existing := range jobs {
		if existing.IsCurrent && existing.GenerationAttempt == job.GenerationAttempt && existing.Kind == job.Kind && pointer.Equal(existing.TargetID, job.TargetID) {
			return existing, nil
		}
	}

	if s.config.MaxPendingJobs > 0 {
		pending, err := repositories.GenerationJobs().CountPendingWithAdmissionLock(ctx)
		if err != nil {
			return domain.GenerationJob{}, err
		}
		if pending >= s.config.MaxPendingJobs {
			return domain.GenerationJob{}, contract.ErrGenerationQueueSaturated
		}
	}
	return repositories.GenerationJobs().Enqueue(ctx, job)
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

func deterministicJobKey(prefix string, requestID uuid.UUID, payload json.RawMessage) string {
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%s:%s:%x", prefix, requestID, digest[:12])
}

func generationRequestIdempotencyKey(owner string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "analysis:clerk_user:" + strings.TrimSpace(owner) + ":client:" + value
}

func generationBriefFromStructureParams(params contract.GenerateStructureParams) domain.GenerationBrief {
	return domain.GenerationBrief{
		Title:        params.Title,
		Synopsis:     params.Synopsis,
		CurrentLevel: params.CurrentLevel,
		TargetLevel:  params.TargetLevel,
		Goals:        params.Goals,
		Language:     params.Language,
	}
}

func generationBriefEqual(left, right domain.GenerationBrief) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func clarificationSubmissionMatches(request domain.GenerationRequest, params contract.SubmitClarificationsParams) bool {
	if request.ConfirmedBrief == nil || request.ClarificationsSubmittedAt == nil {
		return false
	}
	clone := request
	clone.PipelineStatus = domain.PipelineStatusAwaitingClarification
	clone.ConfirmedBrief = nil
	clone.BriefConfirmedAt = nil
	clone.ClarificationsSubmittedAt = nil
	clone.ClarificationAnswers = nil
	clone.ClarificationVersion = 0
	if err := clone.SubmitClarifications(params.Answers, params.Title, params.Synopsis, params.Language, clone.UpdatedAt); err != nil {
		return false
	}
	return generationBriefEqual(*request.ConfirmedBrief, *clone.ConfirmedBrief) &&
		clarificationAnswersEqual(request.ClarificationAnswers, clone.ClarificationAnswers)
}

func clarificationAnswersEqual(left, right []domain.ClarificationAnswer) bool {
	canonical := func(answers []domain.ClarificationAnswer) map[string]string {
		values := make(map[string]string, len(answers))
		for _, answer := range answers {
			selected := append([]string(nil), answer.SelectedValues...)
			sort.Strings(selected)
			values[strings.TrimSpace(answer.QuestionID)] = strings.Join(selected, "\x00")
		}
		return values
	}
	return reflect.DeepEqual(canonical(left), canonical(right))
}
