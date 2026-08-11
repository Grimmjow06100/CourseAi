package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	openaidto "github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/openai/dto"
	openaisdk "github.com/openai/openai-go/v3"
)

var (
	ErrMissingClient      = errors.New("openai client is missing")
	ErrMissingPromptStore = errors.New("prompt store is missing")
	ErrPromptNotFound     = errors.New("prompt not found")
	ErrInvalidModelOutput = errors.New("invalid model output")
	ErrMissingPromptInput = errors.New("prompt input is missing")
)

type Config struct {
	Model           string
	MaxOutputTokens int64
}

type CourseAIGenerator struct {
	client  *openaisdk.Client
	prompts contract.PromptStore
	conf    Config
}

func NewCourseAIGenerator(client *openaisdk.Client, prompts contract.PromptStore, config Config) *CourseAIGenerator {
	return &CourseAIGenerator{
		client:  client,
		prompts: prompts,
		conf:    config,
	}
}

func (g *CourseAIGenerator) AnalyzePrompt(ctx context.Context, input contract.AnalysisInput) (contract.AnalysisOutput, error) {
	if strings.TrimSpace(input.Prompt) == "" {
		return contract.AnalysisOutput{}, ErrMissingPromptInput
	}

	payload := map[string]string{"prompt": input.Prompt}
	raw, err := g.callStructuredJSON(ctx, "analysis", payload, "analysis_response", analysisSchema())
	if err != nil {
		return contract.AnalysisOutput{}, err
	}

	var response openaidto.AnalysisResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.AnalysisOutput{}, fmt.Errorf("%w: analysis json: %v", ErrInvalidModelOutput, err)
	}

	summary, err := response.ToDomain()
	if err != nil {
		return contract.AnalysisOutput{}, err
	}

	return contract.AnalysisOutput{Summary: summary, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateArchitecture(ctx context.Context, input contract.ArchitectureInput) (contract.ArchitectureOutput, error) {
	payload := openaidto.ArchitecturePromptInputFromContract(input)
	raw, err := g.callStructuredJSON(ctx, "architecture", payload, "architecture_response", architectureSchema())
	if err != nil {
		return contract.ArchitectureOutput{}, err
	}

	var response openaidto.ArchitectureResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.ArchitectureOutput{}, fmt.Errorf("%w: architecture json: %v", ErrInvalidModelOutput, err)
	}

	course, err := response.ToDomain(input)
	if err != nil {
		return contract.ArchitectureOutput{}, err
	}

	return contract.ArchitectureOutput{Course: course, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateLessonPlan(ctx context.Context, input contract.LessonPlanInput) (contract.LessonPlanOutput, error) {
	payload := openaidto.LessonPlanPromptInputFromDomain(input.Course, input.Module)
	raw, err := g.callStructuredJSON(ctx, "lessons", payload, "lessons_response", lessonsSchema())
	if err != nil {
		return contract.LessonPlanOutput{}, err
	}

	var response openaidto.LessonsResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.LessonPlanOutput{}, fmt.Errorf("%w: lessons json: %v", ErrInvalidModelOutput, err)
	}

	lessons, err := response.ToDomain(input.Module.ID)
	if err != nil {
		return contract.LessonPlanOutput{}, err
	}

	return contract.LessonPlanOutput{Lessons: lessons, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateLessonContent(ctx context.Context, input contract.LessonContentInput) (contract.LessonContentOutput, error) {
	payload := openaidto.LessonContentPromptInputFromDomain(input.Course, input.Module, input.Lesson)
	raw, err := g.callStructuredJSON(ctx, "lesson-content", payload, "lesson_content_response", lessonContentSchema())
	if err != nil {
		return contract.LessonContentOutput{}, err
	}

	var response openaidto.LessonContentResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: lesson content json: %v", ErrInvalidModelOutput, err)
	}

	content := strings.TrimSpace(response.ContentMarkdown)
	if content == "" {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: contentMarkdown is blank", ErrInvalidModelOutput)
	}

	exercises, err := response.ExercisesToDomain(input.Lesson.ID)
	if err != nil {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: exercises: %v", ErrInvalidModelOutput, err)
	}
	quizzes, err := response.QuizzesToDomain(input.Lesson.ID)
	if err != nil {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: quizzes: %v", ErrInvalidModelOutput, err)
	}
	if err := domain.ValidateLessonActivitiesForType(input.Lesson.Type, exercises, quizzes); err != nil {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: activity policy: %v", ErrInvalidModelOutput, err)
	}

	lesson := input.Lesson
	if err := lesson.AttachContent(content); err != nil {
		return contract.LessonContentOutput{}, err
	}
	lesson.Exercises = exercises
	lesson.Quizzes = quizzes

	return contract.LessonContentOutput{
		Lesson:          lesson,
		ContentMarkdown: content,
		Exercises:       exercises,
		Quizzes:         quizzes,
		Raw:             raw,
	}, nil
}
