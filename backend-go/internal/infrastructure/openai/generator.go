package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const (
	defaultModel           = "gpt-5.6"
	defaultMaxOutputTokens = int64(12000)
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
	client          *openaisdk.Client
	prompts         contract.PromptStore
	model           string
	maxOutputTokens int64
}

func NewCourseAIGenerator(client *openaisdk.Client, prompts contract.PromptStore, config Config) *CourseAIGenerator {
	model := strings.TrimSpace(config.Model)
	if model == "" {
		model = defaultModel
	}

	maxOutputTokens := config.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultMaxOutputTokens
	}

	return &CourseAIGenerator{
		client:          client,
		prompts:         prompts,
		model:           model,
		maxOutputTokens: maxOutputTokens,
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

	var response analysisResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.AnalysisOutput{}, fmt.Errorf("%w: analysis json: %v", ErrInvalidModelOutput, err)
	}

	summary, err := response.toDomain()
	if err != nil {
		return contract.AnalysisOutput{}, err
	}

	return contract.AnalysisOutput{Summary: summary, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateArchitecture(ctx context.Context, input contract.ArchitectureInput) (contract.ArchitectureOutput, error) {
	payload := architecturePromptInputFromRequest(input.Request)
	raw, err := g.callStructuredJSON(ctx, "architecture", payload, "architecture_response", architectureSchema())
	if err != nil {
		return contract.ArchitectureOutput{}, err
	}

	var response architectureResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.ArchitectureOutput{}, fmt.Errorf("%w: architecture json: %v", ErrInvalidModelOutput, err)
	}

	course, err := response.toDomain(input.Request)
	if err != nil {
		return contract.ArchitectureOutput{}, err
	}

	return contract.ArchitectureOutput{Course: course, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateLessonPlan(ctx context.Context, input contract.LessonPlanInput) (contract.LessonPlanOutput, error) {
	payload := lessonPlanPromptInputFromDomain(input.Course, input.Module)
	raw, err := g.callStructuredJSON(ctx, "lessons", payload, "lessons_response", lessonsSchema())
	if err != nil {
		return contract.LessonPlanOutput{}, err
	}

	var response lessonsResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.LessonPlanOutput{}, fmt.Errorf("%w: lessons json: %v", ErrInvalidModelOutput, err)
	}

	lessons, err := response.toDomain(input.Module.ID)
	if err != nil {
		return contract.LessonPlanOutput{}, err
	}

	return contract.LessonPlanOutput{Lessons: lessons, Raw: raw}, nil
}

func (g *CourseAIGenerator) GenerateLessonContent(ctx context.Context, input contract.LessonContentInput) (contract.LessonContentOutput, error) {
	payload := lessonContentPromptInputFromDomain(input.Course, input.Module, input.Lesson)
	raw, err := g.callStructuredJSON(ctx, "lesson-content", payload, "lesson_content_response", lessonContentSchema())
	if err != nil {
		return contract.LessonContentOutput{}, err
	}

	var response lessonContentResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: lesson content json: %v", ErrInvalidModelOutput, err)
	}

	content := strings.TrimSpace(response.ContentMarkdown)
	if content == "" {
		return contract.LessonContentOutput{}, fmt.Errorf("%w: contentMarkdown is blank", ErrInvalidModelOutput)
	}

	lesson := input.Lesson
	if err := lesson.AttachContent(content); err != nil {
		return contract.LessonContentOutput{}, err
	}

	return contract.LessonContentOutput{Lesson: lesson, ContentMarkdown: content, Raw: raw}, nil
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
		Model:        g.model,
		Instructions: openaisdk.String(prompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openaisdk.String(string(inputJSON)),
		},
		Store: openaisdk.Bool(false),
		Text: responses.ResponseTextConfigParam{
			Format: format,
		},
		MaxOutputTokens: openaisdk.Int(g.maxOutputTokens),
	}

	response, err := g.client.Responses.New(ctx, params)
	if err != nil {
		return nil, err
	}

	output := strings.TrimSpace(response.OutputText())
	if output == "" {
		return nil, fmt.Errorf("%w: empty model output", ErrInvalidModelOutput)
	}
	if !json.Valid([]byte(output)) {
		return nil, fmt.Errorf("%w: output is not valid json", ErrInvalidModelOutput)
	}

	return json.RawMessage(output), nil
}

