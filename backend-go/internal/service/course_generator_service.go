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
	stepAnalysis          = "analysis"
	stepAnalysisCompleted = "analysis_completed"
	stepArchitecture      = "architecture_generation"
	stepLessonPlan        = "lesson_plan_generation"
	stepLessonContent     = "lesson_content_generation"
	stepGenerationSuccess = "generation_completed"
)

var (
	ErrCourseGeneratorDependency            = errors.New("course generator service dependency is missing")
	ErrPromptRequired                       = errors.New("generation prompt is required")
	ErrGenerationOutOfScope                 = errors.New("generation request is out of scope")
	ErrGenerationAnalysisRequired           = errors.New("generation request must be analyzed before structure generation")
	ErrGenerationNotCompleted               = errors.New("generation is not completed")
	ErrGenerationNotRetryable               = errors.New("only failed generations can be retried")
	ErrGenerationStructureRetryNotAllowed   = errors.New("only failed structure generations can be retried")
	ErrGenerationStructureRetryStepMismatch = errors.New("generation failure is not related to structure generation")
	ErrMissingGeneratedCourse               = errors.New("AI generator returned no course")
	ErrMissingGeneratedModules              = errors.New("AI generator returned no modules")
	ErrMissingGeneratedLessons              = errors.New("AI generator returned no lessons")
	ErrMissingGeneratedContent              = errors.New("AI generator returned no lesson content")
)

type CourseGeneratorConfig struct {
	StatusURLFormat    string
	JobStatusURLFormat string
	ResultURLFormat    string
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
	return &CourseGeneratorService{
		ai:     ai,
		uow:    uow,
		clock:  clock,
		config: config,
	}
}

func (s *CourseGeneratorService) AnalyzePrompt(ctx context.Context, params contract.AnalyzePromptParams) (contract.GenerationAnalysisResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationAnalysisResult{}, err
	}

	prompt := strings.TrimSpace(params.Prompt)
	if prompt == "" {
		return contract.GenerationAnalysisResult{}, ErrPromptRequired
	}

	request, err := domain.NewGenerationRequestAt(prompt, s.now())
	if err != nil {
		return contract.GenerationAnalysisResult{}, err
	}

	if err := s.persistNewRequest(ctx, request); err != nil {
		return contract.GenerationAnalysisResult{}, err
	}

	analyzedRequest, err := s.runPromptAnalysis(ctx, request)
	if err != nil {
		if failErr := s.markPipelineFailed(ctx, request.ID, err); failErr != nil {
			return contract.GenerationAnalysisResult{}, errors.Join(err, failErr)
		}
		return contract.GenerationAnalysisResult{}, err
	}
	request = analyzedRequest

	if request.IsOutOfScope {
		if err := s.completeRequest(ctx, request.ID); err != nil {
			return contract.GenerationAnalysisResult{}, err
		}
		request, err = s.loadGenerationRequest(ctx, request.ID)
		if err != nil {
			return contract.GenerationAnalysisResult{}, err
		}
	}

	return contract.GenerationAnalysisResult{Request: request}, nil
}

func (s *CourseGeneratorService) StartFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	return s.enqueueFullCourseGeneration(ctx, params)
}

func (s *CourseGeneratorService) GenerateCourseStructure(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationResult{}, err
	}

	params, err := normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	request, err := s.loadGenerationRequest(ctx, params.RequestID)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	if request.IsOutOfScope {
		return contract.GenerationResult{}, ErrGenerationOutOfScope
	}
	if !requestHasAnalysis(request) {
		return contract.GenerationResult{}, ErrGenerationAnalysisRequired
	}

	_, course, err := s.runStructurePipeline(ctx, request, params)
	if err != nil {
		if failErr := s.markPipelineFailed(ctx, request.ID, err); failErr != nil {
			return contract.GenerationResult{}, errors.Join(err, failErr)
		}
		return contract.GenerationResult{}, err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	if err := s.completeRequest(ctx, request.ID); err != nil {
		return contract.GenerationResult{}, err
	}

	completedRequest, err := s.loadGenerationRequest(ctx, request.ID)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	persistedCourse, err := s.loadCourseByID(ctx, course.ID)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	return contract.GenerationResult{Request: completedRequest, Course: persistedCourse}, nil
}

