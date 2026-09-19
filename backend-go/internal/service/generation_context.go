package service

import (
	"context"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) loadCourseByID(ctx context.Context, courseID uuid.UUID) (domain.Course, error) {
	var course domain.Course
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, courseID)
		if err != nil {
			return err
		}
		course = loadedCourse
		return nil
	})
	return course, err
}

func (s *CourseGeneratorService) loadLessonGenerationContext(ctx context.Context, lessonID uuid.UUID) (domain.Course, domain.Module, domain.Lesson, error) {
	var course domain.Course
	var module domain.Module
	var lesson domain.Lesson
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedLesson, err := repositories.Lessons().FindLessonByID(ctx, lessonID)
		if err != nil {
			return err
		}
		loadedModule, err := repositories.Modules().FindModuleByID(ctx, loadedLesson.ModuleID)
		if err != nil {
			return err
		}
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, loadedModule.CourseID)
		if err != nil {
			return err
		}

		lesson = loadedLesson
		module = loadedModule
		course = loadedCourse
		return nil
	})
	return course, module, lesson, err
}

func (s *CourseGeneratorService) loadModuleGenerationContext(ctx context.Context, moduleID uuid.UUID) (domain.Course, domain.Module, error) {
	var course domain.Course
	var module domain.Module
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedModule, err := repositories.Modules().FindModuleByID(ctx, moduleID)
		if err != nil {
			return err
		}
		loadedCourse, err := repositories.Courses().FindCourseByID(ctx, loadedModule.CourseID)
		if err != nil {
			return err
		}

		module = loadedModule
		course = loadedCourse
		return nil
	})
	return course, module, err
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

func (s *CourseGeneratorService) loadGenerationRequest(ctx context.Context, requestID uuid.UUID) (domain.GenerationRequest, error) {
	var request domain.GenerationRequest
	err := s.withinTx(ctx, func(ctx context.Context, repositories contract.TransactionalRepositories) error {
		loadedRequest, err := repositories.GenerationRequests().FindGenerationRequestForUpdate(ctx, requestID)
		if err != nil {
			return err
		}
		request = loadedRequest
		return nil
	})
	return request, err
}
