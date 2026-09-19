package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

func (s *CourseGeneratorService) enqueueWithCapacity(ctx context.Context, repositories contract.TransactionalRepositories, job domain.GenerationJob) (domain.GenerationJob, error) {
	if job.GenerationAttempt > 1 {
		job.IdempotencyKey += fmt.Sprintf(":attempt:%d", job.GenerationAttempt)
	}
	jobs, err := repositories.GenerationJobs().ListByRequestID(ctx, job.RequestID)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	for _, existing := range jobs {
		if existing.IsCurrent && existing.GenerationAttempt == job.GenerationAttempt && existing.Kind == job.Kind && pointer.Equal(existing.TargetID, job.TargetID) {
			return existing, nil
		}
	}

	if s.config.MaxPendingJobs > 0 {
		pending, err := repositories.GenerationJobs().CountPendingWithAdmissionLock(ctx)
		if err != nil {
			return domain.GenerationJob{}, err
		}
		if pending >= s.config.MaxPendingJobs {
			return domain.GenerationJob{}, contract.ErrGenerationQueueSaturated
		}
	}
	return repositories.GenerationJobs().Enqueue(ctx, job)
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
			if job.IsCurrent && job.GenerationAttempt == request.GenerationAttempt && job.Kind == domain.GenerationJobKindLessonPlan && job.TargetID != nil {
				planParents[*job.TargetID] = job.ID
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

func deterministicJobKey(prefix string, requestID uuid.UUID, payload json.RawMessage) string {
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%s:%s:%x", prefix, requestID, digest[:12])
}

func generationRequestIdempotencyKey(owner string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "analysis:clerk_user:" + strings.TrimSpace(owner) + ":client:" + value
}
