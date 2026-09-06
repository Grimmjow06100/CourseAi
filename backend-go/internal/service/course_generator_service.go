package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	"github.com/google/uuid"
)

const (
	stepAnalysis                = "analysis"
	stepAnalysisCompleted       = "analysis_completed"
	stepAwaitingClarification   = "awaiting_clarification"
	stepClarificationsCompleted = "clarifications_completed"
	stepArchitecture            = "architecture_generation"
	stepLessonPlan              = "lesson_plan_generation"
	stepLessonContent           = "lesson_content_generation"
	stepGenerationSuccess       = "generation_completed"
)

var (
	ErrCourseGeneratorDependency            = fmt.Errorf("course generator service: %w", contract.ErrServiceDependency)
	ErrPromptRequired                       = contract.ErrPromptRequired
	ErrPromptTooLong                        = contract.ErrPromptTooLong
	ErrGenerationOutOfScope                 = contract.ErrGenerationOutOfScope
	ErrGenerationAnalysisRequired           = contract.ErrGenerationAnalysisRequired
	ErrGenerationBriefRequired              = contract.ErrGenerationBriefRequired
	ErrGenerationAwaitingClarification      = contract.ErrGenerationAwaitingClarification
	ErrGenerationNotAwaitingClarification   = contract.ErrGenerationNotAwaitingClarification
	ErrClarificationAlreadySubmitted        = contract.ErrClarificationAlreadySubmitted
	ErrGenerationNotCompleted               = contract.ErrGenerationNotCompleted
	ErrGenerationNotRetryable               = contract.ErrGenerationNotRetryable
	ErrGenerationStructureRetryNotAllowed   = contract.ErrGenerationStructureRetryNotAllowed
	ErrGenerationStructureRetryStepMismatch = contract.ErrGenerationStructureRetryStepMismatch
	ErrMissingGeneratedCourse               = errors.New("AI generator returned no course")
	ErrMissingGeneratedModules              = errors.New("AI generator returned no modules")
	ErrMissingGeneratedLessons              = errors.New("AI generator returned no lessons")
	ErrMissingGeneratedContent              = errors.New("AI generator returned no lesson content")
)

type CourseGeneratorConfig struct {
	StatusURLFormat        string
	JobStatusURLFormat     string
	ResultURLFormat        string
	ClarificationURLFormat string
	MaxActivePerUser       int64
	MaxDailyPerUser        int64
	MaxPendingJobs         int64
}

type CourseGeneratorService struct {
	ai     contract.CourseAIGenerator
	uow    contract.UnitOfWork
	clock  contract.Clock
	config CourseGeneratorConfig
}

func NewCourseGeneratorService(
	ai contract.CourseAIGenerator,
	uow contract.UnitOfWork,
	clock contract.Clock,
	config CourseGeneratorConfig,
) *CourseGeneratorService {
	if config.MaxActivePerUser == 0 {
		config.MaxActivePerUser = 2
	}
	if config.MaxDailyPerUser == 0 {
		config.MaxDailyPerUser = 10
	}
	if config.MaxPendingJobs == 0 {
		config.MaxPendingJobs = 500
	}
	return &CourseGeneratorService{
		ai:     ai,
		uow:    uow,
		clock:  clock,
		config: config,
	}
}

func (s *CourseGeneratorService) enforceGenerationAdmission(ctx context.Context, repository contract.GenerationRequestRepository, owner string) error {
	usage, err := repository.GetGenerationAdmissionUsage(ctx, owner, s.now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if s.config.MaxActivePerUser > 0 && usage.ActiveRequests >= s.config.MaxActivePerUser {
		return contract.ErrGenerationActiveLimitExceeded
	}
	if s.config.MaxDailyPerUser > 0 && usage.DailyRequests >= s.config.MaxDailyPerUser {
		return contract.ErrGenerationDailyLimitExceeded
	}
	if s.config.MaxPendingJobs > 0 && usage.PendingJobs >= s.config.MaxPendingJobs {
		return contract.ErrGenerationQueueSaturated
	}
	return nil
}

func (s *CourseGeneratorService) StartFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	return s.enqueueFullCourseGeneration(ctx, params)
}
func (s *CourseGeneratorService) GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (contract.GenerationStatus, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStatus{}, err
	}
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationStatus{}, err
	}

	var status contract.GenerationStatus
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedGenerationRequest, requestID, owner); err != nil {
			return err
		}
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
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

