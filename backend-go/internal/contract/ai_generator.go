package contract

import (
	"context"
	"encoding/json"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type AnalysisInput struct {
	Prompt string
}

type AnalysisOutput struct {
	Summary domain.AnalysisSummary
	Raw     json.RawMessage
}

type ArchitectureInput struct {
	Request domain.GenerationRequest
}

type ArchitectureOutput struct {
	Course domain.Course
	Raw    json.RawMessage
}

type LessonPlanInput struct {
	Course domain.Course
	Module domain.Module
}

type LessonPlanOutput struct {
	Lessons []domain.Lesson
	Raw     json.RawMessage
}

type LessonContentInput struct {
	Course domain.Course
	Module domain.Module
	Lesson domain.Lesson
}

type LessonContentOutput struct {
	Lesson          domain.Lesson
	ContentMarkdown string
	Raw             json.RawMessage
}

type CourseAIGenerator interface {
	AnalyzePrompt(ctx context.Context, input AnalysisInput) (AnalysisOutput, error)
	GenerateArchitecture(ctx context.Context, input ArchitectureInput) (ArchitectureOutput, error)
	GenerateLessonPlan(ctx context.Context, input LessonPlanInput) (LessonPlanOutput, error)
	GenerateLessonContent(ctx context.Context, input LessonContentInput) (LessonContentOutput, error)
}
