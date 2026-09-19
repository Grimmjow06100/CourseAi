package service

import (
	"context"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

var _ contract.GenerationTrackingService = (*CourseGeneratorService)(nil)

func (s *CourseGeneratorService) GetGenerationTracking(ctx context.Context, id uuid.UUID) (contract.GenerationTracking, error) {
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationTracking{}, err
	}
	var snapshot contract.GenerationTracking
	err = s.withinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, r.Ownership(), ownedGenerationRequest, id, owner); err != nil {
			return err
		}
		var err error
		snapshot, err = r.Tracking().Snapshot(ctx, id)
		return err
	})
	if err == nil {
		enrichTracking(&snapshot)
	}
	return snapshot, err
}

func (s *CourseGeneratorService) GetGenerationEvents(ctx context.Context, id uuid.UUID, cursor int64, limit int) (contract.GenerationEventPage, error) {
	owner, err := authenticatedOwner(ctx)
	if err != nil {
		return contract.GenerationEventPage{}, err
	}
	if cursor < 0 || limit < 1 || limit > 100 {
		return contract.GenerationEventPage{}, fmt.Errorf("%w: event pagination", domain.ErrBlankField)
	}
	var result contract.GenerationEventPage
	err = s.withinTx(ctx, func(ctx context.Context, r contract.TransactionalRepositories) error {
		if err := authorizeOwnedResource(ctx, r.Ownership(), ownedGenerationRequest, id, owner); err != nil {
			return err
		}
		items, err := r.Tracking().Events(ctx, id, cursor, limit+1)
		if err != nil {
			return err
		}
		if len(items) > limit {
			items = items[:limit]
			next := items[len(items)-1].ID
			result.NextCursor = &next
		}
		result.Items = items
		return nil
	})
	return result, err
}

func enrichTracking(t *contract.GenerationTracking) {
	t.Counts = contract.TrackingCounts{Modules: len(t.Modules)}
	known := 0
	available := map[uuid.UUID]bool{}
	for _, module := range t.Modules {
		if len(module.Lessons) > 0 {
			t.Counts.PlansReady++
		}
		known += len(module.Lessons)
		for _, lesson := range module.Lessons {
			available[lesson.ID] = lesson.HasContent
			if lesson.HasContent {
				t.Counts.LessonsAvailable++
			}
		}
	}
	allPlans := t.Counts.Modules > 0 && t.Counts.PlansReady == t.Counts.Modules
	if allPlans {
		t.Counts.LessonsExpected = &known
	}
	t.ContentAvailability = "none"
	if t.Counts.LessonsAvailable > 0 {
		t.ContentAvailability = "partial"
	}
	if t.ContentComplete {
		t.ContentAvailability = "complete"
	}
	phaseStates := map[domain.GenerationJobKind]string{}
	for i := range t.Operations {
		op := &t.Operations[i]
		if !op.Status.IsTerminal() {
			t.Counts.JobsActive++
		}
		if op.Status == domain.GenerationJobStatusFailed || op.Status == domain.GenerationJobStatusCancelled {
			t.Counts.JobsFailed++
		}
		if op.Status == domain.GenerationJobStatusCompleted {
			t.Counts.JobsCompleted++
		}
		op.Retryable = (op.Status == domain.GenerationJobStatusFailed || op.Status == domain.GenerationJobStatusCancelled) && !t.IsOutOfScope && t.PipelineStatus != domain.PipelineStatusCompleted
		if op.FailureCode != nil && *op.FailureCode == "operator_action_required" {
			op.Retryable = false
		}
		if op.Kind == domain.GenerationJobKindLessonContent && op.TargetID != nil {
			op.Retryable = op.Retryable && !available[*op.TargetID] && allPlans
		}
		state := string(op.Status)
		if old := phaseStates[op.Kind]; phasePriority(state) > phasePriority(old) {
			phaseStates[op.Kind] = state
		}
	}
	t.HasActiveWork = t.Counts.JobsActive > 0
	t.Reconciliation = "none"
	if t.ContentComplete && t.PipelineStatus != domain.PipelineStatusCompleted {
		t.Reconciliation = "pending"
	}
	if !t.HasActiveWork && !t.ContentComplete && !t.PipelineStatus.IsTerminal() && t.PipelineStatus != domain.PipelineStatusAwaitingClarification {
		t.Reconciliation = "attention_required"
	}
	t.Phases = make([]contract.TrackingPhase, 0, 5)
	for _, kind := range []domain.GenerationJobKind{domain.GenerationJobKindAnalysis, domain.GenerationJobKindArchitecture, domain.GenerationJobKindLessonPlan, domain.GenerationJobKindLessonContent, domain.GenerationJobKindFinalizeCourse} {
		state := phaseStates[kind]
		if state == "" {
			state = "pending"
		}
		// Persisted descendants prove these stages finished even after legacy
		// jobs were purged, or when a new generation attempt reuses the course.
		if (kind == domain.GenerationJobKindAnalysis && t.CourseID != nil) || (kind == domain.GenerationJobKindArchitecture && t.Counts.Modules > 0) {
			state = "completed"
		}
		if kind == domain.GenerationJobKindLessonPlan && allPlans {
			state = "completed"
		}
		if kind == domain.GenerationJobKindLessonContent && !allPlans {
			state = "blocked"
		}
		if kind == domain.GenerationJobKindLessonContent && t.ContentComplete {
			state = "completed"
		}
		if kind == domain.GenerationJobKindFinalizeCourse && t.PipelineStatus == domain.PipelineStatusCompleted {
			state = "completed"
		}
		if kind == domain.GenerationJobKindAnalysis && t.PipelineStatus == domain.PipelineStatusAwaitingClarification {
			state = "awaiting_clarification"
		}
		t.Phases = append(t.Phases, contract.TrackingPhase{Kind: string(kind), Status: state})
	}
}

func phasePriority(status string) int {
	switch status {
	case "running":
		return 6
	case "retry_scheduled":
		return 5
	case "queued":
		return 4
	case "failed":
		return 3
	case "cancelled":
		return 2
	case "completed":
		return 1
	default:
		return 0
	}
}