type analysisResponse struct {
	IsOutOfScope           bool                           `json:"isOutOfScope"`
	ErrorMessage           *string                        `json:"errorMessage"`
	WarningMessage         *string                        `json:"warningMessage"`
	SuggestedTitle         string                         `json:"suggestedTitle"`
	ShortSynopsis          string                         `json:"shortSynopsis"`
	DetectedCurrentLevel   string                         `json:"detectedCurrentLevel"`
	DetectedTargetLevel    string                         `json:"detectedTargetLevel"`
	DetectedGoal           string                         `json:"detectedGoal"`
	DetectedLanguage       string                         `json:"detectedLanguage"`
	ClarificationQuestions []clarificationQuestionPayload `json:"clarificationQuestions"`
}

type clarificationQuestionPayload struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func (r analysisResponse) toDomain() (domain.AnalysisSummary, error) {
	currentLevel, err := parseLevel(r.DetectedCurrentLevel)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}
	targetLevel, err := parseLevel(r.DetectedTargetLevel)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}
	language, err := domain.ParseCourseLanguage(r.DetectedLanguage)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}

	questions := make([]domain.ClarificationQuestion, 0, len(r.ClarificationQuestions))
	for _, question := range r.ClarificationQuestions {
		questions = append(questions, domain.ClarificationQuestion{
			ID:       question.ID,
			Question: question.Question,
			Options:  question.Options,
		})
	}

	goal := strings.TrimSpace(r.DetectedGoal)
	if goal == "" {
		goal = "unknown"
	}

	return domain.AnalysisSummary{
		IsOutOfScope:           r.IsOutOfScope,
		ErrorMessage:           cleanStringPtr(r.ErrorMessage),
		WarningMessage:         cleanStringPtr(r.WarningMessage),
		SuggestedTitle:         stringPtr(r.SuggestedTitle),
		ShortSynopsis:          stringPtr(r.ShortSynopsis),
		DetectedCurrentLevel:   &currentLevel,
		DetectedTargetLevel:    &targetLevel,
		DetectedGoal:           &goal,
		DetectedLanguage:       &language,
		ClarificationQuestions: questions,
	}, nil
}

type architecturePromptInput struct {
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	CurrentLevel string   `json:"currentLevel"`
	TargetLevel  string   `json:"targetLevel"`
	Goals        []string `json:"goals"`
	Language     string   `json:"language"`
}

func architecturePromptInputFromRequest(request domain.GenerationRequest) architecturePromptInput {
	title := request.InitialUserPrompt
	if request.SuggestedTitle != nil {
		title = *request.SuggestedTitle
	}

	synopsis := request.InitialUserPrompt
	if request.ShortSynopsis != nil {
		synopsis = *request.ShortSynopsis
	}

	currentLevel := string(domain.LevelUnknown)
	if request.DetectedCurrentLevel != nil {
		currentLevel = string(*request.DetectedCurrentLevel)
	}
	targetLevel := string(domain.LevelUnknown)
	if request.DetectedTargetLevel != nil {
		targetLevel = string(*request.DetectedTargetLevel)
	}

	language := string(domain.CourseLanguageFR)
	if request.DetectedLanguage != nil {
		language = string(*request.DetectedLanguage)
	}

	goals := []string{request.InitialUserPrompt}
	if request.DetectedGoal != nil && !isUnknown(*request.DetectedGoal) {
		goals = []string{*request.DetectedGoal}
	}

	return architecturePromptInput{
		Title:        title,
		Synopsis:     synopsis,
		CurrentLevel: currentLevel,
		TargetLevel:  targetLevel,
		Goals:        goals,
		Language:     language,
	}
}

type architectureResponse struct {
	Title          string                   `json:"title"`
	Synopsis       string                   `json:"synopsis"`
	TargetAudience string                   `json:"targetAudience"`
	Prerequisites  []string                 `json:"prerequisites"`
	Goals          []string                 `json:"goals"`
	AcquiredSkills []string                 `json:"acquiredSkills"`
	Modules        []architectureModule     `json:"modules"`
	FinalProject   architectureFinalProject `json:"finalProject"`
}

type architectureModule struct {
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	KeyLearningPoints []string `json:"keyLearningPoints"`
}

type architectureFinalProject struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Constraints []string `json:"constraints"`
}

