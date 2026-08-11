//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	appdb "github.com/Grimmjow06100/course-ai/backend-go/internal/db"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

func TestBulkPersistenceAgainstPostgres(t *testing.T) {
	if os.Getenv("COURSE_AI_INTEGRATION_TEST") != "1" {
		t.Skip("set COURSE_AI_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := appdb.Open(ctx)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	now := time.Now().UTC().Truncate(time.Millisecond)
	requestID := uuid.New()
	courseID := uuid.New()
	if _, err := tx.Exec(ctx, `
		INSERT INTO generation_requests (id, initial_user_prompt, updated_at)
		VALUES ($1, 'integration test', $2)`, requestID, now); err != nil {
		t.Fatalf("insert generation request: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO courses (
			id, request_id, language, status, initial_user_prompt, title, synopsis,
			current_level, target_level, updated_at
		) VALUES ($1, $2, 'fr', 'structure_generated', 'integration test', 'Linux',
			'Linux course', 'beginner', 'intermediate', $3)`, courseID, requestID, now); err != nil {
		t.Fatalf("insert course: %v", err)
	}

	moduleID := uuid.New()
	modules := []domain.Module{
		{ID: moduleID, CourseID: courseID, Order: 1, Title: "Module 1", Description: "Description 1", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), CourseID: courseID, Order: 2, Title: "Module 2", Description: "Description 2", CreatedAt: now, UpdatedAt: now},
	}
	if _, err := postgres.NewModuleRepository(tx).SaveModules(ctx, modules); err != nil {
		t.Fatalf("copy modules: %v", err)
	}

	lessonID := uuid.New()
	lessons := []domain.Lesson{
		{ID: lessonID, ModuleID: moduleID, Order: 1, Title: "Lesson 1", Type: domain.LessonTypeMixed, EstimatedDurationMinutes: 30, LearningGoal: "Learn", CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), ModuleID: moduleID, Order: 2, Title: "Lesson 2", Type: domain.LessonTypeTheory, EstimatedDurationMinutes: 30, LearningGoal: "Understand", CreatedAt: now, UpdatedAt: now},
	}
	if _, err := postgres.NewLessonRepository(tx).SaveLessons(ctx, lessons); err != nil {
		t.Fatalf("copy lessons: %v", err)
	}

	exercises := []domain.Exercise{
		integrationExercise(uuid.New(), lessonID, now),
		integrationExercise(uuid.New(), lessonID, now),
	}
	if _, err := postgres.NewExerciseRepository(tx).SaveExercises(ctx, exercises); err != nil {
		t.Fatalf("copy exercises: %v", err)
	}

	answer := "Linux"
	quizzes := []domain.Quiz{
		integrationQuiz(uuid.New(), lessonID, answer, now),
		integrationQuiz(uuid.New(), lessonID, answer, now),
	}
	if _, err := postgres.NewQuizRepository(tx).SaveQuizzes(ctx, quizzes); err != nil {
		t.Fatalf("copy quizzes: %v", err)
	}

	counts := []struct {
		name  string
		query string
		id    uuid.UUID
	}{
		{name: "modules", query: "SELECT count(*) FROM modules WHERE course_id = $1", id: courseID},
		{name: "lessons", query: "SELECT count(*) FROM lessons WHERE module_id = $1", id: moduleID},
		{name: "lesson_exercises", query: "SELECT count(*) FROM lesson_exercises WHERE lesson_id = $1", id: lessonID},
		{name: "lesson_quizzes", query: "SELECT count(*) FROM lesson_quizzes WHERE lesson_id = $1", id: lessonID},
	}
	for _, check := range counts {
		var count int
		if err := tx.QueryRow(ctx, check.query, check.id).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", check.name, err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows in %s, got %d", check.name, count)
		}
	}

	content := "# Generated content"
	replacement := lessons[0]
	replacement.ContentMarkdown = &content
	replacement.RawContentOutput = []byte(`{"content":"generated"}`)
	replacement.Exercises = exercises
	replacement.Quizzes = quizzes
	if _, err := postgres.NewLessonRepository(tx).ReplaceLessonContent(ctx, replacement); err != nil {
		t.Fatalf("replace lesson content: %v", err)
	}
	for _, table := range []string{"lesson_exercises", "lesson_quizzes"} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE lesson_id = $1"
		if err := tx.QueryRow(ctx, query, lessonID).Scan(&count); err != nil {
			t.Fatalf("count cleared %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("expected activities in %s to be cleared, got %d", table, count)
		}
	}
}

func integrationExercise(id, lessonID uuid.UUID, now time.Time) domain.Exercise {
	return domain.Exercise{
		ID: id, LessonID: lessonID, Type: domain.ExerciseTypeCommandLine,
		Difficulty: domain.DifficultyBeginner, Title: "Exercise", Objective: "Run a command",
		InstructionsMarkdown: "Run it.", ContentMarkdown: "`uname -a`",
		CorrectionMarkdown: "The command prints system information.", Payload: domain.ExercisePayload{},
		CreatedAt: now, UpdatedAt: now,
	}
}

func integrationQuiz(id, lessonID uuid.UUID, answer string, now time.Time) domain.Quiz {
	return domain.Quiz{
		ID: id, LessonID: lessonID, Type: domain.QuizTypeShortAnswer,
		Difficulty: domain.DifficultyBeginner, Title: "Quiz", Objective: "Check understanding",
		Questions: []domain.QuizQuestion{{
			Order: 1, Type: domain.QuizQuestionTypeShortAnswer, Question: "Quel systeme ?",
			Answer: domain.QuizAnswer{Answer: &answer}, Correction: "Linux.",
		}},
		CreatedAt: now, UpdatedAt: now,
	}
}