func (s *CourseGeneratorService) RetryCourseStructure(ctx context.Context, params contract.GenerateStructureParams) (contract.GenerationResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationResult{}, err
	}

	params, err := normalizeStructureParams(params)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	request, err := s.loadGenerationRequest(ctx, params.RequestID)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	if request.PipelineStatus != domain.PipelineStatusFailed {
		return contract.GenerationResult{}, ErrGenerationStructureRetryNotAllowed
	}
	if !isStructureRetryableStep(request.CurrentStep) {
		return contract.GenerationResult{}, ErrGenerationStructureRetryStepMismatch
	}
	if request.IsOutOfScope {
		return contract.GenerationResult{}, ErrGenerationOutOfScope
	}
	if !requestHasAnalysis(request) {
		return contract.GenerationResult{}, ErrGenerationAnalysisRequired
	}

	request, err = s.prepareStructureRetry(ctx, request.ID)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	_, course, err := s.runStructurePipeline(ctx, request, params)
	if err != nil {
		if failErr := s.markPipelineFailed(ctx, request.ID, err); failErr != nil {
			return contract.GenerationResult{}, errors.Join(err, failErr)
		}
		return contract.GenerationResult{}, err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	if err := s.completeRequest(ctx, request.ID); err != nil {
		return contract.GenerationResult{}, err
	}

	completedRequest, err := s.loadGenerationRequest(ctx, request.ID)
	if err != nil {
		return contract.GenerationResult{}, err
	}
	persistedCourse, err := s.loadCourseByID(ctx, course.ID)
	if err != nil {
		return contract.GenerationResult{}, err
	}

	return contract.GenerationResult{Request: completedRequest, Course: persistedCourse}, nil
}
func (s *CourseGeneratorService) GenerateLessonContent(ctx context.Context, lessonID uuid.UUID) (domain.Lesson, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.Lesson{}, err
	}

	course, module, lesson, err := s.loadLessonGenerationContext(ctx, lessonID)
	if err != nil {
		return domain.Lesson{}, err
	}

	course, err = s.ensureCourseContentGenerating(ctx, course)
	if err != nil {
		return domain.Lesson{}, err
	}

	output, err := s.ai.GenerateLessonContent(ctx, contract.LessonContentInput{Course: course, Module: module, Lesson: lesson})
	if err != nil {
		return domain.Lesson{}, fmt.Errorf("generate content for lesson %s: %w", lesson.ID, err)
	}

	lessonWithContent, err := s.attachGeneratedContent(lesson, output)
	if err != nil {
		return domain.Lesson{}, err
	}
	if err := s.persistLessonContent(ctx, lessonWithContent); err != nil {
		return domain.Lesson{}, err
	}
	if err := s.completeCourseIfReady(ctx, course.ID); err != nil {
		return domain.Lesson{}, err
	}

	return s.loadLessonByID(ctx, lesson.ID)
}

func (s *CourseGeneratorService) GenerateModuleLessonContents(ctx context.Context, moduleID uuid.UUID) (domain.Module, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.Module{}, err
	}

	course, module, err := s.loadModuleGenerationContext(ctx, moduleID)
	if err != nil {
		return domain.Module{}, err
	}
	if len(module.Lessons) == 0 {
		return domain.Module{}, ErrMissingGeneratedLessons
	}

	course, err = s.ensureCourseContentGenerating(ctx, course)
	if err != nil {
		return domain.Module{}, err
	}

	for _, lesson := range module.Lessons {
		output, err := s.ai.GenerateLessonContent(ctx, contract.LessonContentInput{Course: course, Module: module, Lesson: lesson})
		if err != nil {
			return domain.Module{}, fmt.Errorf("generate content for lesson %s: %w", lesson.ID, err)
		}

		lessonWithContent, err := s.attachGeneratedContent(lesson, output)
		if err != nil {
			return domain.Module{}, err
		}
		if err := s.persistLessonContent(ctx, lessonWithContent); err != nil {
			return domain.Module{}, err
		}
	}

	if err := s.completeCourseIfReady(ctx, course.ID); err != nil {
		return domain.Module{}, err
	}

	return s.loadModuleByID(ctx, module.ID)
}
func (s *CourseGeneratorService) GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (contract.GenerationStatus, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStatus{}, err
	}

	var status contract.GenerationStatus
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		var err error
		status, err = repositories.GenerationRequests().FindGenerationStatusByID(ctx, requestID)
		return err
	})
	if err != nil {
		return contract.GenerationStatus{}, err
	}
	return status, nil
}