func courseRecoveryStatus(course domain.Course) domain.CourseGenerationStatus {
	if hasAllLessonPlans(course) {
		for _, module := range course.Modules {
			for _, lesson := range module.Lessons {
				if lesson.HasContent() {
					return domain.CourseStatusContentGenerating
				}
			}
		}
		return domain.CourseStatusLessonsGenerated
	}
	return domain.CourseStatusStructureGenerated
}

func (s *CourseGeneratorService) runPromptAnalysis(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error) {
	request, err := s.updateRequestProgress(ctx, request.ID, stepAnalysis, 5)
	if err != nil {
		return domain.GenerationRequest{}, err
	}

	analysis, err := s.ai.AnalyzePrompt(ctx, contract.AnalysisInput{Prompt: request.InitialUserPrompt})
	if err != nil {
		return domain.GenerationRequest{}, fmt.Errorf("analyze prompt: %w", err)
	}

	request, err = s.persistAnalysis(ctx, request.ID, analysis.Summary, analysis.Raw)
	if err != nil {
		return domain.GenerationRequest{}, err
	}

	return request, nil
}

func (s *CourseGeneratorService) loadGenerationRequest(ctx context.Context, requestID uuid.UUID) (domain.GenerationRequest, error) {
	var request domain.GenerationRequest
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedRequest, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		request = loadedRequest
		return nil
	})
	return request, err
}

func (s *CourseGeneratorService) updateRequestProgress(ctx context.Context, requestID uuid.UUID, step string, percent int) (domain.GenerationRequest, error) {
	var updatedRequest domain.GenerationRequest
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}

		now := s.now()
		if request.PipelineStatus == domain.PipelineStatusQueued {
			if err := request.MarkRunning(step, now); err != nil {
				return err
			}
		}
		if err := request.UpdateProgress(step, percent, now); err != nil {
			return err
		}

		updatedRequest, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, request)
		return err
	})
	return updatedRequest, err
}

func (s *CourseGeneratorService) persistAnalysis(ctx context.Context, requestID uuid.UUID, summary domain.AnalysisSummary, rawOutput json.RawMessage) (domain.GenerationRequest, error) {
	var updatedRequest domain.GenerationRequest
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}

		now := s.now()
		if err := request.ApplyAnalysis(summary, now); err != nil {
			return err
		}
		request.RawAnalysisOutput = jsonutil.Clone(rawOutput)
		if err := request.UpdateProgress(stepAnalysisCompleted, 25, now); err != nil {
			return err
		}

		updatedRequest, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, request)
		return err
	})
	return updatedRequest, err
}

func (s *CourseGeneratorService) persistArchitecture(ctx context.Context, request domain.GenerationRequest, generatedCourse domain.Course, rawOutput json.RawMessage) (domain.Course, error) {
	course, err := s.normalizeGeneratedCourse(request, generatedCourse)
	if err != nil {
		return domain.Course{}, err
	}
	course.RawArchitectureOutput = jsonutil.Clone(rawOutput)

	modules, err := s.normalizeGeneratedModules(course.ID, course.Modules)
	if err != nil {
		return domain.Course{}, err
	}
	course.Modules = modules

	var savedCourse domain.Course
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		persistedCourse, err := repositories.Courses().SaveCourse(ctx, course)
		if err != nil {
			return err
		}

		savedModules, err := repositories.Modules().SaveModules(ctx, modules)
		if err != nil {
			return err
		}

		persistedCourse.Modules = savedModules
		savedCourse = persistedCourse
		return nil
	})
	return savedCourse, err
}

func (s *CourseGeneratorService) persistLessonPlan(ctx context.Context, module domain.Module, lessons []domain.Lesson) ([]domain.Lesson, error) {
	var savedLessons []domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if _, err := repositories.Modules().UpdateModule(ctx, module); err != nil {
			return err
		}

		persistedLessons, err := repositories.Lessons().SaveLessons(ctx, lessons)
		if err != nil {
			return err
		}
		savedLessons = persistedLessons
		return nil
	})
	return savedLessons, err
}

