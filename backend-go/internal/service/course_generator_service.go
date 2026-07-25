package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

const (
	stepAnalysis          = "analysis"
	stepArchitecture      = "architecture_generation"
	stepLessonPlan        = "lesson_plan_generation"
	stepLessonContent     = "lesson_content_generation"
	stepGenerationSuccess = "generation_completed"
)

var (
	ErrCourseGeneratorDependency = errors.New("course generator service dependency is missing")
	ErrPromptRequired            = errors.New("generation prompt is required")
	ErrGenerationOutOfScope      = errors.New("generation request is out of scope")
	ErrGenerationNotCompleted    = errors.New("generation is not completed")
	ErrGenerationNotRetryable    = errors.New("only failed generations can be retried")
	ErrMissingGeneratedCourse    = errors.New("AI generator returned no course")
	ErrMissingGeneratedModules   = errors.New("AI generator returned no modules")
	ErrMissingGeneratedLessons   = errors.New("AI generator returned no lessons")
	ErrMissingGeneratedContent   = errors.New("AI generator returned no lesson content")
)

type CourseGeneratorConfig struct {
	StatusURLFormat string
	ResultURLFormat string
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

func (s *CourseGeneratorService) StartFullCourseGeneration(ctx context.Context, params contract.StartGenerationParams) (contract.GenerationStarted, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStarted{}, err
	}

	prompt := strings.TrimSpace(params.Prompt)
	if prompt == "" {
		return contract.GenerationStarted{}, ErrPromptRequired
	}

	request, err := domain.NewGenerationRequestAt(prompt, s.now())
	if err != nil {
		return contract.GenerationStarted{}, err
	}

	if err := s.persistNewRequest(ctx, request); err != nil {
		return contract.GenerationStarted{}, err
	}

	started := s.generationStarted(request.ID, request.PipelineStatus)
	if err := s.runFullPipeline(ctx, request); err != nil {
		if failErr := s.markPipelineFailed(ctx, request.ID, err); failErr != nil {
			return started, errors.Join(err, failErr)
		}
		started.Status = domain.PipelineStatusFailed
		return started, err
	}

	completedRequest, err := s.loadGenerationRequest(ctx, request.ID)
	if err != nil {
		return started, err
	}

	return s.generationStarted(completedRequest.ID, completedRequest.PipelineStatus), nil
}

func (s *CourseGeneratorService) GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (contract.GenerationStatus, error) {
	if err := s.validateDependencies(); err != nil {
		return contract.GenerationStatus{}, err
	}

	var status contract.GenerationStatus
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}

		status = contract.GenerationStatus{
			RequestID:       request.ID,
			PipelineStatus:  request.PipelineStatus,
			CurrentStep:     request.CurrentStep,
			ProgressPercent: request.ProgressPercent,
			FailureMessage:  request.FailureMessage,
		}

		course, err := repositories.Courses().FindCourseByRequestID(ctx, request.ID)
		if err != nil {
			if errors.Is(err, contract.ErrCourseNotFound) {
				return nil
			}
			return err
		}

		status.CourseID = &course.ID
		status.CourseStatus = &course.Status
		return nil
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
	request, err := s.updateRequestProgress(ctx, request.ID, stepAnalysis, 5)
	if err != nil {
		return err
	}

	analysis, err := s.ai.AnalyzePrompt(ctx, contract.AnalysisInput{Prompt: request.InitialUserPrompt})
	if err != nil {
		return fmt.Errorf("analyze prompt: %w", err)
	}

	request, err = s.persistAnalysis(ctx, request.ID, analysis.Summary)
	if err != nil {
		return err
	}
	if request.IsOutOfScope {
		return ErrGenerationOutOfScope
	}

	architecture, err := s.ai.GenerateArchitecture(ctx, contract.ArchitectureInput{Request: request})
	if err != nil {
		return fmt.Errorf("generate architecture: %w", err)
	}

	course, err := s.persistArchitecture(ctx, request, architecture.Course)
	if err != nil {
		return err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepLessonPlan, 50)
	if err != nil {
		return err
	}

	course, err = s.transitionCourse(ctx, course.ID, func(course *domain.Course) error {
		return course.MarkLessonsGenerating()
	})
	if err != nil {
		return err
	}

	course, err = s.generateAndPersistLessonPlans(ctx, course)
	if err != nil {
		return err
	}

	course, err = s.transitionCourse(ctx, course.ID, func(course *domain.Course) error {
		return course.MarkLessonsGenerated()
	})
	if err != nil {
		return err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepLessonContent, 75)
	if err != nil {
		return err
	}

	course, err = s.transitionCourse(ctx, course.ID, func(course *domain.Course) error {
		return course.MarkContentGenerating()
	})
	if err != nil {
		return err
	}

	if err := s.generateAndPersistLessonContents(ctx, course); err != nil {
		return err
	}

	if _, err := s.transitionCourse(ctx, course.ID, func(course *domain.Course) error {
		return course.MarkCompleted()
	}); err != nil {
		return err
	}

	request, err = s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95)
	if err != nil {
		return err
	}

	return s.completeRequest(ctx, request.ID)
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

