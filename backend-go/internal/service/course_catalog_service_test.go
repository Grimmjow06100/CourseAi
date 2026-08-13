package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestCourseCatalogServiceDelegatesOperations(t *testing.T) {
	t.Parallel()

	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	wantCourse := domain.Course{ID: courseID}
	wantModule := domain.Module{ID: moduleID, CourseID: courseID}
	wantLesson := domain.Lesson{ID: lessonID, ModuleID: moduleID}
	wantPage := contract.Page[domain.Course]{Items: []domain.Course{wantCourse}, Page: 1, TotalItems: 1}
	courses := &catalogCourseRepository{course: wantCourse, page: wantPage}
	modules := &catalogModuleRepository{module: wantModule, modules: []domain.Module{wantModule}}
	lessons := &catalogLessonRepository{lesson: wantLesson, lessons: []domain.Lesson{wantLesson}}
	service := NewCourseCatalogService(catalogUnitOfWork{repositories: catalogRepositories{courses: courses, modules: modules, lessons: lessons}})
	ctx := context.Background()

	if got, err := service.GetCourse(ctx, courseID); err != nil || got.ID != courseID {
		t.Fatalf("GetCourse() = %+v, %v", got, err)
	}
	filters := contract.CourseFilters{Search: "linux"}
	if got, err := service.ListCourses(ctx, filters); err != nil || !reflect.DeepEqual(got, wantPage) {
		t.Fatalf("ListCourses() = %+v, %v", got, err)
	}
	if err := service.DeleteCourse(ctx, courseID); err != nil || courses.deletedID != courseID {
		t.Fatalf("DeleteCourse() error = %v, deletedID=%s", err, courses.deletedID)
	}
	if got, err := service.GetModule(ctx, moduleID); err != nil || got.ID != moduleID {
		t.Fatalf("GetModule() = %+v, %v", got, err)
	}
	if got, err := service.ListModulesByCourseID(ctx, courseID); err != nil || !reflect.DeepEqual(got, modules.modules) {
		t.Fatalf("ListModulesByCourseID() = %+v, %v", got, err)
	}
	if got, err := service.GetLesson(ctx, lessonID); err != nil || got.ID != lessonID {
		t.Fatalf("GetLesson() = %+v, %v", got, err)
	}
	if got, err := service.ListLessonsByModuleID(ctx, moduleID); err != nil || !reflect.DeepEqual(got, lessons.lessons) {
		t.Fatalf("ListLessonsByModuleID() = %+v, %v", got, err)
	}
}

func TestCourseCatalogServiceValidatesDependenciesAndPropagatesErrors(t *testing.T) {
	t.Parallel()

	var nilService *CourseCatalogService
	if _, err := nilService.GetCourse(context.Background(), uuid.New()); !errors.Is(err, ErrCourseCatalogDependency) {
		t.Fatalf("nil service error = %v", err)
	}
	repositoryErr := errors.New("query failed")
	service := NewCourseCatalogService(catalogUnitOfWork{repositories: catalogRepositories{
		courses: &catalogCourseRepository{err: repositoryErr},
	}})
	if _, err := service.GetCourse(context.Background(), uuid.New()); !errors.Is(err, repositoryErr) {
		t.Fatalf("repository error = %v, want wrapped query error", err)
	}
}

type catalogUnitOfWork struct {
	repositories contract.TransactionalRepositories
}

func (u catalogUnitOfWork) WithinTx(ctx context.Context, fn func(context.Context, contract.TransactionalRepositories) error) error {
	return fn(ctx, u.repositories)
}

type catalogRepositories struct {
	contract.TransactionalRepositories
	courses contract.CourseRepository
	modules contract.ModuleRepository
	lessons contract.LessonRepository
}

func (r catalogRepositories) Courses() contract.CourseRepository { return r.courses }
func (r catalogRepositories) Modules() contract.ModuleRepository { return r.modules }
func (r catalogRepositories) Lessons() contract.LessonRepository { return r.lessons }

type catalogCourseRepository struct {
	contract.CourseRepository
	course    domain.Course
	page      contract.Page[domain.Course]
	err       error
	deletedID uuid.UUID
}

func (r *catalogCourseRepository) FindCourseByID(context.Context, uuid.UUID) (domain.Course, error) {
	return r.course, r.err
}
func (r *catalogCourseRepository) ListCourses(context.Context, contract.CourseFilters) (contract.Page[domain.Course], error) {
	return r.page, r.err
}
func (r *catalogCourseRepository) DeleteCourse(_ context.Context, id uuid.UUID) error {
	r.deletedID = id
	return r.err
}

type catalogModuleRepository struct {
	contract.ModuleRepository
	module  domain.Module
	modules []domain.Module
	err     error
}

func (r *catalogModuleRepository) FindModuleByID(context.Context, uuid.UUID) (domain.Module, error) {
	return r.module, r.err
}
func (r *catalogModuleRepository) ListModulesByCourseID(context.Context, uuid.UUID) ([]domain.Module, error) {
	return r.modules, r.err
}

type catalogLessonRepository struct {
	contract.LessonRepository
	lesson  domain.Lesson
	lessons []domain.Lesson
	err     error
}

func (r *catalogLessonRepository) FindLessonByID(context.Context, uuid.UUID) (domain.Lesson, error) {
	return r.lesson, r.err
}
func (r *catalogLessonRepository) ListLessonsByModuleID(context.Context, uuid.UUID) ([]domain.Lesson, error) {
	return r.lessons, r.err
}