func (s *CourseGeneratorService) persistLessonContent(ctx context.Context, lesson domain.Lesson) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if _, err := repositories.Lessons().ReplaceLessonContent(ctx, lesson); err != nil {
			return err
		}
		if _, err := repositories.Exercises().SaveExercises(ctx, lesson.Exercises); err != nil {
			return err
		}
		_, err := repositories.Quizzes().SaveQuizzes(ctx, lesson.Quizzes)
		return err
	})
}

func (s *CourseGeneratorService) transitionCourse(ctx context.Context, course domain.Course, mutate func(course *domain.Course) error) (domain.Course, error) {
	var updatedCourse domain.Course
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := mutate(&course); err != nil {
			return err
		}
		course.UpdatedAt = s.now()

		var err error
		updatedCourse, err = repositories.Courses().UpdateCourse(ctx, course)
		return err
	})
	return updatedCourse, err
}

func (s *CourseGeneratorService) completeRequest(ctx context.Context, requestID uuid.UUID) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		if err := request.MarkCompleted(s.now()); err != nil {
			return err
		}
		_, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, request)
		return err
	})
}

func (s *CourseGeneratorService) markPipelineFailed(ctx context.Context, requestID uuid.UUID, cause error) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		now := s.now()
		message := failureMessage(cause)

		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		if !request.PipelineStatus.IsTerminal() {
			if err := request.MarkFailed(message, now); err != nil {
				return err
			}
			if _, err := repositories.GenerationRequests().UpdateGenerationRequest(ctx, request); err != nil {
				return err
			}
		}

		course, err := repositories.Courses().FindCourseStateByRequestID(ctx, request.ID)
		if err != nil {
			if errors.Is(err, contract.ErrCourseNotFound) {
				return nil
			}
			return err
		}
		if !course.Status.IsTerminal() {
			if err := course.MarkFailed(); err != nil {
				return err
			}
			course.UpdatedAt = now
			if _, err := repositories.Courses().UpdateCourse(ctx, course); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *CourseGeneratorService) prepareStructureRetry(ctx context.Context, requestID uuid.UUID) (domain.GenerationRequest, error) {
	var updatedRequest domain.GenerationRequest
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		if request.PipelineStatus != domain.PipelineStatusFailed {
			return ErrGenerationStructureRetryNotAllowed
		}
		if !isStructureRetryableStep(request.CurrentStep) {
			return ErrGenerationStructureRetryStepMismatch
		}

		if err := repositories.Courses().DeleteCourseByRequestID(ctx, request.ID); err != nil && !errors.Is(err, contract.ErrCourseNotFound) {
			return err
		}

		now := s.now()
		if err := request.RestartFromFailure(stepAnalysisCompleted, 25, now); err != nil {
			return err
		}

		updatedRequest, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, request)
		return err
	})
	return updatedRequest, err
}

func isStructureRetryableStep(step *string) bool {
	if step == nil {
		return false
	}

	switch strings.TrimSpace(*step) {
	case stepArchitecture, stepLessonPlan, "structure_generation":
		return true
	default:
		return false
	}
}
func (s *CourseGeneratorService) loadCourseByID(ctx context.Context, courseID uuid.UUID) (domain.Course, error) {
	var course domain.Course
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, courseID)
		if err != nil {
			return err
		}
		course = loadedCourse
		return nil
	})
	return course, err
}

func (s *CourseGeneratorService) loadLessonGenerationContext(ctx context.Context, lessonID uuid.UUID) (domain.Course, domain.Module, domain.Lesson, error) {
	var course domain.Course
	var module domain.Module
	var lesson domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedLesson, err := repositories.Lessons().FindLessonByID(ctx, lessonID)
		if err != nil {
			return err
		}
		loadedModule, err := repositories.Modules().FindModuleByID(ctx, loadedLesson.ModuleID)
		if err != nil {
			return err
		}
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, loadedModule.CourseID)
		if err != nil {
			return err
		}

		lesson = loadedLesson
		module = loadedModule
		course = loadedCourse
		return nil
	})
	return course, module, lesson, err
}

func (s *CourseGeneratorService) loadModuleGenerationContext(ctx context.Context, moduleID uuid.UUID) (domain.Course, domain.Module, error) {
	var course domain.Course
	var module domain.Module
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedModule, err := repositories.Modules().FindModuleByID(ctx, moduleID)
		if err != nil {
			return err
		}
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, loadedModule.CourseID)
		if err != nil {
			return err
		}

		module = loadedModule
		course = loadedCourse
		return nil
	})
	return course, module, err
}

