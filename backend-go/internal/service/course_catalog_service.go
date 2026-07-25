package service

import (
	"context"
	"errors"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

var ErrCourseCatalogDependency = errors.New("course catalog service dependency is missing")

type CourseCatalogService struct {
	uow contract.UnitOfWork
}

func NewCourseCatalogService(uow contract.UnitOfWork) *CourseCatalogService {
	return &CourseCatalogService{uow: uow}
}

func (s *CourseCatalogService) GetCourse(ctx context.Context, id uuid.UUID) (domain.Course, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.Course{}, err
	}

	var course domain.Course
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		foundCourse, err := repositories.Courses().FindCourseByID(ctx, id)
		if err != nil {
			return err
		}
		course = foundCourse
		return nil
	})
	return course, err
}

func (s *CourseCatalogService) ListCourses(ctx context.Context, filters contract.CourseFilters) (contract.Page[domain.Course], error) {
	if err := s.validateDependencies(); err != nil {
		return contract.Page[domain.Course]{}, err
	}

	var courses contract.Page[domain.Course]
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		page, err := repositories.Courses().ListCourses(ctx, filters)
		if err != nil {
			return err
		}
		courses = page
		return nil
	})
	return courses, err
}

func (s *CourseCatalogService) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}

	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		return repositories.Courses().DeleteCourse(ctx, id)
	})
}

func (s *CourseCatalogService) GetModule(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.Module{}, err
	}

	var module domain.Module
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		foundModule, err := repositories.Modules().FindModuleByID(ctx, id)
		if err != nil {
			return err
		}
		module = foundModule
		return nil
	})
	return module, err
}

func (s *CourseCatalogService) ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	var modules []domain.Module
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		foundModules, err := repositories.Modules().ListModulesByCourseID(ctx, courseID)
		if err != nil {
			return err
		}
		modules = foundModules
		return nil
	})
	return modules, err
}

func (s *CourseCatalogService) GetLesson(ctx context.Context, id uuid.UUID) (domain.Lesson, error) {
	if err := s.validateDependencies(); err != nil {
		return domain.Lesson{}, err
	}

	var lesson domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		foundLesson, err := repositories.Lessons().FindLessonByID(ctx, id)
		if err != nil {
			return err
		}
		lesson = foundLesson
		return nil
	})
	return lesson, err
}

func (s *CourseCatalogService) ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	var lessons []domain.Lesson
	err := s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		foundLessons, err := repositories.Lessons().ListLessonsByModuleID(ctx, moduleID)
		if err != nil {
			return err
		}
		lessons = foundLessons
		return nil
	})
	return lessons, err
}

func (s *CourseCatalogService) validateDependencies() error {
	if s == nil || s.uow == nil {
		return ErrCourseCatalogDependency
	}
	return nil
}
