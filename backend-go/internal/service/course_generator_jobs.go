package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/google/uuid"
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

func (s *CourseGeneratorService) enqueueLessonPlanJobs(ctx context.Context, parentJob domain.GenerationJob, course domain.Course) error {
	if len(course.Modules) == 0 {
		return ErrMissingGeneratedModules
	}
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, parentJob.RequestID)
		if err != nil {
			return err
		}
		for _, module := range course.Modules {
			if len(module.Lessons) > 0 {
				continue
			}
			targetID := module.ID
			parentID := parentJob.ID
			job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
				GenerationAttempt: request.GenerationAttempt,
				RequestID:         parentJob.RequestID,
				ParentJobID:       &parentID,
				Kind:              domain.GenerationJobKindLessonPlan,
				TargetID:          &targetID,
				IdempotencyKey:    fmt.Sprintf("lesson_plan:%s:v%d:%s", request.ID, request.ClarificationVersion, module.ID),
				Payload:           json.RawMessage(`{}`),
				AvailableAt:       s.now(),
			}, s.now())
			if err != nil {
				return err
			}
			if _, err := s.enqueueWithCapacity(ctx, repositories, job); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *CourseGeneratorService) finishLessonPlanPhaseAndEnqueueContents(ctx context.Context, currentJob domain.GenerationJob, courseID uuid.UUID) error {
	course, err := s.loadCourseByID(ctx, courseID)
	if err != nil {
		return err
	}
	if !hasAllLessonPlans(course) {
		return nil
	}
	switch course.Status {
	case domain.CourseStatusStructureGenerated:
		course, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
			if err := course.MarkLessonsGenerating(); err != nil {
				return err
			}
			return course.MarkLessonsGenerated()
		})
	case domain.CourseStatusLessonsGenerating:
		course, err = s.transitionCourse(ctx, course, func(course *domain.Course) error {
			return course.MarkLessonsGenerated()
		})
	case domain.CourseStatusLessonsGenerated, domain.CourseStatusContentGenerating, domain.CourseStatusCompleted:
		// The phase was already closed by another worker.
	default:
		err = fmt.Errorf("%w: cannot finish lesson plans from course status %s", domain.ErrInvalidStatusTransition, course.Status)
	}
	if err != nil {
		return err
	}

	lessons := make([]domain.Lesson, 0)
	for _, module := range course.Modules {
		lessons = append(lessons, module.Lessons...)
	}
	if err := s.enqueueLessonContentJobs(ctx, currentJob.RequestID, lessons); err != nil {
		return err
	}
	return s.enqueueFinalizeIfReady(ctx, currentJob.RequestID, course.ID)
}

func (s *CourseGeneratorService) enqueueLessonContentJobs(ctx context.Context, requestID uuid.UUID, lessons []domain.Lesson) error {
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		jobs, err := repositories.GenerationJobs().ListByRequestID(ctx, requestID)
		if err != nil {
			return err
		}
		planParents := make(map[uuid.UUID]uuid.UUID)
		for _, job := range jobs {
			if job.Kind == domain.GenerationJobKindLessonPlan && job.TargetID != nil {
				expectedKey := fmt.Sprintf("lesson_plan:%s:v%d:%s", request.ID, request.ClarificationVersion, *job.TargetID)
				if job.IdempotencyKey == expectedKey {
					planParents[*job.TargetID] = job.ID
				}
			}
		}
		for _, lesson := range lessons {
			if lesson.HasContent() {
				continue
			}
			targetID := lesson.ID
			parentID, hasParent := planParents[lesson.ModuleID]
			var parentJobID *uuid.UUID
			if hasParent {
				parentJobID = &parentID
			}
			job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
				GenerationAttempt: request.GenerationAttempt,
				RequestID:         requestID,
				ParentJobID:       parentJobID,
				Kind:              domain.GenerationJobKindLessonContent,
				TargetID:          &targetID,
				IdempotencyKey:    fmt.Sprintf("lesson_content:%s:v%d:%s", request.ID, request.ClarificationVersion, lesson.ID),
				Payload:           json.RawMessage(`{}`),
				AvailableAt:       s.now(),
			}, s.now())
			if err != nil {
				return err
			}
			if _, err := s.enqueueWithCapacity(ctx, repositories, job); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *CourseGeneratorService) enqueueFinalizeIfReady(ctx context.Context, requestID, courseID uuid.UUID) error {
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		complete, err := repositories.Courses().IsCourseContentComplete(ctx, courseID)
		if err != nil || !complete {
			return err
		}
		request, err := repositories.GenerationRequests().FindGenerationRequestByID(ctx, requestID)
		if err != nil {
			return err
		}
		targetID := courseID
		job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
			GenerationAttempt: request.GenerationAttempt,
			RequestID:         requestID,
			Kind:              domain.GenerationJobKindFinalizeCourse,
			TargetID:          &targetID,
			IdempotencyKey:    fmt.Sprintf("finalize_course:%s:v%d:%s", request.ID, request.ClarificationVersion, courseID),
			Payload:           json.RawMessage(`{}`),
			AvailableAt:       s.now(),
		}, s.now())
		if err != nil {
			return err
		}
		_, err = s.enqueueWithCapacity(ctx, repositories, job)
		return err
	})
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
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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

func (s *CourseGeneratorService) enqueueArchitectureJobWithRepositories(
	ctx context.Context,
	repositories contract.TransactionalRepositories,
	request domain.GenerationRequest,
	parentJobID *uuid.UUID,
) (domain.GenerationJob, error) {
	if request.ConfirmedBrief == nil {
		return domain.GenerationJob{}, ErrGenerationBriefRequired
	}
	key, err := architectureJobKey(request.ID, request.ClarificationVersion, *request.ConfirmedBrief)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	job, err := domain.NewGenerationJobAt(domain.NewGenerationJobParams{
		GenerationAttempt: request.GenerationAttempt,
		RequestID:         request.ID,
		ParentJobID:       parentJobID,
		Kind:              domain.GenerationJobKindArchitecture,
		IdempotencyKey:    key,
		Payload:           json.RawMessage(`{}`),
		AvailableAt:       s.now(),
	}, s.now())
	if err != nil {
		return domain.GenerationJob{}, err
	}
	return s.enqueueWithCapacity(ctx, repositories, job)
}

func architectureJobKey(requestID uuid.UUID, version int, brief domain.GenerationBrief) (string, error) {
	payload, err := json.Marshal(struct {
		Version int                    `json:"version"`
		Brief   domain.GenerationBrief `json:"brief"`
	}{Version: version, Brief: brief})
	if err != nil {
		return "", fmt.Errorf("marshal confirmed generation brief: %w", err)
	}
	return deterministicJobKey("architecture", requestID, payload), nil
}

func structureParamsFromBrief(requestID uuid.UUID, brief domain.GenerationBrief) contract.GenerateStructureParams {
	return contract.GenerateStructureParams{
		RequestID:    requestID,
		Title:        brief.Title,
		Synopsis:     brief.Synopsis,
		CurrentLevel: brief.CurrentLevel,
		TargetLevel:  brief.TargetLevel,
		Goals:        brief.Goals,
		Language:     brief.Language,
	}
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
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, job.RequestID)
		if err != nil {
			return err
		}
		if request.GenerationAttempt != job.GenerationAttempt || request.PipelineStatus.IsTerminal() {
			return nil
		}
		complete, err := s.finalizePersistedCourse(ctx, repositories, request)
		if err != nil || complete {
			return err
		}
		return s.failRequestWithRepositories(ctx, repositories, request, cause)
	})
}