func (s *CourseGeneratorService) completeCourseIfReady(ctx context.Context, courseID uuid.UUID) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		isComplete, err := repositories.Courses().IsCourseContentComplete(ctx, courseID)
		if err != nil || !isComplete {
			return err
		}

		course, err := repositories.Courses().FindCourseStateByID(ctx, courseID)
		if err != nil {
			return err
		}
		if course.Status == domain.CourseStatusCompleted {
			return nil
		}
		if course.Status != domain.CourseStatusContentGenerating {
			if err := course.MarkContentGenerating(); err != nil {
				return err
			}
		}
		if err := course.MarkCompletedWithValidatedContent(isComplete); err != nil {
			return err
		}
		course.UpdatedAt = s.now()
		_, err = repositories.Courses().UpdateCourse(ctx, course)
		return err
	})
}
func (s *CourseGeneratorService) generationStarted(request domain.GenerationRequest, job domain.GenerationJob) contract.GenerationStarted {
	return contract.GenerationStarted{
		JobID:        job.ID,
		RequestID:    request.ID,
		Status:       request.PipelineStatus,
		JobStatus:    job.Status,
		StatusURL:    formatGenerationURL(s.config.StatusURLFormat, "/api/generations/%s/status", request.ID),
		JobStatusURL: formatGenerationURL(s.config.JobStatusURLFormat, "/api/generation-jobs/%s", job.ID),
		ResultURL:    formatGenerationURL(s.config.ResultURLFormat, "/api/generations/%s/result", request.ID),
	}
}

func (s *CourseGeneratorService) validateDependencies() error {
	if s == nil || s.ai == nil || s.uow == nil {
		return ErrCourseGeneratorDependency
	}
	return nil
}

func (s *CourseGeneratorService) now() time.Time {
	if s == nil || s.clock == nil {
		return time.Now()
	}
	return s.clock.Now()
}

func isEmptyGeneratedCourse(course domain.Course) bool {
	return course.ID == uuid.Nil &&
		course.RequestID == uuid.Nil &&
		strings.TrimSpace(course.Title) == "" &&
		strings.TrimSpace(course.Synopsis) == "" &&
		len(course.Modules) == 0
}

func failureMessage(err error) string {
	return "generation failed; retry the operation or contact support with the request id"
}

func normalizeStructureParams(params contract.GenerateStructureParams) (contract.GenerateStructureParams, error) {
	if params.RequestID == uuid.Nil {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}

	params.Title = strings.TrimSpace(params.Title)
	if params.Title == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: title", domain.ErrBlankField)
	}
	params.Synopsis = strings.TrimSpace(params.Synopsis)
	if params.Synopsis == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: synopsis", domain.ErrBlankField)
	}
	if params.CurrentLevel == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: current level", domain.ErrGenerationBriefIncomplete)
	}
	if err := params.CurrentLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if params.CurrentLevel == domain.LevelUnknown {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: current level", domain.ErrGenerationBriefIncomplete)
	}
	if params.TargetLevel == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: target level", domain.ErrGenerationBriefIncomplete)
	}
	if err := params.TargetLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if params.TargetLevel == domain.LevelUnknown {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: target level", domain.ErrGenerationBriefIncomplete)
	}
	if params.Language == "" {
		params.Language = domain.CourseLanguageFR
	}
	if err := params.Language.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}

	params.Goals = textutil.TrimNonBlank(params.Goals)
	if len(params.Goals) == 0 {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: goals", domain.ErrInvalidCollection)
	}

	return params, nil
}

func requestHasAnalysis(request domain.GenerationRequest) bool {
	return request.AnalysisCompletedAt != nil
}
func formatGenerationURL(format string, fallbackFormat string, requestID uuid.UUID) string {
	format = strings.TrimSpace(format)
	if format == "" {
		return fmt.Sprintf(fallbackFormat, requestID.String())
	}
	if strings.Contains(format, "%s") {
		return fmt.Sprintf(format, requestID.String())
	}
	return strings.TrimRight(format, "/") + "/" + requestID.String()
}
