package service

import (
	"context"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

var ErrCourseCatalogDependency = fmt.Errorf("course catalog service: %w", contract.ErrServiceDependency)

type CourseCatalogService struct {
	uow contract.UnitOfWork
}

func NewCourseCatalogService(uow contract.UnitOfWork) *CourseCatalogService {
	return &CourseCatalogService{uow: uow}
}

func (s *CourseCatalogService) GetCourse(ctx context.Context, id uuid.UUID) (domain.Course, error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return domain.Course{}, err
	}
	return queryOwnedResource(ctx, s.uow, owner, ownedCourse, id, func(ctx context.Context, repositories contract.TransactionalRepositories) (domain.Course, error) {
		return repositories.Courses().FindCourseByID(ctx, id)
	})
}

func (s *CourseCatalogService) ListCourses(ctx context.Context, filters contract.CourseFilters) (contract.Page[domain.Course], error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}
	filters.ClerkUserID = owner

	var courses contract.Page[domain.Course]
	err = s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
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
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return err
	}

	return s.uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), ownedCourse, id, owner); err != nil {
			return err
		}
		return repositories.Courses().DeleteCourseGeneration(ctx, id)
	})
}

func (s *CourseCatalogService) GetModule(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	return queryOwnedResource(ctx, s.uow, owner, ownedModule, id, func(ctx context.Context, repositories contract.TransactionalRepositories) (domain.Module, error) {
		return repositories.Modules().FindModuleByID(ctx, id)
	})
}

func (s *CourseCatalogService) ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return nil, err
	}
	return queryOwnedResource(ctx, s.uow, owner, ownedCourse, courseID, func(ctx context.Context, repositories contract.TransactionalRepositories) ([]domain.Module, error) {
		return repositories.Modules().ListModulesByCourseID(ctx, courseID)
	})
}

func (s *CourseCatalogService) GetLesson(ctx context.Context, id uuid.UUID) (domain.Lesson, error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return domain.Lesson{}, err
	}
	return queryOwnedResource(ctx, s.uow, owner, ownedLesson, id, func(ctx context.Context, repositories contract.TransactionalRepositories) (domain.Lesson, error) {
		return repositories.Lessons().FindLessonByID(ctx, id)
	})
}

func (s *CourseCatalogService) ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	owner, err := s.requestOwner(ctx)
	if err != nil {
		return nil, err
	}
	return queryOwnedResource(ctx, s.uow, owner, ownedModule, moduleID, func(ctx context.Context, repositories contract.TransactionalRepositories) ([]domain.Lesson, error) {
		return repositories.Lessons().ListLessonsByModuleID(ctx, moduleID)
	})
}

func (s *CourseCatalogService) requestOwner(ctx context.Context) (string, error) {
	if err := s.validateDependencies(); err != nil {
		return "", err
	}
	return authenticatedOwner(ctx)
}

func queryOwnedResource[T any](
	ctx context.Context,
	uow contract.UnitOfWork,
	owner string,
	kind ownedResource,
	id uuid.UUID,
	query func(context.Context, contract.TransactionalRepositories) (T, error),
) (T, error) {
	var result T
	err := uow.WithinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, repositories.Ownership(), kind, id, owner); err != nil {
			return err
		}
		var err error
		result, err = query(ctx, repositories)
		return err
	})
	return result, err
}

func (s *CourseCatalogService) validateDependencies() error {
	if s == nil || s.uow == nil {
		return ErrCourseCatalogDependency
	}
	return nil
}
