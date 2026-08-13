package domain

import "testing"

func TestStatusParsersNormalizeAndValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parse   func(string) error
		valid   string
		invalid string
	}{
		{name: "course language", valid: " FR ", invalid: "de", parse: func(value string) error { _, err := ParseCourseLanguage(value); return err }},
		{name: "level", valid: " ADVANCED ", invalid: "master", parse: func(value string) error { _, err := ParseLevel(value); return err }},
		{name: "lesson type", valid: " MIXED ", invalid: "video", parse: func(value string) error { _, err := ParseLessonType(value); return err }},
		{name: "difficulty", valid: " BEGINNER ", invalid: "easy", parse: func(value string) error { _, err := ParseDifficulty(value); return err }},
		{name: "exercise type", valid: " CODING ", invalid: "essay", parse: func(value string) error { _, err := ParseExerciseType(value); return err }},
		{name: "quiz type", valid: " TRUE_FALSE ", invalid: "poll", parse: func(value string) error { _, err := ParseQuizType(value); return err }},
		{name: "quiz question type", valid: " SHORT_ANSWER ", invalid: "free_text", parse: func(value string) error { _, err := ParseQuizQuestionType(value); return err }},
		{name: "course status", valid: " CONTENT_GENERATING ", invalid: "published", parse: func(value string) error { _, err := ParseCourseGenerationStatus(value); return err }},
		{name: "pipeline status", valid: " RUNNING ", invalid: "paused", parse: func(value string) error { _, err := ParseGenerationPipelineStatus(value); return err }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.parse(test.valid); err != nil {
				t.Fatalf("parse valid value: %v", err)
			}
			if err := test.parse(test.invalid); err == nil {
				t.Fatal("expected invalid value to be rejected")
			}
		})
	}
}

func TestCourseGenerationStatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from CourseGenerationStatus
		to   CourseGenerationStatus
		want bool
	}{
		{name: "analysis can request clarification", from: CourseStatusAnalysisPending, to: CourseStatusNeedsClarification, want: true},
		{name: "clarification can restart analysis", from: CourseStatusNeedsClarification, to: CourseStatusAnalysisPending, want: true},
		{name: "content can complete", from: CourseStatusContentGenerating, to: CourseStatusCompleted, want: true},
		{name: "same status is idempotent", from: CourseStatusStructureGenerated, to: CourseStatusStructureGenerated, want: true},
		{name: "cannot skip stages", from: CourseStatusAnalysisCompleted, to: CourseStatusCompleted, want: false},
		{name: "terminal cannot restart", from: CourseStatusFailed, to: CourseStatusAnalysisPending, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.from.CanTransitionTo(test.to); got != test.want {
				t.Fatalf("CanTransitionTo(%q) = %t, want %t", test.to, got, test.want)
			}
		})
	}
}

func TestGenerationPipelineStatusTransitions(t *testing.T) {
	t.Parallel()

	if !PipelineStatusQueued.CanTransitionTo(PipelineStatusRunning) {
		t.Fatal("queued should transition to running")
	}
	if !PipelineStatusRunning.CanTransitionTo(PipelineStatusCompleted) {
		t.Fatal("running should transition to completed")
	}
	if PipelineStatusCompleted.CanTransitionTo(PipelineStatusRunning) {
		t.Fatal("completed must be terminal")
	}
	if PipelineStatusQueued.IsTerminal() || !PipelineStatusFailed.IsTerminal() {
		t.Fatal("terminal status classification is incorrect")
	}
}
