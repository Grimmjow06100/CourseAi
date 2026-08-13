package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) runFullCourseJob(ctx context.Context, requestID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		return nil
	}

	if !requestHasAnalysis(request) {
		request, err = s.runPromptAnalysis(ctx, request)
		if err != nil {
			return err
		}
	}
	if request.IsOutOfScope {
		return s.completeRequestIfNeeded(ctx, request)
	}

	course, err := s.loadCourseByRequestID(ctx, request.ID)
	if errors.Is(err, contract.ErrCourseNotFound) {
		params, paramsErr := structureParamsFromAnalysis(request)
		if paramsErr != nil {
			return paramsErr
		}
		_, course, err = s.generateArchitectureJob(ctx, request, params)
	}
	if err != nil {
		return err
	}

	if err := s.generateMissingLessonPlans(ctx, request.ID, course); err != nil {
		return err
	}
	course, err = s.loadCourseByID(ctx, course.ID)
	if err != nil {
		return err
	}
	if err := s.generateMissingLessonContents(ctx, request.ID, course); err != nil {
		return err
	}

	return s.runFinalizeCourseJob(ctx, request.ID, course.ID)
}

func (s *CourseGeneratorService) runAnalysisJob(ctx context.Context, requestID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if requestHasAnalysis(request) || request.PipelineStatus == domain.PipelineStatusCompleted {
		return nil
	}

	request, err = s.runPromptAnalysis(ctx, request)
	if err != nil {
		return err
	}
	if request.IsOutOfScope {
		return s.completeRequestIfNeeded(ctx, request)
	}
	return nil
}

func (s *CourseGeneratorService) runArchitectureJob(ctx context.Context, requestID uuid.UUID, payload contract.ArchitectureJobPayload) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if request.IsOutOfScope {
		return ErrGenerationOutOfScope
	}
	if !requestHasAnalysis(request) {
		return ErrGenerationAnalysisRequired
	}

	course, err := s.loadCourseByRequestID(ctx, request.ID)
	if errors.Is(err, contract.ErrCourseNotFound) {
		params, paramsErr := architectureParamsFromJob(request, payload)
		if paramsErr != nil {
			return paramsErr
		}
		_, course, err = s.generateArchitectureJob(ctx, request, params)
	} else if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if err := s.generateMissingLessonPlans(ctx, request.ID, course); err != nil {
		return err
	}
	request, err = s.loadGenerationRequest(ctx, request.ID)
	if err != nil {
		return err
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		return nil
	}
	if _, err := s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95); err != nil {
		return err
	}
	return s.completeRequest(ctx, request.ID)
}

func (s *CourseGeneratorService) runLessonPlanJob(ctx context.Context, requestID, moduleID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	course, module, err := s.loadModuleGenerationContext(ctx, moduleID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: module does not belong to generation request", domain.ErrInvalidCollection)
	}
	if len(module.Lessons) > 0 {
		return s.finishLessonPlanPhaseIfReady(ctx, course.ID)
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		return ErrGenerationNotCompleted
	}

	course, err = s.prepareCourseForLessonPlans(ctx, course)
	if err != nil {
		return err
	}
	if _, err := s.updateRequestProgress(ctx, request.ID, stepLessonPlan, 60); err != nil {
		return err
	}

	output, err := s.ai.GenerateLessonPlan(ctx, contract.LessonPlanInput{Course: course, Module: module})
	if err != nil {
		return fmt.Errorf("generate lesson plan for module %s: %w", module.ID, err)
	}
	lessons, err := s.normalizeGeneratedLessons(module.ID, output.Lessons)
	if err != nil {
		return err
	}
	module.RawLessonsPlanOutput = jsonutil.Clone(output.Raw)
	module.UpdatedAt = s.now()
	if _, err := s.persistLessonPlan(ctx, module, lessons); err != nil {
		return err
	}
	return s.finishLessonPlanPhaseIfReady(ctx, course.ID)
}

func (s *CourseGeneratorService) runLessonContentJob(ctx context.Context, requestID, lessonID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	course, _, lesson, err := s.loadLessonGenerationContext(ctx, lessonID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: lesson does not belong to generation request", domain.ErrInvalidCollection)
	}
	if lesson.HasContent() {
		return s.completeCourseIfReady(ctx, course.ID)
	}
	if _, err := s.prepareCourseForLessonContent(ctx, course); err != nil {
		return err
	}
	if request.PipelineStatus != domain.PipelineStatusCompleted {
		if _, err := s.updateRequestProgress(ctx, request.ID, stepLessonContent, 75); err != nil {
			return err
		}
	}
	_, err = s.GenerateLessonContent(ctx, lesson.ID)
	return err
}

