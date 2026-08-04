package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)
func extractJSONPayload(output string) (json.RawMessage, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, errors.New("empty model output")
	}

	if json.Valid([]byte(output)) {
		return json.RawMessage(output), nil
	}

	if fenced, ok := extractFencedJSON(output); ok && json.Valid([]byte(fenced)) {
		return json.RawMessage(fenced), nil
	}

	if balanced, ok := extractBalancedJSON(output); ok && json.Valid([]byte(balanced)) {
		return json.RawMessage(balanced), nil
	}

	return nil, fmt.Errorf("output is not valid json: %s", outputPreview(output))
}

func extractFencedJSON(output string) (string, bool) {
	start := strings.Index(output, "```")
	if start == -1 {
		return "", false
	}

	content := output[start+3:]
	lineBreak := strings.IndexAny(content, "\r\n")
	if lineBreak == -1 {
		return "", false
	}
	content = content[lineBreak+1:]

	end := strings.LastIndex(content, "```")
	if end == -1 {
		return "", false
	}

	return strings.TrimSpace(content[:end]), true
}

func extractBalancedJSON(output string) (string, bool) {
	start := -1
	for i := 0; i < len(output); i++ {
		if output[i] == '{' || output[i] == '[' {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}

	stack := make([]byte, 0, 8)
	inString := false
	escaped := false

	for i := start; i < len(output); i++ {
		char := output[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		switch char {
		case '"':
			inString = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) == 0 || stack[len(stack)-1] != char {
				return "", false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return strings.TrimSpace(output[start : i+1]), true
			}
		}
	}

	return "", false
}

func outputPreview(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	const maxPreviewLength = 240
	if len(value) <= maxPreviewLength {
		return value
	}
	return value[:maxPreviewLength] + "..."
}


func (g *CourseAIGenerator) callStructuredJSON(ctx context.Context, promptName string, payload any, schemaName string, schema map[string]any) (json.RawMessage, error) {
	if g == nil || g.client == nil {
		return nil, ErrMissingClient
	}
	if g.prompts == nil {
		return nil, ErrMissingPromptStore
	}

	prompt, ok := g.prompts.Get(promptName)
	if !ok || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: %s", ErrPromptNotFound, promptName)
	}

	inputJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal prompt payload: %w", err)
	}

	format := responses.ResponseFormatTextConfigParamOfJSONSchema(schemaName, schema)
	if format.OfJSONSchema != nil {
		format.OfJSONSchema.Strict = openaisdk.Bool(true)
	}

	params := responses.ResponseNewParams{
		Model: "gpt-5.6-sol",
		Instructions: openaisdk.String(prompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openaisdk.String(string(inputJSON)),
		},
		Store: openaisdk.Bool(false),
		Text: responses.ResponseTextConfigParam{
			Format: format,
		},
	}

	response, err := g.client.Responses.New(ctx, params)
	if err != nil {
		return nil, err
	}

	output := strings.TrimSpace(response.OutputText())
	raw, err := extractJSONPayload(output)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidModelOutput, err)
	}

	return raw, nil
}

func lessonPlanPromptInputFromDomain(course domain.Course, module domain.Module) lessonPlanPromptInput {
	summary := make([]string, 0, len(course.Modules))
	for _, item := range course.Modules {
		summary = append(summary, fmt.Sprintf("Module %d: %s", item.Order, item.Title))
	}

	return lessonPlanPromptInput{
		CourseContext: lessonPlanCourseContext{
			Title:          course.Title,
			Synopsis:       course.Synopsis,
			TargetAudience: course.TargetAudience,
			Prerequisites:  course.Prerequisites,
			Goals:          course.Goals,
			AcquiredSkills: course.AcquiredSkills,
			FinalProject: lessonPlanFinalProject{
				Title:       course.FinalProjectTitle,
				Description: course.FinalProjectDescription,
				Constraints: course.FinalProjectConstraints,
			},
		},
		ModuleToExpand: lessonPlanModule{
			Order:             module.Order,
			Title:             module.Title,
			Description:       module.Description,
			KeyLearningPoints: module.KeyLearningPoints,
		},
		GlobalPlanSummary: summary,
	}
}

func lessonContentPromptInputFromDomain(course domain.Course, module domain.Module, lesson domain.Lesson) lessonContentPromptInput {
	return lessonContentPromptInput{
		Course: lessonContentCourse{
			ID:        course.ID.String(),
			Language:  string(course.Language),
			Title:     course.Title,
			Synopsis:  course.Synopsis,
			Goals:     course.Goals,
			LevelFrom: string(course.CurrentLevel),
			LevelTo:   string(course.TargetLevel),
		},
		Module: lessonContentModule{
			ID:                module.ID.String(),
			Order:             module.Order,
			Title:             module.Title,
			Description:       module.Description,
			KeyLearningPoints: module.KeyLearningPoints,
		},
		Lesson: lessonContentLesson{
			ID:                       lesson.ID.String(),
			Order:                    lesson.Order,
			Title:                    lesson.Title,
			Type:                     string(lesson.Type),
			EstimatedDurationMinutes: lesson.EstimatedDurationMinutes,
			LearningGoal:             lesson.LearningGoal,
			RequiresDiagram:          lesson.RequiresDiagram,
			TechnicalKeywords:        lesson.TechnicalKeywords,
		},
	}
}


func parseLevel(value string) (domain.Level, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "unknow" {
		value = "unknown"
	}
	return domain.ParseLevel(value)
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	return stringPtr(*value)
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func isUnknown(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "" || value == "unknown" || value == "unknow"
}

