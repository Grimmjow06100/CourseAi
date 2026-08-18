package dto

import (
	"reflect"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

func TestCourseFromDomainMapsNestedAggregate(t *testing.T) {
	t.Parallel()

	now := time.Unix(100, 0).UTC()
	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	answer := "ls"
	content := "# Linux"
	payload := domain.ExercisePayload{"command": "ls"}
	lesson := domain.Lesson{
		ID: lessonID, ModuleID: moduleID, Order: 1, Title: "Commands", Type: domain.LessonTypeMixed,
		EstimatedDurationMinutes: 30, LearningGoal: "Use commands", ContentMarkdown: &content,
		Exercises: []domain.Exercise{{
			ID: uuid.New(), LessonID: lessonID, Type: domain.ExerciseTypeCommandLine,
			Difficulty: domain.DifficultyBeginner, Title: "Exercise", Payload: payload,
		}},
		Quizzes: []domain.Quiz{{
			ID: uuid.New(), LessonID: lessonID, Type: domain.QuizTypeSingleChoice,
			Difficulty: domain.DifficultyBeginner, Title: "Quiz",
			Questions: []domain.QuizQuestion{{
				Order: 1, Type: domain.QuizQuestionTypeSingleChoice, Question: "Command?",
				Options: []domain.QuizOption{{Order: 1, Text: "pwd"}, {Order: 2, Text: "ls"}},
				Answer:  domain.QuizAnswer{Answer: &answer}, Correction: "ls lists files",
			}},
		}},
		CreatedAt: now, UpdatedAt: now,
	}
	course := domain.Course{
		ID: courseID, RequestID: uuid.New(), Language: domain.CourseLanguageEN,
		Status: domain.CourseStatusContentGenerating, Title: "Linux", CurrentLevel: domain.LevelBeginner,
		TargetLevel: domain.LevelAdvanced, Modules: []domain.Module{{
			ID: moduleID, CourseID: courseID, Order: 1, Title: "Basics", Lessons: []domain.Lesson{lesson},
		}},
	}

	response := CourseFromDomain(course)
	if response.ID != courseID.String() || response.TotalDurationMinutes != 30 || len(response.Modules) != 1 {
		t.Fatalf("unexpected course response: %+v", response)
	}
	gotLesson := response.Modules[0].Lessons[0]
	if !gotLesson.HasContent || !gotLesson.HasStructuredActivities || len(gotLesson.Exercises) != 1 || len(gotLesson.Quizzes) != 1 {
		t.Fatalf("unexpected lesson response: %+v", gotLesson)
	}
	if gotLesson.Quizzes[0].Questions[0].Answer.Answer == nil || *gotLesson.Quizzes[0].Questions[0].Answer.Answer != "ls" {
		t.Fatal("quiz answer was not mapped")
	}
	gotLesson.Exercises[0].Payload["new"] = true
	if _, exists := payload["new"]; exists {
		t.Fatal("exercise payload should be copied at the HTTP boundary")
	}
}

func TestCoursePageFromDomainPreservesPagination(t *testing.T) {
	t.Parallel()

	page := contract.Page[domain.Course]{
		Items: []domain.Course{{ID: uuid.New()}}, Page: 2, PageSize: 10,
		TotalItems: 21, TotalPages: 3, HasNext: true, HasPrevious: true,
	}
	response := CoursePageFromDomain(page)
	if len(response.Items) != 1 || response.Page != 2 || response.TotalPages != 3 || !response.HasNext || !response.HasPrevious {
		t.Fatalf("unexpected page response: %+v", response)
	}
}

func TestQuizQuestionFromDomainUsesEmptyAnswersArray(t *testing.T) {
	t.Parallel()

	response := QuizQuestionFromDomain(domain.QuizQuestion{Answer: domain.QuizAnswer{Answer: pointer.To("yes")}})
	if response.Answer.Answers == nil || !reflect.DeepEqual(response.Answer.Answers, []string{}) {
		t.Fatalf("Answers = %#v, want non-nil empty slice", response.Answer.Answers)
	}
}