func (s *CourseGeneratorService) runModuleContentJob(ctx context.Context, requestID, moduleID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	course, module, err := s.loadModuleGenerationContext(ctx, moduleID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: module does not belong to generation request", domain.ErrInvalidCollection)
	}
	if len(module.Lessons) == 0 {
		return ErrMissingGeneratedLessons
	}
	if !module.HasCompleteLessons() {
		if _, err := s.prepareCourseForLessonContent(ctx, course); err != nil {
			return err
		}
		if request.PipelineStatus != domain.PipelineStatusCompleted {
			if _, err := s.updateRequestProgress(ctx, request.ID, stepLessonContent, 75); err != nil {
				return err
			}
		}
	}
	for _, lesson := range module.Lessons {
		if lesson.HasContent() {
			continue
		}
		if err := s.runLessonContentJob(ctx, request.ID, lesson.ID); err != nil {
			return err
		}
	}
	return s.completeCourseIfReady(ctx, course.ID)
}

func (s *CourseGeneratorService) runFinalizeCourseJob(ctx context.Context, requestID, courseID uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, requestID)
	if err != nil {
		return err
	}
	course, err := s.loadCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: course does not belong to generation request", domain.ErrInvalidCollection)
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		if course.Status != domain.CourseStatusCompleted {
			return ErrGenerationNotCompleted
		}
		return nil
	}

	if err := s.completeCourseIfReady(ctx, course.ID); err != nil {
		return err
	}
	course, err = s.loadCourseByID(ctx, course.ID)
	if err != nil {
		return err
	}
	if course.Status != domain.CourseStatusCompleted {
		return domain.ErrMissingCourseContent
	}
	if _, err := s.updateRequestProgress(ctx, request.ID, stepGenerationSuccess, 95); err != nil {
		return err
	}
	return s.completeRequest(ctx, request.ID)
}

func (s *CourseGeneratorService) generateArchitectureJob(
	ctx context.Context,
	request domain.GenerationRequest,
	params contract.GenerateStructureParams,
) (domain.GenerationRequest, domain.Course, error) {
	request, err := s.updateRequestProgress(ctx, request.ID, stepArchitecture, 35)
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}
	architecture, err := s.ai.GenerateArchitecture(ctx, contract.ArchitectureInput{
		Request:      request,
		Title:        params.Title,
		Synopsis:     params.Synopsis,
		CurrentLevel: params.CurrentLevel,
		TargetLevel:  params.TargetLevel,
		Goals:        params.Goals,
		Language:     params.Language,
	})
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, fmt.Errorf("generate architecture: %w", err)
	}
	course, err := s.persistArchitecture(ctx, request, architecture.Course, architecture.Raw)
	if err != nil {
		return domain.GenerationRequest{}, domain.Course{}, err
	}
	return request, course, nil
}

func (s *CourseGeneratorService) generateMissingLessonPlans(ctx context.Context, requestID uuid.UUID, course domain.Course) error {
	if len(course.Modules) == 0 {
		return ErrMissingGeneratedModules
	}
	for _, module := range course.Modules {
		if len(module.Lessons) > 0 {
			continue
		}
		if err := s.runLessonPlanJob(ctx, requestID, module.ID); err != nil {
			return err
		}
	}
	return s.finishLessonPlanPhaseIfReady(ctx, course.ID)
}