func (s *CourseGeneratorService) persistAnalysis(ctx context.Context, requestID uuid.UUID, summary domain.AnalysisSummary) (domain.GenerationRequest, error) {
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
		if err := request.UpdateProgress(stepArchitecture, 25, now); err != nil {
			return err
		}

		updatedRequest, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, request)
		return err
	})
	return updatedRequest, err
}

func (s *CourseGeneratorService) persistArchitecture(ctx context.Context, request domain.GenerationRequest, generatedCourse domain.Course) (domain.Course, error) {
	course, err := s.normalizeGeneratedCourse(request, generatedCourse)
	if err != nil {
		return domain.Course{}, err
	}

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

		savedModules := make([]domain.Module, 0, len(modules))
		for _, module := range modules {
			savedModule, err := repositories.Modules().SaveModule(ctx, module)
			if err != nil {
				return err
			}
			savedModules = append(savedModules, savedModule)
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
		output, err := s.ai.GenerateLessonPlan(ctx, contract.LessonPlanInput{Course: course, Module: module})
		if err != nil {
			return domain.Course{}, fmt.Errorf("generate lesson plan for module %s: %w", module.ID, err)
		}

		lessons, err := s.normalizeGeneratedLessons(module.ID, output.Lessons)
		if err != nil {
			return domain.Course{}, err
		}

		savedLessons, err := s.persistLessons(ctx, lessons)
		if err != nil {
			return domain.Course{}, err
		}

		module.Lessons = savedLessons
		modules = append(modules, module)
	}

	course.Modules = modules
	return course, nil
}

func (s *CourseGeneratorService) persistLessons(ctx context.Context, lessons []domain.Lesson) ([]domain.Lesson, error) {
	var savedLessons []domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
		_, err := repositories.Lessons().UpdateLesson(ctx, lesson)
		return err
	})
}

func (s *CourseGeneratorService) transitionCourse(ctx context.Context, courseID uuid.UUID, mutate func(course *domain.Course) error) (domain.Course, error) {
	var updatedCourse domain.Course
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		course, err := repositories.Courses().FindCourseByID(ctx, courseID)
		if err != nil {
			return err
		}
		if err := mutate(&course); err != nil {
			return err
		}
		course.UpdatedAt = s.now()

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

		course, err := repositories.Courses().FindCourseByRequestID(ctx, request.ID)
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
			InitialUserPrompt:       firstNonBlank(generatedCourse.InitialUserPrompt, request.InitialUserPrompt),
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

	return generatedCourse, generatedCourse.Validate()
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

	if err := lesson.AttachContent(content); err != nil {
		return domain.Lesson{}, err
	}
	lesson.UpdatedAt = s.now()
	return lesson, nil
}

func (s *CourseGeneratorService) generationStarted(requestID uuid.UUID, status domain.GenerationPipelineStatus) contract.GenerationStarted {
	return contract.GenerationStarted{
		RequestID: requestID,
		Status:    status,
		StatusURL: formatGenerationURL(s.config.StatusURLFormat, "/api/generations/%s/status", requestID),
		ResultURL: formatGenerationURL(s.config.ResultURLFormat, "/api/generations/%s/result", requestID),
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

func firstNonBlank(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
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
