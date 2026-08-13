package domain

import (
	"errors"
	"testing"
	"time"
)

func TestGenerationRequestLifecycle(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.August, 13, 9, 0, 0, 0, time.UTC)
	request, err := NewGenerationRequestAt("  create   a Linux course ", createdAt)
	if err != nil {
		t.Fatalf("NewGenerationRequestAt() error = %v", err)
	}
	if request.InitialUserPrompt != "create a Linux course" || request.PipelineStatus != PipelineStatusQueued {
		t.Fatalf("unexpected request: %+v", request)
	}

	runningAt := createdAt.Add(time.Minute)
	if err := request.MarkRunning(" analysis ", runningAt); err != nil {
		t.Fatalf("MarkRunning() error = %v", err)
	}
	if request.CurrentStep == nil || *request.CurrentStep != "analysis" || request.StartedAt == nil {
		t.Fatal("running state was not persisted")
	}

	currentLevel := LevelBeginner
	targetLevel := LevelAdvanced
	language := CourseLanguageFR
	title := "  Linux essentials "
	questions := []ClarificationQuestion{{ID: " goals ", Question: " Main goal? ", Options: []string{" Admin ", "", "DevOps"}}}
	analysisAt := runningAt.Add(time.Minute)
	if err := request.ApplyAnalysis(AnalysisSummary{
		SuggestedTitle:         &title,
		DetectedCurrentLevel:   &currentLevel,
		DetectedTargetLevel:    &targetLevel,
		DetectedLanguage:       &language,
		ClarificationQuestions: questions,
	}, analysisAt); err != nil {
		t.Fatalf("ApplyAnalysis() error = %v", err)
	}
	questions[0].Options[0] = "changed"
	if request.SuggestedTitle == nil || *request.SuggestedTitle != "Linux essentials" {
		t.Fatalf("suggested title was not normalized: %v", request.SuggestedTitle)
	}
	if got := request.ClarificationQuestions[0].Options[0]; got != "Admin" {
		t.Fatalf("questions were not defensively copied: %q", got)
	}

	progressAt := analysisAt.Add(time.Minute)
	if err := request.UpdateProgress(" structure ", 75, progressAt); err != nil {
		t.Fatalf("UpdateProgress() error = %v", err)
	}
	completedAt := progressAt.Add(time.Minute)
	if err := request.MarkCompleted(completedAt); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}
	if request.ProgressPercent != 100 || request.CompletedAt == nil || !request.CompletedAt.Equal(completedAt) {
		t.Fatalf("unexpected completed state: %+v", request)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("completed request is invalid: %v", err)
	}
}

func TestGenerationRequestFailureCanBeRestarted(t *testing.T) {
	t.Parallel()

	request, err := NewGenerationRequestAt("Linux", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	failedAt := time.Unix(60, 0)
	if err := request.MarkFailed(" provider timeout ", failedAt); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if request.FailureMessage == nil || *request.FailureMessage != "provider timeout" {
		t.Fatalf("unexpected failure message: %v", request.FailureMessage)
	}

	restartedAt := time.Unix(120, 0)
	if err := request.RestartFromFailure(" architecture ", 25, restartedAt); err != nil {
		t.Fatalf("RestartFromFailure() error = %v", err)
	}
	if request.PipelineStatus != PipelineStatusRunning || request.FailureMessage != nil || request.CompletedAt != nil {
		t.Fatalf("unexpected restarted state: %+v", request)
	}
}

func TestGenerationRequestRejectsInvalidTransitionsAndData(t *testing.T) {
	t.Parallel()

	request, err := NewGenerationRequestAt("Linux", time.Unix(0, 0))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := request.UpdateProgress("analysis", 10, time.Now()); !errors.Is(err, ErrGenerationRequestNotReady) {
		t.Fatalf("queued progress error = %v", err)
	}
	if err := request.MarkRunning(" ", time.Now()); !errors.Is(err, ErrBlankField) {
		t.Fatalf("blank step error = %v", err)
	}
	if err := request.MarkRunning("analysis", time.Now()); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := request.UpdateProgress("analysis", 101, time.Now()); !errors.Is(err, ErrInvalidProgress) {
		t.Fatalf("invalid progress error = %v", err)
	}
	if err := request.NeedsClarification(nil, time.Now()); !errors.Is(err, ErrInvalidClarification) {
		t.Fatalf("empty clarification error = %v", err)
	}
	if err := request.MarkFailed(" ", time.Now()); !errors.Is(err, ErrBlankField) {
		t.Fatalf("blank failure error = %v", err)
	}
	if err := request.RestartFromFailure("analysis", 0, time.Now()); !errors.Is(err, ErrGenerationRequestNotReady) {
		t.Fatalf("restart running request error = %v", err)
	}
}

func TestClarificationQuestionValidate(t *testing.T) {
	t.Parallel()

	valid := ClarificationQuestion{ID: "goals", Question: "What is your goal?", Options: []string{"Learn"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid question rejected: %v", err)
	}
	valid.Options = []string{" ", ""}
	if err := valid.Validate(); !errors.Is(err, ErrInvalidClarification) {
		t.Fatalf("blank options error = %v", err)
	}
}
