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

type AnalyzePromptParams struct {
	Prompt string
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

type GenerationAnalysisResult struct {
	Request domain.GenerationRequest
}

type CourseGenerationService interface {
	AnalyzePrompt(ctx context.Context, params AnalyzePromptParams) (GenerationAnalysisResult, error)
	StartFullCourseGeneration(ctx context.Context, params StartGenerationParams) (GenerationStarted, error)
	SubmitClarifications(ctx context.Context, params SubmitClarificationsParams) (GenerationStarted, error)
	EnqueueCourseStructure(ctx context.Context, params GenerateStructureParams) (GenerationStarted, error)
	EnqueueStructureRetry(ctx context.Context, params GenerateStructureParams) (GenerationStarted, error)
	EnqueueLessonContentGeneration(ctx context.Context, lessonID uuid.UUID) (GenerationStarted, error)
	EnqueueModuleContentGeneration(ctx context.Context, moduleID uuid.UUID) (GenerationStarted, error)
	GetGenerationJob(ctx context.Context, jobID uuid.UUID) (domain.GenerationJob, error)
	GenerateCourseStructure(ctx context.Context, params GenerateStructureParams) (GenerationResult, error)
	RetryCourseStructure(ctx context.Context, params GenerateStructureParams) (GenerationResult, error)
	GenerateLessonContent(ctx context.Context, lessonID uuid.UUID) (domain.Lesson, error)
	GenerateModuleLessonContents(ctx context.Context, moduleID uuid.UUID) (domain.Module, error)
	GetGenerationStatus(ctx context.Context, requestID uuid.UUID) (GenerationStatus, error)
	GetGenerationResult(ctx context.Context, requestID uuid.UUID) (GenerationResult, error)
	RetryFullCourseGeneration(ctx context.Context, requestID uuid.UUID) (GenerationStarted, error)
}
