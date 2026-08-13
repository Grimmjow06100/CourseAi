package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCourseAggregateLifecycleAndDuration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)
	course := newValidCourseForTest(t, now)
	module, err := NewModuleAt(NewModuleParams{
		CourseID: course.ID, Order: 1, Title: "Foundations", Description: "Linux foundations",
	}, now)
	if err != nil {
		t.Fatalf("NewModuleAt() error = %v", err)
	}
	lesson, err := NewLessonAt(NewLessonParams{
		ModuleID: module.ID, Order: 1, Title: "Filesystem", Type: LessonTypeTheory,
		EstimatedDurationMinutes: 35, LearningGoal: "Navigate the filesystem",
	}, now)
	if err != nil {
		t.Fatalf("NewLessonAt() error = %v", err)
	}
	if err := lesson.AttachContent("# Filesystem"); err != nil {
		t.Fatalf("AttachContent() error = %v", err)
	}
	if err := module.AddLesson(lesson); err != nil {
		t.Fatalf("AddLesson() error = %v", err)
	}
	if err := course.AddModule(module); err != nil {
		t.Fatalf("AddModule() error = %v", err)
	}

	transitions := []func() error{
		course.MarkArchitectureGenerating,
		course.MarkArchitectureGenerated,
		course.MarkLessonsGenerating,
		course.MarkLessonsGenerated,
		course.MarkContentGenerating,
		course.MarkCompleted,
	}
	for index, transition := range transitions {
		if err := transition(); err != nil {
			t.Fatalf("transition %d error = %v", index, err)
		}
	}
	if course.TotalDurationMinutes() != 35 || !course.HasCompleteContent() {
		t.Fatalf("unexpected aggregate metrics: duration=%d complete=%t", course.TotalDurationMinutes(), course.HasCompleteContent())
	}
	if err := course.ValidateWithRelations(); err != nil {
		t.Fatalf("completed aggregate is invalid: %v", err)
	}
}

func TestCourseAndModuleRejectInvalidRelations(t *testing.T) {
	t.Parallel()

	course := newValidCourseForTest(t, time.Unix(0, 0))
	module, err := NewModuleAt(NewModuleParams{CourseID: course.ID, Order: 1, Title: "Module", Description: "Description"}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new module: %v", err)
	}
	if err := course.AddModule(module); err != nil {
		t.Fatalf("add module: %v", err)
	}
	if err := course.AddModule(module); !errors.Is(err, ErrDuplicateModuleOrder) {
		t.Fatalf("duplicate module error = %v", err)
	}
	foreignModule := module
	foreignModule.Order = 2
	foreignModule.CourseID = uuid.New()
	if err := course.AddModule(foreignModule); !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("foreign module error = %v", err)
	}

	lesson, err := NewLessonAt(NewLessonParams{
		ModuleID: module.ID, Order: 1, Title: "Lesson", Type: LessonTypeTheory,
		EstimatedDurationMinutes: 10, LearningGoal: "Learn",
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new lesson: %v", err)
	}
	if err := module.AddLesson(lesson); err != nil {
		t.Fatalf("add lesson: %v", err)
	}
	if err := module.AddLesson(lesson); !errors.Is(err, ErrDuplicateLessonOrder) {
		t.Fatalf("duplicate lesson error = %v", err)
	}
	foreignLesson := lesson
	foreignLesson.Order = 2
	foreignLesson.ModuleID = uuid.New()
	if err := module.AddLesson(foreignLesson); !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("foreign lesson error = %v", err)
	}
}

func TestCourseCannotCompleteWithoutGeneratedContent(t *testing.T) {
	t.Parallel()

	course := newValidCourseForTest(t, time.Unix(0, 0))
	if err := course.MarkCompleted(); !errors.Is(err, ErrMissingCourseContent) {
		t.Fatalf("MarkCompleted() error = %v, want ErrMissingCourseContent", err)
	}
	if err := course.TransitionTo(CourseStatusCompleted); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("skipped transition error = %v", err)
	}
	if err := course.MarkFailed(); err != nil || course.Status != CourseStatusFailed {
		t.Fatalf("MarkFailed() = %s, %v", course.Status, err)
	}
}

func TestLessonContentAndRelationValidation(t *testing.T) {
	t.Parallel()

	lesson, err := NewLessonAt(NewLessonParams{
		ModuleID: uuid.New(), Order: 1, Title: "Commands", Type: LessonTypePractice,
		EstimatedDurationMinutes: 20, LearningGoal: "Use commands",
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new lesson: %v", err)
	}
	if err := lesson.AttachContent("  "); !errors.Is(err, ErrBlankField) {
		t.Fatalf("blank content error = %v", err)
	}
	exercise, err := NewExerciseAt(NewExerciseParams{
		LessonID: lesson.ID, Type: ExerciseTypeCommandLine, Difficulty: DifficultyBeginner,
		Title: "List files", Objective: "Use ls", InstructionsMarkdown: "Run it",
		ContentMarkdown: "`ls`", CorrectionMarkdown: "Expected output",
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new exercise: %v", err)
	}
	foreign := exercise
	foreign.LessonID = uuid.New()
	if err := lesson.AddExercise(foreign); !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("foreign exercise error = %v", err)
	}
	if err := lesson.AddExercise(exercise); err != nil {
		t.Fatalf("add exercise: %v", err)
	}
	if !lesson.HasExercise() || !lesson.HasStructuredActivities() || !lesson.HasContent() {
		t.Fatal("exercise should count as structured lesson content")
	}
}

func newValidCourseForTest(t *testing.T, now time.Time) Course {
	t.Helper()
	course, err := NewCourseAt(NewCourseParams{
		RequestID:         uuid.New(),
		Language:          CourseLanguageEN,
		InitialUserPrompt: "Learn Linux",
		Title:             "Linux",
		Synopsis:          "A Linux course",
		CurrentLevel:      LevelBeginner,
		TargetLevel:       LevelAdvanced,
	}, now)
	if err != nil {
		t.Fatalf("NewCourseAt() error = %v", err)
	}
	return course
}