func (s *CourseGeneratorService) GetGenerationResult(ctx context.Context, requestID uuid.UUID) (contract.GenerationResult, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationResult{}, err
	}

	var result contract.GenerationResult
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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

func (s *CourseGeneratorService) RetryFullCourseGeneration(ctx context.Context, requestID uuid.UUID) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}

	request, err := s.loadGenerationRequest(ctx, requestID)
	if err != nil {
		return contract.GenerationStarted{}, err
	}
	if request.PipelineStatus != domain.PipelineStatusFailed {
		return contract.GenerationStarted{}, ErrGenerationNotRetryable
	}

	return s.StartFullCourseGeneration(ctx, contract.StartGenerationParams{Prompt: request.InitialUserPrompt})
}

func (s *CourseGeneratorService) runFullPipeline(ctx context.Context, request domain.GenerationRequest) error {
	request, err := s.runPromptAnalysis(ctx, request)
	if err != nil {
		return err
	}
	if request.IsOutOfScope {
		return ErrGenerationOutOfScope
	}

	structureParams, err := structureParamsFromAnalysis(request)
	if err != nil {
		return err
	}

	request, course, err := s.runStructurePipeline(ctx, request, structureParams)
	if err != nil {
		return err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepLessonContent, 75)
	if err != nil {
		return err
	}

	course, err = s.ensureCourseContentGenerating(ctx, course)
	if err != nil {
		return err
	}

	if err := s.generateAndPersistLessonContents(ctx, course); err != nil {
		return err
	}

	if err := s.completeCourseIfReady(ctx, course.ID); err != nil {
		return err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95)
	if err != nil {
		return err
	}

	return s.completeRequest(ctx, request.ID)
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

func (s *CourseGeneratorService) runStructurePipeline(ctx context.Context, request domain.GenerationRequest, params contract.GenerateStructureParams) (domain.GenerationRequest, domain.Course, error) {
	request, course, err := s.generateArchitectureJob(ctx, request, params)
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepLessonPlan, 60)
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}

	course, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
		return course.MarkLessonsGenerating()
	})
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}

	course, err = s.generateAndPersistLessonPlans(ctx, course)
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}

	course, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
		return course.MarkLessonsGenerated()
	})
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}

	return request, course, nil
}

func (s *CourseGeneratorService) persistNewRequest(ctx context.Context, request domain.GenerationRequest) error {
	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		_, err := repositories.GenerationRequests().SaveGenerationRequest(ctx, request)
		return err
	})
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

