package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

// The caller holds the request lock. Both lifecycle updates commit together.
func (s *CourseGeneratorService) finalizePersistedCourse(ctx context.Context, repositories contract.TransactionalRepositories, request domain.GenerationRequest) (bool, error) {
	course, err := repositories.Courses().FindCourseStateByRequestID(ctx, request.ID)
	if errors.Is(err, contract.ErrCourseNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	complete, err := repositories.Courses().IsCourseContentComplete(ctx, course.ID)
	if err != nil || !complete {
		return false, err
	}
	if request.PipelineStatus == domain.PipelineStatusCompleted && course.Status == domain.CourseStatusCompleted {
		return true, nil
	}
	if err := course.ReconcileCompletedContent(complete); err != nil {
		return false, err
	}
	if err := request.ReconcileCompletedCourse(complete, s.now()); err != nil {
		return false, err
	}
	course.UpdatedAt = s.now()
	if _, err := repositories.Courses().UpdateCourse(ctx, course); err != nil {
		return false, err
	}
	if _, err := repositories.GenerationRequests().UpdateGenerationRequest(ctx, request); err != nil {
		return false, err
	}
	return true, nil
}

// ReconcileCompletedGenerations repairs interrupted finalization, never generates content.
// Active attempts are excluded and checked again under the same lock used by writers.
func (s *CourseGeneratorService) ReconcileCompletedGenerations(ctx context.Context, limit int) error {
	if s == nil || s.uow == nil {
		return ErrCourseGeneratorDependency
	}
	var ids []uuid.UUID
	if err := s.withinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		var err error
		ids, err = r.GenerationRequests().ListCompletionCandidates(ctx, limit)
		return err
	}); err != nil {
		return err
	}
	var result error
	for _, id := range ids {
		if _, err := s.ReconcileCompletedGeneration(ctx, id); err != nil {
			result = errors.Join(result, fmt.Errorf("reconcile generation %s: %w", id, err))
		}
	}
	return result
}

// ReconcileCompletedGeneration rechecks one candidate under lock. It does not call
// AI or change the attempt, and skips any request with current work still active.
func (s *CourseGeneratorService) ReconcileCompletedGeneration(ctx context.Context, id uuid.UUID) (bool, error) {
	if s == nil || s.uow == nil {
		return false, ErrCourseGeneratorDependency
	}
	repaired := false
	err := s.withinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		request, err := r.GenerationRequests().FindGenerationRequestForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if request.IsOutOfScope || request.PipelineStatus == domain.PipelineStatusAwaitingClarification {
			return nil
		}
		jobs, err := r.GenerationJobs().ListByRequestID(ctx, id)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			if job.GenerationAttempt == request.GenerationAttempt && !job.Status.IsTerminal() {
				return nil
			}
		}
		repaired, err = s.finalizePersistedCourse(ctx, r, request)
		return err
	})
	return repaired, err
}

func (s *CourseGeneratorService) cancelSupersededJobs(ctx context.Context, repositories contract.TransactionalRepositories, request domain.GenerationRequest) error {
	jobs, err := repositories.GenerationJobs().ListByRequestID(ctx, request.ID)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.GenerationAttempt < request.GenerationAttempt && (job.Status == domain.GenerationJobStatusQueued || job.Status == domain.GenerationJobStatusRetryScheduled) {
			if err := repositories.GenerationJobs().Cancel(ctx, job.ID, s.now()); err != nil && !errors.Is(err, contract.ErrGenerationJobNotCancellable) {
				return err
			}
		}
	}
	return nil
}
