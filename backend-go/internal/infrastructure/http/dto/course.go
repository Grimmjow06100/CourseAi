package dto

import (
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type CourseResponse struct {
	ID                      string           `json:"id"`
	RequestID               string           `json:"requestId"`
	Language                string           `json:"language"`
	Status                  string           `json:"status"`
	InitialUserPrompt       string           `json:"initialUserPrompt"`
	Title                   string           `json:"title"`
	Synopsis                string           `json:"synopsis"`
	TargetAudience          *string          `json:"targetAudience"`
	CurrentLevel            string           `json:"currentLevel"`
	TargetLevel             string           `json:"targetLevel"`
	Prerequisites           []string         `json:"prerequisites"`
	Goals                   []string         `json:"goals"`
	AcquiredSkills          []string         `json:"acquiredSkills"`
	FinalProjectTitle       *string          `json:"finalProjectTitle"`
	FinalProjectDescription *string          `json:"finalProjectDescription"`
	FinalProjectConstraints []string         `json:"finalProjectConstraints"`
	TotalDurationMinutes    int              `json:"totalDurationMinutes"`
	Modules                 []ModuleResponse `json:"modules"`
	CreatedAt               time.Time        `json:"createdAt"`
	UpdatedAt               time.Time        `json:"updatedAt"`
}

type ModuleResponse struct {
	ID                   string           `json:"id"`
	CourseID             string           `json:"courseId"`
	Order                int              `json:"order"`
	Title                string           `json:"title"`
	Description          string           `json:"description"`
	KeyLearningPoints    []string         `json:"keyLearningPoints"`
	TotalDurationMinutes int              `json:"totalDurationMinutes"`
	Lessons              []LessonResponse `json:"lessons"`
	CreatedAt            time.Time        `json:"createdAt"`
	UpdatedAt            time.Time        `json:"updatedAt"`
}

type LessonResponse struct {
	ID                       string             `json:"id"`
	ModuleID                 string             `json:"moduleId"`
	Order                    int                `json:"order"`
	Title                    string             `json:"title"`
	Type                     string             `json:"type"`
	EstimatedDurationMinutes int                `json:"estimatedDurationMinutes"`
	LearningGoal             string             `json:"learningGoal"`
	RequiresDiagram          bool               `json:"requiresDiagram"`
	TechnicalKeywords        []string           `json:"technicalKeywords"`
	ContentMarkdown          *string            `json:"contentMarkdown"`
	Exercises                []ExerciseResponse `json:"exercises"`
	Quizzes                  []QuizResponse     `json:"quizzes"`
	HasContent               bool               `json:"hasContent"`
	HasStructuredActivities  bool               `json:"hasStructuredActivities"`
	CreatedAt                time.Time          `json:"createdAt"`
	UpdatedAt                time.Time          `json:"updatedAt"`
}

type ExerciseResponse struct {
	ID                   string         `json:"id"`
	LessonID             string         `json:"lessonId"`
	Type                 string         `json:"type"`
	Difficulty           string         `json:"difficulty"`
	Title                string         `json:"title"`
	Objective            string         `json:"objective"`
	InstructionsMarkdown string         `json:"instructionsMarkdown"`
	ContentMarkdown      string         `json:"contentMarkdown"`
	CorrectionMarkdown   string         `json:"correctionMarkdown"`
	Payload              map[string]any `json:"payload"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type QuizResponse struct {
	ID         string                 `json:"id"`
	LessonID   string                 `json:"lessonId"`
	Type       string                 `json:"type"`
	Difficulty string                 `json:"difficulty"`
	Title      string                 `json:"title"`
	Objective  string                 `json:"objective"`
	Questions  []QuizQuestionResponse `json:"questions"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type QuizQuestionResponse struct {
	Order      int                  `json:"order"`
	Type       string               `json:"type"`
	Question   string               `json:"question"`
	Options    []QuizOptionResponse `json:"options"`
	Answer     QuizAnswerResponse   `json:"answer"`
	Correction string               `json:"correction"`
}

type QuizOptionResponse struct {
	Order int    `json:"order"`
	Text  string `json:"text"`
}

type QuizAnswerResponse struct {
	Answer  *string  `json:"answer"`
	Answers []string `json:"answers"`
}

func CourseFromDomain(course domain.Course) CourseResponse {
	modules := make([]ModuleResponse, 0, len(course.Modules))
	for _, module := range course.Modules {
		modules = append(modules, ModuleFromDomain(module))
	}

	return CourseResponse{
		ID:                      course.ID.String(),
		RequestID:               course.RequestID.String(),
		Language:                string(course.Language),
		Status:                  string(course.Status),
		InitialUserPrompt:       course.InitialUserPrompt,
		Title:                   course.Title,
		Synopsis:                course.Synopsis,
		TargetAudience:          course.TargetAudience,
		CurrentLevel:            string(course.CurrentLevel),
		TargetLevel:             string(course.TargetLevel),
		Prerequisites:           course.Prerequisites,
		Goals:                   course.Goals,
		AcquiredSkills:          course.AcquiredSkills,
		FinalProjectTitle:       course.FinalProjectTitle,
		FinalProjectDescription: course.FinalProjectDescription,
		FinalProjectConstraints: course.FinalProjectConstraints,
		TotalDurationMinutes:    course.TotalDurationMinutes(),
		Modules:                 modules,
		CreatedAt:               course.CreatedAt,
		UpdatedAt:               course.UpdatedAt,
	}
}

func ModuleFromDomain(module domain.Module) ModuleResponse {
	lessons := make([]LessonResponse, 0, len(module.Lessons))
	for _, lesson := range module.Lessons {
		lessons = append(lessons, LessonFromDomain(lesson))
	}

	return ModuleResponse{
		ID:                   module.ID.String(),
		CourseID:             module.CourseID.String(),
		Order:                module.Order,
		Title:                module.Title,
		Description:          module.Description,
		KeyLearningPoints:    module.KeyLearningPoints,
		TotalDurationMinutes: module.TotalDurationMinutes(),
		Lessons:              lessons,
		CreatedAt:            module.CreatedAt,
		UpdatedAt:            module.UpdatedAt,
	}
}

func LessonFromDomain(lesson domain.Lesson) LessonResponse {
	exercises := make([]ExerciseResponse, 0, len(lesson.Exercises))
	for _, exercise := range lesson.Exercises {
		exercises = append(exercises, ExerciseFromDomain(exercise))
	}

	quizzes := make([]QuizResponse, 0, len(lesson.Quizzes))
	for _, quiz := range lesson.Quizzes {
		quizzes = append(quizzes, QuizFromDomain(quiz))
	}

	return LessonResponse{
		ID:                       lesson.ID.String(),
		ModuleID:                 lesson.ModuleID.String(),
		Order:                    lesson.Order,
		Title:                    lesson.Title,
		Type:                     string(lesson.Type),
		EstimatedDurationMinutes: lesson.EstimatedDurationMinutes,
		LearningGoal:             lesson.LearningGoal,
		RequiresDiagram:          lesson.RequiresDiagram,
		TechnicalKeywords:        lesson.TechnicalKeywords,
		ContentMarkdown:          lesson.ContentMarkdown,
		Exercises:                exercises,
		Quizzes:                  quizzes,
		HasContent:               lesson.HasContent(),
		HasStructuredActivities:  lesson.HasStructuredActivities(),
		CreatedAt:                lesson.CreatedAt,
		UpdatedAt:                lesson.UpdatedAt,
	}
}

func ExerciseFromDomain(exercise domain.Exercise) ExerciseResponse {
	return ExerciseResponse{
		ID:                   exercise.ID.String(),
		LessonID:             exercise.LessonID.String(),
		Type:                 string(exercise.Type),
		Difficulty:           string(exercise.Difficulty),
		Title:                exercise.Title,
		Objective:            exercise.Objective,
		InstructionsMarkdown: exercise.InstructionsMarkdown,
		ContentMarkdown:      exercise.ContentMarkdown,
		CorrectionMarkdown:   exercise.CorrectionMarkdown,
		Payload:              exercisePayloadFromDomain(exercise.Payload),
		CreatedAt:            exercise.CreatedAt,
		UpdatedAt:            exercise.UpdatedAt,
	}
}

func QuizFromDomain(quiz domain.Quiz) QuizResponse {
	questions := make([]QuizQuestionResponse, 0, len(quiz.Questions))
	for _, question := range quiz.Questions {
		questions = append(questions, QuizQuestionFromDomain(question))
	}

	return QuizResponse{
		ID:         quiz.ID.String(),
		LessonID:   quiz.LessonID.String(),
		Type:       string(quiz.Type),
		Difficulty: string(quiz.Difficulty),
		Title:      quiz.Title,
		Objective:  quiz.Objective,
		Questions:  questions,
		CreatedAt:  quiz.CreatedAt,
		UpdatedAt:  quiz.UpdatedAt,
	}
}

func QuizQuestionFromDomain(question domain.QuizQuestion) QuizQuestionResponse {
	options := make([]QuizOptionResponse, 0, len(question.Options))
	for _, option := range question.Options {
		options = append(options, QuizOptionResponse{
			Order: option.Order,
			Text:  option.Text,
		})
	}
	answers := question.Answer.Answers
	if answers == nil {
		answers = []string{}
	}

	return QuizQuestionResponse{
		Order:      question.Order,
		Type:       string(question.Type),
		Question:   question.Question,
		Options:    options,
		Answer:     QuizAnswerResponse{Answer: question.Answer.Answer, Answers: answers},
		Correction: question.Correction,
	}
}

func exercisePayloadFromDomain(payload domain.ExercisePayload) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	values := make(map[string]any, len(payload))
	for key, value := range payload {
		values[key] = value
	}
	return values
}

func CoursePageFromDomain(page contract.Page[domain.Course]) PageResponse[CourseResponse] {
	items := make([]CourseResponse, 0, len(page.Items))
	for _, course := range page.Items {
		items = append(items, CourseFromDomain(course))
	}

	return PageResponse[CourseResponse]{
		Items:       items,
		Page:        page.Page,
		PageSize:    page.PageSize,
		TotalItems:  page.TotalItems,
		TotalPages:  page.TotalPages,
		HasNext:     page.HasNext,
		HasPrevious: page.HasPrevious,
	}
}
