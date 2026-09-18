package contract

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type StartGenerationParams struct {
	Prompt         string
	IdempotencyKey string
}

type GenerateStructureParams struct {
	RequestID    uuid.UUID
	Title        string
	Synopsis     string
	CurrentLevel domain.Level
	TargetLevel  domain.Level
	Goals        []string
	Language     domain.CourseLanguage
}

type SubmitClarificationsParams struct {
	RequestID uuid.UUID
	Answers   []domain.ClarificationAnswer
	Title     string
	Synopsis  string
	Language  domain.CourseLanguage
}

type GenerationStarted struct {
	JobID        uuid.UUID
	RequestID    uuid.UUID
	Status       domain.GenerationPipelineStatus
	JobStatus    domain.GenerationJobStatus
	StatusURL    string
	JobStatusURL string
	ResultURL    string
}

type GenerationStatus struct {
	GenerationAttempt      int
	ContentComplete        bool
	RequestID              uuid.UUID
	CourseID               *uuid.UUID
	PipelineStatus         domain.GenerationPipelineStatus
	CourseStatus           *domain.CourseGenerationStatus
	CurrentStep            *string
	ProgressPercent        int
	FailureMessage         *string
	IsOutOfScope           bool
	ErrorMessage           *string
	WarningMessage         *string
	SuggestedTitle         *string
	ShortSynopsis          *string
	DetectedCurrentLevel   *domain.Level
	DetectedTargetLevel    *domain.Level
	DetectedGoal           *string
	DetectedLanguage       *domain.CourseLanguage
	ClarificationQuestions []domain.ClarificationQuestion
	ActionRequired         *GenerationActionRequired
}

type GenerationActionRequired struct {
	Type string
	URL  string
}

type GenerationResult struct {
	Request domain.GenerationRequest
	Course  domain.Course
}

type GenerationCommandService interface {
	StartFullCourseGeneration(ctx context.Context, params StartGenerationParams) (GenerationStarted, error)
	SubmitClarifications(ctx context.Context, params SubmitClarificationsParams) (GenerationStarted, error)
	EnqueueCourseStructure(ctx context.Context, params GenerateStructureParams) (GenerationStarted, error)
	EnqueueStructureRetry(ctx context.Context, params GenerateStructureParams) (GenerationStarted, error)
	EnqueueLessonContentGeneration(ctx context.Context, lessonID uuid.UUID) (GenerationStarted, error)
	EnqueueModuleContentGeneration(ctx context.Context, moduleID uuid.UUID) (GenerationStarted, error)
	RetryFullCourseGeneration(ctx context.Context, requestID uuid.UUID) (GenerationStarted, error)
	DeleteGenerationRequest(ctx context.Context, requestID uuid.UUID) error
}

type GenerationQueryService interface {
	ListGenerationJobs(ctx context.Context, requestID uuid.UUID) ([]domain.GenerationJob, error)
	ListGenerationRequests(ctx context.Context, filters GenerationHistoryFilters) (Page[GenerationSummary], error)
	GetGenerationJob(ctx context.Context, jobID uuid.UUID) (domain.GenerationJob, error)
	GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (GenerationStatus, error)
	GetGenerationResult(ctx context.Context, requestID uuid.UUID) (GenerationResult, error)
}

type CourseGenerationService interface {
	GenerationCommandService
	GenerationQueryService
}
