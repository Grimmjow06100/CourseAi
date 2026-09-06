package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
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
	starterCode := "ls --"
	payload := domain.ExercisePayload{
		"tasks": []string{"List files"}, "resources": []string{"terminal"},
		"starterCode": &starterCode, "expectedOutput": "README.md", "hints": []string{"Use ls"},
	}
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
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal public course response: %v", err)
	}
	for _, forbiddenKey := range []string{`"correctionMarkdown"`, `"answer"`, `"answers"`, `"correction"`} {
		if strings.Contains(string(encoded), forbiddenKey) {
			t.Errorf("public course response exposes %s", forbiddenKey)
		}
	}
	solutions := LessonSolutionsFromDomain(lesson)
	if len(solutions.Quizzes) != 1 || solutions.Quizzes[0].Questions[0].Answer.Answer == nil || *solutions.Quizzes[0].Questions[0].Answer.Answer != "ls" {
		t.Fatal("quiz answer was not mapped to the explicit solution response")
	}
	gotLesson.Exercises[0].Payload.Tasks[0] = "Changed"
	if payload["tasks"].([]string)[0] != "List files" {
		t.Fatal("exercise payload slices should be copied at the HTTP boundary")
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

func TestLessonSolutionsFromDomainUsesEmptyAnswersArray(t *testing.T) {
	t.Parallel()

	lesson := domain.Lesson{ID: uuid.New(), Quizzes: []domain.Quiz{{ID: uuid.New(), Questions: []domain.QuizQuestion{{Order: 1}}}}}
	response := LessonSolutionsFromDomain(lesson)
	answers := response.Quizzes[0].Questions[0].Answer.Answers
	if answers == nil || len(answers) != 0 {
		t.Fatalf("Answers = %#v, want non-nil empty slice", answers)
	}
}
