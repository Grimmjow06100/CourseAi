package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

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
