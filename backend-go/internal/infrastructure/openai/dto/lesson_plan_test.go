package dto

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestLessonPlanPromptInputFromDomain(t *testing.T) {
	t.Parallel()

	module := domain.Module{ID: uuid.New(), Order: 2, Title: "Shell", Description: "Commands", KeyLearningPoints: []string{"pipes"}}
	course := domain.Course{Title: "Linux", Synopsis: "Course", Modules: []domain.Module{{Order: 1, Title: "Basics"}, module}}
	input := LessonPlanPromptInputFromDomain(course, module)
	if input.CourseContext.Title != "Linux" || input.ModuleToExpand.Title != "Shell" ||
		!reflect.DeepEqual(input.GlobalPlanSummary, []string{"Module 1: Basics", "Module 2: Shell"}) {
		t.Fatalf("unexpected prompt input: %+v", input)
	}
}

func TestLessonsResponseToDomain(t *testing.T) {
	t.Parallel()

	moduleID := uuid.New()
	response := LessonsResponse{Lessons: []lessonPayload{{
		Order: 1, Title: "Filesystem", Type: "theory", EstimatedDuration: 20,
		LearningGoal: "Navigate", RequiresDiagram: true, TechnicalKeywords: []string{"inode"},
	}}}
	lessons, err := response.ToDomain(moduleID)
	if err != nil {
		t.Fatalf("ToDomain() error = %v", err)
	}
	if len(lessons) != 1 || lessons[0].ModuleID != moduleID || lessons[0].Type != domain.LessonTypeTheory || lessons[0].EstimatedDurationMinutes != 20 {
		t.Fatalf("unexpected lessons: %+v", lessons)
	}

	response.Lessons[0].Type = "video"
	if _, err := response.ToDomain(moduleID); !errors.Is(err, domain.ErrInvalidLessonType) {
		t.Fatalf("invalid type error = %v", err)
	}
}
