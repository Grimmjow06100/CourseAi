package contract

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type CourseOrderField string

const (
	CourseOrderByCreatedAt CourseOrderField = "created_at"
	CourseOrderByUpdatedAt CourseOrderField = "updated_at"
	CourseOrderByTitle     CourseOrderField = "title"
	CourseOrderByStatus    CourseOrderField = "status"
)

type CourseFilters struct {
	Status         *domain.CourseGenerationStatus
	Language       *domain.CourseLanguage
	Search         string
	OrderBy        CourseOrderField
	OrderDirection SortDirection
	Pagination     Pagination
}

type UserRepository interface {
	SaveUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	FindUserByUsername(ctx context.Context, username domain.Username) (domain.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type GenerationRequestRepository interface {
	SaveGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error)
	UpdateGenerationRequest(ctx context.Context, request domain.GenerationRequest) (domain.GenerationRequest, error)
	FindGenerationRequestByID(ctx context.Context, id uuid.UUID) (domain.GenerationRequest, error)
	FindGenerationRequestByCourseID(ctx context.Context, courseID uuid.UUID) (domain.GenerationRequest, error)
}

type CourseRepository interface {
	SaveCourse(ctx context.Context, course domain.Course) (domain.Course, error)
	UpdateCourse(ctx context.Context, course domain.Course) (domain.Course, error)
	FindCourseByID(ctx context.Context, id uuid.UUID) (domain.Course, error)
	FindCourseByRequestID(ctx context.Context, requestID uuid.UUID) (domain.Course, error)
	ListCourses(ctx context.Context, filters CourseFilters) (Page[domain.Course], error)
	DeleteCourse(ctx context.Context, id uuid.UUID) error
}

type ModuleRepository interface {
	SaveModule(ctx context.Context, module domain.Module) (domain.Module, error)
	UpdateModule(ctx context.Context, module domain.Module) (domain.Module, error)
	FindModuleByID(ctx context.Context, id uuid.UUID) (domain.Module, error)
	ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error)
	DeleteModule(ctx context.Context, id uuid.UUID) error
}

type LessonRepository interface {
	SaveLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error)
	SaveLessons(ctx context.Context, lessons []domain.Lesson) ([]domain.Lesson, error)
	UpdateLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error)
	FindLessonByID(ctx context.Context, id uuid.UUID) (domain.Lesson, error)
	ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error)
	DeleteLesson(ctx context.Context, id uuid.UUID) error
}