func (s *CourseGeneratorService) generateAndPersistLessonPlans(ctx context.Context, course domain.Course) (domain.Course, error) {
	modules := make([]domain.Module, 0, len(course.Modules))
	for _, module := range course.Modules {
		if module.ID == uuid.Nil {
			return domain.Course{}, fmt.Errorf("%w: module id before lesson plan generation", domain.ErrBlankField)
		}

		output, err := s.ai.GenerateLessonPlan(ctx, contract.LessonPlanInput{Course: course, Module: module})
		if err != nil {
			return domain.Course{}, fmt.Errorf("generate lesson plan for module %s: %w", module.ID, err)
		}

		lessons, err := s.normalizeGeneratedLessons(module.ID, output.Lessons)
		if err != nil {
			return domain.Course{}, err
		}

		module.RawLessonsPlanOutput = jsonutil.Clone(output.Raw)
		module.UpdatedAt = s.now()

		savedLessons, err := s.persistLessonPlan(ctx, module, lessons)
		if err != nil {
			return domain.Course{}, err
		}

		module.Lessons = savedLessons
		modules = append(modules, module)
	}

	course.Modules = modules
	return course, nil
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

func (s *CourseGeneratorService) generateAndPersistLessonContents(ctx context.Context, course domain.Course) error {
	for _, module := range course.Modules {
		for _, lesson := range module.Lessons {
			output, err := s.ai.GenerateLessonContent(ctx, contract.LessonContentInput{Course: course, Module: module, Lesson: lesson})
			if err != nil {
				return fmt.Errorf("generate content for lesson %s: %w", lesson.ID, err)
			}

			lessonWithContent, err := s.attachGeneratedContent(lesson, output)
			if err != nil {
				return err
			}
			if err := s.persistLessonContent(ctx, lessonWithContent); err != nil {
				return err
			}
		}
	}
	return nil
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
func (s *CourseGeneratorService) normalizeGeneratedCourse(request domain.GenerationRequest, generatedCourse domain.Course) (domain.Course, error) {
	if isEmptyGeneratedCourse(generatedCourse) {
		return domain.Course{}, ErrMissingGeneratedCourse
	}

	now := s.now()
	modules := generatedCourse.Modules
	if generatedCourse.ID == uuid.Nil {
		course, err := domain.NewCourseAt(domain.NewCourseParams{
			RequestID:               request.ID,
			Language:                generatedCourse.Language,
			InitialUserPrompt:       textutil.FirstNonBlank(generatedCourse.InitialUserPrompt, request.InitialUserPrompt),
			Title:                   generatedCourse.Title,
			Synopsis:                generatedCourse.Synopsis,
			TargetAudience:          generatedCourse.TargetAudience,
			CurrentLevel:            generatedCourse.CurrentLevel,
			TargetLevel:             generatedCourse.TargetLevel,
			Prerequisites:           generatedCourse.Prerequisites,
			Goals:                   generatedCourse.Goals,
			AcquiredSkills:          generatedCourse.AcquiredSkills,
			FinalProjectTitle:       generatedCourse.FinalProjectTitle,
			FinalProjectDescription: generatedCourse.FinalProjectDescription,
			FinalProjectConstraints: generatedCourse.FinalProjectConstraints,
		}, now)
		if err != nil {
			return domain.Course{}, err
		}
		generatedCourse = course
		generatedCourse.Modules = modules
	}

	if generatedCourse.RequestID == uuid.Nil {
		generatedCourse.RequestID = request.ID
	}
	if generatedCourse.RequestID != request.ID {
		return domain.Course{}, fmt.Errorf("%w: course request id does not match generation request id", domain.ErrInvalidCollection)
	}
	if strings.TrimSpace(generatedCourse.InitialUserPrompt) == "" {
		generatedCourse.InitialUserPrompt = request.InitialUserPrompt
	}
	if generatedCourse.CreatedAt.IsZero() {
		generatedCourse.CreatedAt = now
	}
	if generatedCourse.UpdatedAt.IsZero() {
		generatedCourse.UpdatedAt = now
	}
	if generatedCourse.Status == "" || generatedCourse.Status == domain.CourseStatusAnalysisPending {
		generatedCourse.Status = domain.CourseStatusAnalysisCompleted
	}

	switch generatedCourse.Status {
	case domain.CourseStatusAnalysisCompleted:
		if err := generatedCourse.MarkArchitectureGenerating(); err != nil {
			return domain.Course{}, err
		}
		generatedCourse.UpdatedAt = now
		if err := generatedCourse.MarkArchitectureGenerated(); err != nil {
			return domain.Course{}, err
		}
	case domain.CourseStatusArchitectureGenerating:
		if err := generatedCourse.MarkArchitectureGenerated(); err != nil {
			return domain.Course{}, err
		}
	case domain.CourseStatusStructureGenerated:
		// Already at the expected persistence boundary.
	default:
		return domain.Course{}, fmt.Errorf("%w: unexpected generated course status %s", domain.ErrInvalidCourseStatus, generatedCourse.Status)
	}
	generatedCourse.UpdatedAt = now
	generatedCourse.Modules = modules

	return generatedCourse, generatedCourse.ValidateCourseOnly()
}

func (s *CourseGeneratorService) normalizeGeneratedModules(courseID uuid.UUID, modules []domain.Module) ([]domain.Module, error) {
	if len(modules) == 0 {
		return nil, ErrMissingGeneratedModules
	}

	now := s.now()
	normalizedModules := make([]domain.Module, 0, len(modules))
	for _, module := range modules {
		if module.ID == uuid.Nil {
			newModule, err := domain.NewModuleAt(domain.NewModuleParams{
				CourseID:          courseID,
				Order:             module.Order,
				Title:             module.Title,
				Description:       module.Description,
				KeyLearningPoints: module.KeyLearningPoints,
			}, now)
			if err != nil {
				return nil, err
			}
			module = newModule
		}
		if module.CourseID == uuid.Nil {
			module.CourseID = courseID
		}
		if module.CourseID != courseID {
			return nil, fmt.Errorf("%w: module course id does not match course id", domain.ErrInvalidCollection)
		}
		if module.CreatedAt.IsZero() {
			module.CreatedAt = now
		}
		if module.UpdatedAt.IsZero() {
			module.UpdatedAt = now
		}
		module.Lessons = nil
		if err := module.Validate(); err != nil {
			return nil, err
		}
		normalizedModules = append(normalizedModules, module)
	}
	return normalizedModules, nil
}

func (s *CourseGeneratorService) normalizeGeneratedLessons(moduleID uuid.UUID, lessons []domain.Lesson) ([]domain.Lesson, error) {
	if len(lessons) == 0 {
		return nil, ErrMissingGeneratedLessons
	}

	now := s.now()
	normalizedLessons := make([]domain.Lesson, 0, len(lessons))
	for _, lesson := range lessons {
		if lesson.ID == uuid.Nil {
			newLesson, err := domain.NewLessonAt(domain.NewLessonParams{
				ModuleID:                 moduleID,
				Order:                    lesson.Order,
				Title:                    lesson.Title,
				Type:                     lesson.Type,
				EstimatedDurationMinutes: lesson.EstimatedDurationMinutes,
				LearningGoal:             lesson.LearningGoal,
				RequiresDiagram:          lesson.RequiresDiagram,
				TechnicalKeywords:        lesson.TechnicalKeywords,
			}, now)
			if err != nil {
				return nil, err
			}
			lesson = newLesson
		}
		if lesson.ModuleID == uuid.Nil {
			lesson.ModuleID = moduleID
		}
		if lesson.ModuleID != moduleID {
			return nil, fmt.Errorf("%w: lesson module id does not match module id", domain.ErrInvalidCollection)
		}
		if lesson.CreatedAt.IsZero() {
			lesson.CreatedAt = now
		}
		if lesson.UpdatedAt.IsZero() {
			lesson.UpdatedAt = now
		}
		lesson.ContentMarkdown = nil
		if err := lesson.Validate(); err != nil {
			return nil, err
		}
		normalizedLessons = append(normalizedLessons, lesson)
	}
	return normalizedLessons, nil
}

func (s *CourseGeneratorService) attachGeneratedContent(lesson domain.Lesson, output contract.LessonContentOutput) (domain.Lesson, error) {
	content := strings.TrimSpace(output.ContentMarkdown)
	if content == "" && output.Lesson.ContentMarkdown != nil {
		content = strings.TrimSpace(*output.Lesson.ContentMarkdown)
	}
	if content == "" {
		return domain.Lesson{}, ErrMissingGeneratedContent
	}
	if err := domain.ValidateLessonActivitiesForType(lesson.Type, output.Exercises, output.Quizzes); err != nil {
		return domain.Lesson{}, err
	}

	if err := lesson.AttachContent(content); err != nil {
		return domain.Lesson{}, err
	}
	lesson.RawContentOutput = jsonutil.Clone(output.Raw)
	lesson.Exercises = attachRawOutputToExercises(output.Exercises, output.Raw)
	lesson.Quizzes = attachRawOutputToQuizzes(output.Quizzes, output.Raw)
	lesson.UpdatedAt = s.now()
	return lesson, nil
}

func attachRawOutputToExercises(exercises []domain.Exercise, rawOutput json.RawMessage) []domain.Exercise {
	if len(exercises) == 0 {
		return nil
	}

	withRawOutput := make([]domain.Exercise, 0, len(exercises))
	for _, exercise := range exercises {
		exercise.RawAIOutput = jsonutil.Clone(rawOutput)
		withRawOutput = append(withRawOutput, exercise)
	}
	return withRawOutput
}

func attachRawOutputToQuizzes(quizzes []domain.Quiz, rawOutput json.RawMessage) []domain.Quiz {
	if len(quizzes) == 0 {
		return nil
	}

	withRawOutput := make([]domain.Quiz, 0, len(quizzes))
	for _, quiz := range quizzes {
		quiz.RawAIOutput = jsonutil.Clone(rawOutput)
		withRawOutput = append(withRawOutput, quiz)
	}
	return withRawOutput
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

func (s *CourseGeneratorService) loadModuleByID(ctx context.Context, moduleID uuid.UUID) (domain.Module, error) {
	var module domain.Module
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedModule, err := repositories.Modules().FindModuleByID(ctx, moduleID)
		if err != nil {
			return err
		}
		module = loadedModule
		return nil
	})
	return module, err
}

func (s *CourseGeneratorService) loadLessonByID(ctx context.Context, lessonID uuid.UUID) (domain.Lesson, error) {
	var lesson domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedLesson, err := repositories.Lessons().FindLessonByID(ctx, lessonID)
		if err != nil {
			return err
		}
		lesson = loadedLesson
		return nil
	})
	return lesson, err
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

