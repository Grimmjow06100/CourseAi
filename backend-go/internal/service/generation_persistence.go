package service

import (
	"context"
	"encoding/json"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/jsonutil"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) updateRequestProgress(ctx context.Context, requestID uuid.UUID, step string, percent int) (domain.GenerationRequest, error) {
	var updatedRequest domain.GenerationRequest
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
		if err != nil {
			return err
		}

		now := s.now()
		if request.PipelineStatus == domain.PipelineStatusQueued {
			if err := request.MarkRunning(step, now); err != nil {
				return err
			}
		}
		if request.PipelineStatus.IsTerminal() || percent < request.ProgressPercent {
			updatedRequest = request
			return nil
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
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
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
	err = s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if _, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, course.RequestID); err != nil {
			return err
		}
		current, err := repositories.Courses().FindCourseByID(ctx, course.ID)
		if err != nil {
			return err
		}
		if current.Status != course.Status {
			updatedCourse = current
			return nil
		}
		course = current
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
	return s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		request, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
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

func (s *CourseGeneratorService) completeRequestIfNeeded(ctx context.Context, request domain.GenerationRequest) error {
	if request.PipelineStatus == domain.PipelineStatusCompleted {
		return nil
	}
	return s.completeRequest(ctx, request.ID)
}
