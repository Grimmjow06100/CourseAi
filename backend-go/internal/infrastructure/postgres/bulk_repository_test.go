package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type copyFromRecorder struct {
	calls  int
	tables []pgx.Identifier
	rows   int
}

func (r *copyFromRecorder) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec call")
}

func (r *copyFromRecorder) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query call")
}

func (r *copyFromRecorder) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow call")
}

func (r *copyFromRecorder) CopyFrom(
	_ context.Context,
	tableName pgx.Identifier,
	_ []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	r.calls++
	r.tables = append(r.tables, tableName)
	inserted := 0
	for rowSrc.Next() {
		if _, err := rowSrc.Values(); err != nil {
			return 0, err
		}
		inserted++
	}
	if err := rowSrc.Err(); err != nil {
		return 0, err
	}
	r.rows += inserted
	return int64(inserted), nil
}

func TestBulkRepositoriesUseOneCopyFromPerCollection(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 11, 10, 0, 0, 0, time.UTC)
	courseID := uuid.New()
	moduleID := uuid.New()
	lessonID := uuid.New()
	answer := "Linux"

	tests := []struct {
		name      string
		tableName string
		save      func(*copyFromRecorder) (int, error)
	}{
		{
			name:      "modules",
			tableName: "modules",
			save: func(db *copyFromRecorder) (int, error) {
				items := []domain.Module{
					validModule(uuid.New(), courseID, 1, now),
					validModule(uuid.New(), courseID, 2, now),
				}
				saved, err := NewModuleRepository(db).SaveModules(context.Background(), items)
				return len(saved), err
			},
		},
		{
			name:      "lessons",
			tableName: "lessons",
			save: func(db *copyFromRecorder) (int, error) {
				items := []domain.Lesson{
					validLesson(uuid.New(), moduleID, 1, now),
					validLesson(uuid.New(), moduleID, 2, now),
				}
				saved, err := NewLessonRepository(db).SaveLessons(context.Background(), items)
				return len(saved), err
			},
		},
		{
			name:      "exercises",
			tableName: "lesson_exercises",
			save: func(db *copyFromRecorder) (int, error) {
				items := []domain.Exercise{
					validExercise(uuid.New(), lessonID, now),
					validExercise(uuid.New(), lessonID, now),
				}
				saved, err := NewExerciseRepository(db).SaveExercises(context.Background(), items)
				return len(saved), err
			},
		},
		{
			name:      "quizzes",
			tableName: "lesson_quizzes",
			save: func(db *copyFromRecorder) (int, error) {
				items := []domain.Quiz{
					validQuiz(uuid.New(), lessonID, answer, now),
					validQuiz(uuid.New(), lessonID, answer, now),
				}
				saved, err := NewQuizRepository(db).SaveQuizzes(context.Background(), items)
				return len(saved), err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			db := &copyFromRecorder{}
			savedCount, err := test.save(db)
			if err != nil {
				t.Fatalf("save collection: %v", err)
			}
			if savedCount != 2 || db.rows != 2 {
				t.Fatalf("expected two saved rows, got saved=%d copied=%d", savedCount, db.rows)
			}
			if db.calls != 1 {
				t.Fatalf("expected one CopyFrom call, got %d", db.calls)
			}
			if len(db.tables) != 1 || len(db.tables[0]) != 1 || db.tables[0][0] != test.tableName {
				t.Fatalf("unexpected CopyFrom table: %#v", db.tables)
			}
		})
	}
}

func TestBulkRepositoriesSkipDatabaseForEmptyCollections(t *testing.T) {
	t.Parallel()

	db := &copyFromRecorder{}
	if _, err := NewModuleRepository(db).SaveModules(context.Background(), nil); err != nil {
		t.Fatalf("save empty modules: %v", err)
	}
	if _, err := NewLessonRepository(db).SaveLessons(context.Background(), nil); err != nil {
		t.Fatalf("save empty lessons: %v", err)
	}
	if _, err := NewExerciseRepository(db).SaveExercises(context.Background(), nil); err != nil {
		t.Fatalf("save empty exercises: %v", err)
	}
	if _, err := NewQuizRepository(db).SaveQuizzes(context.Background(), nil); err != nil {
		t.Fatalf("save empty quizzes: %v", err)
	}
	if db.calls != 0 {
		t.Fatalf("expected no database call, got %d", db.calls)
	}
}

func validModule(id, courseID uuid.UUID, order int, now time.Time) domain.Module {
	return domain.Module{
		ID:          id,
		CourseID:    courseID,
		Order:       order,
		Title:       "Module",
		Description: "Description",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func validLesson(id, moduleID uuid.UUID, order int, now time.Time) domain.Lesson {
	return domain.Lesson{
		ID:                       id,
		ModuleID:                 moduleID,
		Order:                    order,
		Title:                    "Lesson",
		Type:                     domain.LessonTypeTheory,
		EstimatedDurationMinutes: 30,
		LearningGoal:             "Learn Linux",
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

func validExercise(id, lessonID uuid.UUID, now time.Time) domain.Exercise {
	return domain.Exercise{
		ID:                   id,
		LessonID:             lessonID,
		Type:                 domain.ExerciseTypeCommandLine,
		Difficulty:           domain.DifficultyBeginner,
		Title:                "Exercise",
		Objective:            "Run a command",
		InstructionsMarkdown: "Run the command.",
		ContentMarkdown:      "`uname -a`",
		CorrectionMarkdown:   "The command prints system information.",
		Payload:              domain.ExercisePayload{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func validQuiz(id, lessonID uuid.UUID, answer string, now time.Time) domain.Quiz {
	return domain.Quiz{
		ID:         id,
		LessonID:   lessonID,
		Type:       domain.QuizTypeShortAnswer,
		Difficulty: domain.DifficultyBeginner,
		Title:      "Quiz",
		Objective:  "Check understanding",
		Questions: []domain.QuizQuestion{
			{
				Order:      1,
				Type:       domain.QuizQuestionTypeShortAnswer,
				Question:   "Quel systeme etudions-nous ?",
				Answer:     domain.QuizAnswer{Answer: &answer},
				Correction: "La reponse est Linux.",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
