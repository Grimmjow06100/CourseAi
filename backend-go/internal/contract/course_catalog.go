package contract

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type CourseCatalogService interface {
	GetCourse(ctx context.Context, id uuid.UUID) (domain.Course, error)
	ListCourses(ctx context.Context, filters CourseFilters) (Page[domain.Course], error)
	DeleteCourse(ctx context.Context, id uuid.UUID) error
	GetModule(ctx context.Context, id uuid.UUID) (domain.Module, error)
	ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error)
	GetLesson(ctx context.Context, id uuid.UUID) (domain.Lesson, error)
	ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error)
}
