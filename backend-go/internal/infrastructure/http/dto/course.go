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
	ID                       string    `json:"id"`
	ModuleID                 string    `json:"moduleId"`
	Order                    int       `json:"order"`
	Title                    string    `json:"title"`
	Type                     string    `json:"type"`
	EstimatedDurationMinutes int       `json:"estimatedDurationMinutes"`
	LearningGoal             string    `json:"learningGoal"`
	RequiresDiagram          bool      `json:"requiresDiagram"`
	TechnicalKeywords        []string  `json:"technicalKeywords"`
	ContentMarkdown          *string   `json:"contentMarkdown"`
	HasContent               bool      `json:"hasContent"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
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
		HasContent:               lesson.HasContent(),
		CreatedAt:                lesson.CreatedAt,
		UpdatedAt:                lesson.UpdatedAt,
	}
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
