package contract

import (
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

// GenerationHistoryFilters controls the authenticated generation history view.
type GenerationHistoryFilters struct {
	ClerkUserID    string
	PipelineStatus *domain.GenerationPipelineStatus
	Pagination     Pagination
}

// GenerationSummary is the read projection used to resume a generation without
// loading raw AI outputs or the complete generated course graph.
type GenerationSummary struct {
	RequestID         uuid.UUID
	CourseID          *uuid.UUID
	InitialUserPrompt string
	Title             string
	PipelineStatus    domain.GenerationPipelineStatus
	CourseStatus      *domain.CourseGenerationStatus
	CurrentStep       *string
	ProgressPercent   int
	IsOutOfScope      bool
	FailureMessage    *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
