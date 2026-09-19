package domain

import "time"

// SettleIncomplete is called only after the orchestrator has verified that no
// current operation can make progress. Available lessons remain readable.
func (r *GenerationRequest) SettleIncomplete(hasContent bool, now time.Time) error {
	if r.IsOutOfScope || r.PipelineStatus == PipelineStatusAwaitingClarification || r.PipelineStatus == PipelineStatusCompleted {
		return ErrGenerationRequestNotReady
	}
	message := "Generation stopped before all lessons were available"
	r.PipelineStatus = PipelineStatusFailed
	if hasContent {
		r.PipelineStatus = PipelineStatusPartial
	}
	r.FailureMessage = &message
	r.CompletedAt = &now
	r.UpdatedAt = now
	return nil
}

// ResumeOperation preserves the generation attempt so independent workers keep
// their claims. The caller must atomically admit at least one replacement job.
func (r *GenerationRequest) ResumeOperation(now time.Time) error {
	if r.IsOutOfScope || r.PipelineStatus == PipelineStatusAwaitingClarification || r.PipelineStatus == PipelineStatusCompleted {
		return ErrGenerationRequestNotReady
	}
	r.PipelineStatus = PipelineStatusRunning
	r.FailureMessage = nil
	r.CompletedAt = nil
	r.UpdatedAt = now
	return nil
}

func (c *Course) SettleIncomplete(hasContent bool) error {
	if c.Status == CourseStatusCompleted {
		return ErrInvalidStatusTransition
	}
	c.Status = CourseStatusFailed
	if hasContent {
		c.Status = CourseStatusPartial
	}
	return nil
}
