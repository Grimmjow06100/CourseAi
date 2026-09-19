package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
)

func (s *CourseGeneratorService) runAnalysisJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, job.RequestID)
	if err != nil {
		return err
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted || request.PipelineStatus == domain.PipelineStatusAwaitingClarification {
		return nil
	}
	if request.AnalysisCompletedAt == nil {
		request, err = s.runPromptAnalysis(ctx, request)
		if err != nil {
			return err
		}
	}
	return s.settleAnalysisJob(ctx, job, request)
}

func (s *CourseGeneratorService) settleAnalysisJob(ctx context.Context, parentJob domain.GenerationJob, request domain.GenerationRequest) error {
	if request.IsOutOfScope {
		return s.completeRequestIfNeeded(ctx, request)
	}

	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		locked, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, request.ID)
		if err != nil {
			return err
		}
		if locked.PipelineStatus == domain.PipelineStatusAwaitingClarification || locked.PipelineStatus == domain.PipelineStatusCompleted {
			return nil
		}
		if locked.ConfirmedBrief != nil {
			_, err = s.enqueueArchitectureJobWithRepositories(ctx, repositories, locked, &parentJob.ID)
			return err
		}
		if locked.NeedsClarification() {
			if err := locked.MarkAwaitingClarification(s.now()); err != nil {
				return err
			}
			_, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, locked)
			return err
		}
		if err := locked.ConfirmDetectedBrief(s.now()); err != nil {
			return err
		}
		locked, err = repositories.GenerationRequests().UpdateGenerationRequest(ctx, locked)
		if err != nil {
			return err
		}
		_, err = s.enqueueArchitectureJobWithRepositories(ctx, repositories, locked, &parentJob.ID)
		return err
	})
}

func (s *CourseGeneratorService) runArchitectureJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, job.RequestID)
	if err != nil {
		return err
	}
	if request.PipelineStatus == domain.PipelineStatusAwaitingClarification {
		return ErrGenerationAwaitingClarification
	}
	if request.IsOutOfScope {
		return ErrGenerationOutOfScope
	}
	if request.ConfirmedBrief == nil {
		return ErrGenerationBriefRequired
	}
	request, err = s.updateRequestProgress(ctx, request.ID, stepArchitecture, 35)
	if err != nil {
		return err
	}

	course, err := s.loadCourseByRequestID(ctx, request.ID)
	if errors.Is(err, contract.ErrCourseNotFound) {
		_, course, err = s.generateArchitectureJob(ctx, request, structureParamsFromBrief(request.ID, *request.ConfirmedBrief))
	}
	if err != nil {
		return err
	}
	if hasAllLessonPlans(course) {
		return s.finishLessonPlanPhaseAndEnqueueContents(ctx, job, course.ID)
	}
	return s.enqueueLessonPlanJobs(ctx, job, course)
}

func (s *CourseGeneratorService) runLessonPlanJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, job.RequestID)
	if err != nil {
		return err
	}
	moduleID := *job.TargetID
	course, module, err := s.loadModuleGenerationContext(ctx, moduleID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: module does not belong to generation request", domain.ErrInvalidCollection)
	}
	if len(module.Lessons) > 0 {
		return s.finishLessonPlanPhaseAndEnqueueContents(ctx, job, course.ID)
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
	return s.finishLessonPlanPhaseAndEnqueueContents(ctx, job, course.ID)
}

func (s *CourseGeneratorService) runLessonContentJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, job.RequestID)
	if err != nil {
		return err
	}
	lessonID := *job.TargetID
	course, module, lesson, err := s.loadLessonGenerationContext(ctx, lessonID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: lesson does not belong to generation request", domain.ErrInvalidCollection)
	}
	if !lesson.HasContent() {
		course, err = s.prepareCourseForLessonContent(ctx, course)
		if err != nil {
			return err
		}
		if request.PipelineStatus != domain.PipelineStatusCompleted {
			if _, err := s.updateRequestProgress(ctx, request.ID, stepLessonContent, 75); err != nil {
				return err
			}
		}
		if err := s.generateAndPersistSingleLessonContent(ctx, course, module, lesson); err != nil {
			return err
		}
	}
	return s.enqueueFinalizeIfReady(ctx, job.RequestID, course.ID)
}

func (s *CourseGeneratorService) runModuleContentJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	request, err := s.loadRunnableJobRequest(ctx, job.RequestID)
	if err != nil {
		return err
	}
	course, module, err := s.loadModuleGenerationContext(ctx, *job.TargetID)
	if err != nil {
		return err
	}
	if course.RequestID != request.ID {
		return fmt.Errorf("%w: module does not belong to generation request", domain.ErrInvalidCollection)
	}
	if len(module.Lessons) == 0 {
		return ErrMissingGeneratedLessons
	}
	if _, err := s.prepareCourseForLessonContent(ctx, course); err != nil && course.Status != domain.CourseStatusCompleted {
		return err
	}
	if err := s.enqueueLessonContentJobs(ctx, job.RequestID, module.Lessons); err != nil {
		return err
	}
	return s.enqueueFinalizeIfReady(ctx, request.ID, course.ID)
}

func (s *CourseGeneratorService) runFinalizeCourseJob(ctx context.Context, job domain.GenerationJob) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, job.RequestID)
		if err != nil {
			return err
		}
		if request.GenerationAttempt != job.GenerationAttempt {
			return nil
		}
		course, err := repositories.Courses().FindCourseStateByID(ctx, *job.TargetID)
		if err != nil {
			return err
		}
		if course.RequestID != request.ID {
			return domain.ErrInvalidCollection
		}
		complete, err := s.finalizePersistedCourse(ctx, repositories, request)
		if err != nil {
			return err
		}
		if !complete {
			return domain.ErrMissingCourseContent
		}
		return nil
	})
}

func (s *CourseGeneratorService) generateArchitectureJob(
	ctx context.Context,
	request domain.GenerationRequest,
	params contract.GenerateStructureParams,
) (domain.GenerationRequest, domain.Course, error) {
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

func (s *CourseGeneratorService) generateAndPersistSingleLessonContent(ctx context.Context, course domain.Course, module domain.Module, lesson domain.Lesson) error {
	output, err := s.ai.GenerateLessonContent(ctx, contract.LessonContentInput{Course: course, Module: module, Lesson: lesson})
	if err != nil {
		return fmt.Errorf("generate content for lesson %s: %w", lesson.ID, err)
	}
	lessonWithContent, err := s.attachGeneratedContent(lesson, output)
	if err != nil {
		return err
	}
	return s.persistLessonContent(ctx, lessonWithContent)
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
