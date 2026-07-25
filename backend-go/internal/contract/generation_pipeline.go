package contract

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type StartGenerationParams struct {
	Prompt string
}

type GenerationStarted struct {
	RequestID uuid.UUID
	Status    domain.GenerationPipelineStatus
	StatusURL string
	ResultURL string
}

type GenerationStatus struct {
	RequestID       uuid.UUID
	CourseID        *uuid.UUID
	PipelineStatus  domain.GenerationPipelineStatus
	CourseStatus    *domain.CourseGenerationStatus
	CurrentStep     *string
	ProgressPercent int
	FailureMessage  *string
}

type GenerationResult struct {
	Request domain.GenerationRequest
	Course  domain.Course
}

type CourseGenerationService interface {
	StartFullCourseGeneration(ctx context.Context, params StartGenerationParams) (GenerationStarted, error)
	GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (GenerationStatus, error)
	GetGenerationResult(ctx context.Context, requestID uuid.UUID) (GenerationResult, error)
	RetryFullCourseGeneration(ctx context.Context, requestID uuid.UUID) (GenerationStarted, error)
}
