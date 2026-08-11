package dto

import (
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type LessonPlanPromptInput struct {
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

func LessonPlanPromptInputFromDomain(course domain.Course, module domain.Module) LessonPlanPromptInput {
	summary := make([]string, 0, len(course.Modules))
	for _, item := range course.Modules {
		summary = append(summary, fmt.Sprintf("Module %d: %s", item.Order, item.Title))
	}

	return LessonPlanPromptInput{
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

type LessonsResponse struct {
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

func (r LessonsResponse) ToDomain(moduleID uuid.UUID) ([]domain.Lesson, error) {
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