func (s *CourseGeneratorService) ensureCourseContentGenerating(ctx context.Context, course domain.Course) (domain.Course, error) {
	if course.Status == domain.CourseStatusCompleted || course.Status == domain.CourseStatusContentGenerating {
		return course, nil
	}
	return s.transitionCourse(ctx, course, func(course *domain.Course) error {
		return course.MarkContentGenerating()
	})
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
	if err == nil {
		return "generation failed"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "generation failed"
	}
	return message
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
		params.CurrentLevel = domain.LevelUnknown
	}
	if err := params.CurrentLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if params.TargetLevel == "" {
		params.TargetLevel = domain.LevelUnknown
	}
	if err := params.TargetLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
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

func structureParamsFromAnalysis(request domain.GenerationRequest) (contract.GenerateStructureParams, error) {
	currentLevel := domain.LevelUnknown
	if request.DetectedCurrentLevel != nil {
		currentLevel = *request.DetectedCurrentLevel
	}
	targetLevel := domain.LevelUnknown
	if request.DetectedTargetLevel != nil {
		targetLevel = *request.DetectedTargetLevel
	}
	language := domain.CourseLanguageFR
	if request.DetectedLanguage != nil {
		language = *request.DetectedLanguage
	}

	goals := []string{request.InitialUserPrompt}
	if request.DetectedGoal != nil && !isUnknownText(*request.DetectedGoal) {
		goals = []string{*request.DetectedGoal}
	}

	return normalizeStructureParams(contract.GenerateStructureParams{
		RequestID:    request.ID,
		Title:        textutil.ValueOr(request.SuggestedTitle, request.InitialUserPrompt),
		Synopsis:     textutil.ValueOr(request.ShortSynopsis, request.InitialUserPrompt),
		CurrentLevel: currentLevel,
		TargetLevel:  targetLevel,
		Goals:        goals,
		Language:     language,
	})
}

func requestHasAnalysis(request domain.GenerationRequest) bool {
	return request.IsOutOfScope ||
		request.ErrorMessage != nil ||
		request.WarningMessage != nil ||
		request.SuggestedTitle != nil ||
		request.ShortSynopsis != nil ||
		request.DetectedCurrentLevel != nil ||
		request.DetectedTargetLevel != nil ||
		request.DetectedGoal != nil ||
		request.DetectedLanguage != nil ||
		len(request.ClarificationQuestions) > 0
}

func isUnknownText(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "" || value == "unknown" || value == "unknow"
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
