package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
)

func TestCourseAIGeneratorValidatesInputsAndDependencies(t *testing.T) {
	t.Parallel()

	generator := NewCourseAIGenerator(nil, nil, Config{Model: "test", MaxOutputTokens: 100})
	if generator.conf.Model != "test" || generator.conf.MaxOutputTokens != 100 {
		t.Fatalf("constructor did not preserve config: %+v", generator.conf)
	}
	if _, err := generator.AnalyzePrompt(context.Background(), contract.AnalysisInput{}); !errors.Is(err, ErrMissingPromptInput) {
		t.Fatalf("blank analysis error = %v", err)
	}
	if _, err := generator.AnalyzePrompt(context.Background(), contract.AnalysisInput{Prompt: "Linux"}); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("analysis dependency error = %v", err)
	}
	if _, err := generator.GenerateArchitecture(context.Background(), contract.ArchitectureInput{}); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("architecture dependency error = %v", err)
	}
	if _, err := generator.GenerateLessonPlan(context.Background(), contract.LessonPlanInput{}); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("lesson plan dependency error = %v", err)
	}
	if _, err := generator.GenerateLessonContent(context.Background(), contract.LessonContentInput{}); !errors.Is(err, ErrMissingClient) {
		t.Fatalf("lesson content dependency error = %v", err)
	}
}