func (s *CourseGeneratorService) generateMissingLessonContents(ctx context.Context, requestID uuid.UUID, course domain.Course) error {
	if len(course.Modules) == 0 {
		return ErrMissingGeneratedModules
	}
	for _, module := range course.Modules {
		if err := s.runModuleContentJob(ctx, requestID, module.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *CourseGeneratorService) prepareCourseForLessonPlans(ctx context.Context, course domain.Course) (domain.Course, error) {
	switch course.Status {
	case domain.CourseStatusStructureGenerated:
		return s.transitionCourse(ctx, course, func(course *domain.Course) error {
			return course.MarkLessonsGenerating()
		})
	case domain.CourseStatusLessonsGenerating:
		return course, nil
	default:
		return domain.Course{}, fmt.Errorf("%w: cannot generate lesson plans from course status %s", domain.ErrInvalidStatusTransition, course.Status)
	}
}

func (s *CourseGeneratorService) finishLessonPlanPhaseIfReady(ctx context.Context, courseID uuid.UUID) error {
	course, err := s.loadCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if !hasAllLessonPlans(course) {
		return nil
	}
	switch course.Status {
	case domain.CourseStatusStructureGenerated:
		_, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
			if err := course.MarkLessonsGenerating(); err != nil {
				return err
			}
			return course.MarkLessonsGenerated()
		})
		return err
	case domain.CourseStatusLessonsGenerating:
		_, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
			return course.MarkLessonsGenerated()
		})
		return err
	case domain.CourseStatusLessonsGenerated, domain.CourseStatusContentGenerating, domain.CourseStatusCompleted:
		return nil
	default:
		return fmt.Errorf("%w: cannot finish lesson plans from course status %s", domain.ErrInvalidStatusTransition, course.Status)
	}
}

func (s *CourseGeneratorService) prepareCourseForLessonContent(ctx context.Context, course domain.Course) (domain.Course, error) {
	switch course.Status {
	case domain.CourseStatusLessonsGenerated:
		return s.transitionCourse(ctx, course, func(course *domain.Course) error {
			return course.MarkContentGenerating()
		})
	case domain.CourseStatusContentGenerating, domain.CourseStatusCompleted:
		return course, nil
	default:
		return domain.Course{}, fmt.Errorf("%w: cannot generate lesson content from course status %s", domain.ErrInvalidStatusTransition, course.Status)
	}
}

func (s *CourseGeneratorService) loadRunnableJobRequest(ctx context.Context, requestID uuid.UUID) (domain.GenerationRequest, error) {
	if requestID == uuid.Nil {
		return domain.GenerationRequest{}, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}
	request, err := s.loadGenerationRequest(ctx, requestID)
	if err != nil {
		return domain.GenerationRequest{}, err
	}
	if request.PipelineStatus == domain.PipelineStatusFailed {
		return domain.GenerationRequest{}, ErrGenerationNotRetryable
	}
	return request, nil
}

func (s *CourseGeneratorService) loadCourseByRequestID(ctx context.Context, requestID uuid.UUID) (domain.Course, error) {
	var course domain.Course
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedCourse, err := repositories.Courses().FindCourseByRequestID(ctx, requestID)
		if err != nil {
			return err
		}
		course = loadedCourse
		return nil
	})
	return course, err
}

func (s *CourseGeneratorService) completeRequestIfNeeded(ctx context.Context, request domain.GenerationRequest) error {
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		return nil
	}
	return s.completeRequest(ctx, request.ID)
}

func architectureParamsFromJob(request domain.GenerationRequest, payload contract.ArchitectureJobPayload) (contract.GenerateStructureParams, error) {
	params, err := structureParamsFromAnalysis(request)
	if err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if strings.TrimSpace(payload.Title) != "" {
		params.Title = payload.Title
	}
	if strings.TrimSpace(payload.Synopsis) != "" {
		params.Synopsis = payload.Synopsis
	}
	if payload.CurrentLevel != "" {
		params.CurrentLevel = payload.CurrentLevel
	}
	if payload.TargetLevel != "" {
		params.TargetLevel = payload.TargetLevel
	}
	if len(payload.Goals) > 0 {
		params.Goals = payload.Goals
	}
	if payload.Language != "" {
		params.Language = payload.Language
	}
	return normalizeStructureParams(params)
}

func hasAllLessonPlans(course domain.Course) bool {
	if len(course.Modules) == 0 {
		return false
	}
	for _, module := range course.Modules {
		if len(module.Lessons) == 0 {
			return false
		}
	}
	return true
}

func (s *CourseGeneratorService) handleTerminalJobFailure(ctx context.Context, job domain.GenerationJob, cause error) error {
	if job.Kind == domain.GenerationJobKindLessonContent || job.Kind == domain.GenerationJobKindModuleContent {
		request, err := s.loadGenerationRequest(ctx, job.RequestID)
		if err != nil {
			return err
		}
		if request.PipelineStatus == domain.PipelineStatusCompleted {
			return nil
		}
	}
	return s.markPipelineFailed(ctx, job.RequestID, cause)
}