func (r architectureResponse) toDomain(request domain.GenerationRequest) (domain.Course, error) {
	language := domain.CourseLanguageFR
	if request.DetectedLanguage != nil {
		language = *request.DetectedLanguage
	}
	currentLevel := domain.LevelUnknown
	if request.DetectedCurrentLevel != nil {
		currentLevel = *request.DetectedCurrentLevel
	}
	targetLevel := domain.LevelUnknown
	if request.DetectedTargetLevel != nil {
		targetLevel = *request.DetectedTargetLevel
	}

	course := domain.Course{
		RequestID:               request.ID,
		Language:                language,
		InitialUserPrompt:       request.InitialUserPrompt,
		Title:                   r.Title,
		Synopsis:                r.Synopsis,
		TargetAudience:          stringPtr(r.TargetAudience),
		CurrentLevel:            currentLevel,
		TargetLevel:             targetLevel,
		Prerequisites:           r.Prerequisites,
		Goals:                   r.Goals,
		AcquiredSkills:          r.AcquiredSkills,
		FinalProjectTitle:       stringPtr(r.FinalProject.Title),
		FinalProjectDescription: stringPtr(r.FinalProject.Description),
		FinalProjectConstraints: r.FinalProject.Constraints,
		Modules:                 make([]domain.Module, 0, len(r.Modules)),
	}

	for _, module := range r.Modules {
		course.Modules = append(course.Modules, domain.Module{
			Order:             module.Order,
			Title:             module.Title,
			Description:       module.Description,
			KeyLearningPoints: module.KeyLearningPoints,
		})
	}

	return course, nil
}

type lessonPlanPromptInput struct {
	CourseContext     lessonPlanCourseContext `json:"courseContext"`
	ModuleToExpand    lessonPlanModule        `json:"moduleToExpand"`
	GlobalPlanSummary []string                `json:"globalPlanSummary"`
}

type lessonPlanCourseContext struct {
	Title          string                 `json:"title"`
	Synopsis       string                 `json:"synopsis"`
	TargetAudience *string                `json:"targetAudience"`
	Prerequisites  []string               `json:"prerequisites"`
	Goals          []string               `json:"goals"`
	AcquiredSkills []string               `json:"acquiredSkills"`
	FinalProject   lessonPlanFinalProject `json:"finalProject"`
}

type lessonPlanFinalProject struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Constraints []string `json:"constraints"`
}

type lessonPlanModule struct {
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	KeyLearningPoints []string `json:"keyLearningPoints"`
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

type lessonsResponse struct {
	ModuleOrder int             `json:"moduleOrder"`
	ModuleTitle string          `json:"moduleTitle"`
	Lessons     []lessonPayload `json:"lessons"`
}

type lessonPayload struct {
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Type              string   `json:"type"`
	EstimatedDuration int      `json:"estimatedDuration"`
	LearningGoal      string   `json:"learningGoal"`
	RequiresDiagram   bool     `json:"requiresDiagram"`
	TechnicalKeywords []string `json:"technicalKeywords"`
}

func (r lessonsResponse) toDomain(moduleID uuid.UUID) ([]domain.Lesson, error) {
	lessons := make([]domain.Lesson, 0, len(r.Lessons))
	for _, lesson := range r.Lessons {
		lessonType, err := domain.ParseLessonType(lesson.Type)
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, domain.Lesson{
			ModuleID:                 moduleID,
			Order:                    lesson.Order,
			Title:                    lesson.Title,
			Type:                     lessonType,
			EstimatedDurationMinutes: lesson.EstimatedDuration,
			LearningGoal:             lesson.LearningGoal,
			RequiresDiagram:          lesson.RequiresDiagram,
			TechnicalKeywords:        lesson.TechnicalKeywords,
		})
	}
	return lessons, nil
}

type lessonContentPromptInput struct {
	Course lessonContentCourse `json:"course"`
	Module lessonContentModule `json:"module"`
	Lesson lessonContentLesson `json:"lesson"`
}

type lessonContentCourse struct {
	ID        string   `json:"id"`
	Language  string   `json:"language"`
	Title     string   `json:"title"`
	Synopsis  string   `json:"synopsis"`
	Goals     []string `json:"goals"`
	LevelFrom string   `json:"currentLevel"`
	LevelTo   string   `json:"targetLevel"`
}

type lessonContentModule struct {
	ID                string   `json:"id"`
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	KeyLearningPoints []string `json:"keyLearningPoints"`
}

type lessonContentLesson struct {
	ID                       string   `json:"id"`
	Order                    int      `json:"order"`
	Title                    string   `json:"title"`
	Type                     string   `json:"type"`
	EstimatedDurationMinutes int      `json:"estimatedDurationMinutes"`
	LearningGoal             string   `json:"learningGoal"`
	RequiresDiagram          bool     `json:"requiresDiagram"`
	TechnicalKeywords        []string `json:"technicalKeywords"`
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

type lessonContentResponse struct {
	ContentMarkdown string `json:"contentMarkdown"`
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

var _ contract.CourseAIGenerator = (*CourseAIGenerator)(nil)
