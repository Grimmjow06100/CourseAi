package domain

import (
	"errors"
	"testing"
	"time"
)

func TestGenerationRequestLifecycleWithoutClarification(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.August, 13, 9, 0, 0, 0, time.UTC)
	request, err := NewGenerationRequestAt("  create   a Linux course ", createdAt)
	if err != nil {
		t.Fatalf("NewGenerationRequestAt() error = %v", err)
	}
	runningAt := createdAt.Add(time.Minute)
	if err := request.MarkRunning(" analysis ", runningAt); err != nil {
		t.Fatalf("MarkRunning() error = %v", err)
	}

	currentLevel := LevelBeginner
	targetLevel := LevelAdvanced
	language := CourseLanguageFR
	title := " Linux essentials "
	synopsis := " Learn Linux "
	goal := "Administer Linux"
	analysisAt := runningAt.Add(time.Minute)
	if err := request.ApplyAnalysis(AnalysisSummary{
		SuggestedTitle:       &title,
		ShortSynopsis:        &synopsis,
		DetectedCurrentLevel: &currentLevel,
		DetectedTargetLevel:  &targetLevel,
		DetectedGoal:         &goal,
		DetectedLanguage:     &language,
	}, analysisAt); err != nil {
		t.Fatalf("ApplyAnalysis() error = %v", err)
	}
	if err := request.ConfirmDetectedBrief(analysisAt); err != nil {
		t.Fatalf("ConfirmDetectedBrief() error = %v", err)
	}
	if request.ConfirmedBrief == nil || request.ConfirmedBrief.Title != "Linux essentials" {
		t.Fatalf("confirmed brief was not normalized: %+v", request.ConfirmedBrief)
	}

	completedAt := analysisAt.Add(time.Minute)
	if err := request.MarkCompleted(completedAt); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("completed request is invalid: %v", err)
	}
}

func TestGenerationRequestClarificationPauseAndResume(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)
	request, err := NewGenerationRequestAt("Linux", now)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := request.MarkRunning("analysis", now); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	unknown := LevelUnknown
	advanced := LevelAdvanced
	language := CourseLanguageFR
	title := "Linux"
	synopsis := "Linux course"
	goal := "Administer Linux"
	questions := []ClarificationQuestion{{
		ID: ClarificationIDCurrentLevel, Question: "Current level?", AllowMultiple: false,
		Options: []ClarificationOption{{Value: "beginner", Label: "Beginner"}, {Value: "intermediate", Label: "Intermediate"}},
	}}
	if err := request.ApplyAnalysis(AnalysisSummary{
		SuggestedTitle: &title, ShortSynopsis: &synopsis,
		DetectedCurrentLevel: &unknown, DetectedTargetLevel: &advanced,
		DetectedGoal: &goal, DetectedLanguage: &language, ClarificationQuestions: questions,
	}, now); err != nil {
		t.Fatalf("apply analysis: %v", err)
	}
	questions[0].Options[0].Label = "changed"
	if request.ClarificationQuestions[0].Options[0].Label != "Beginner" {
		t.Fatal("clarification questions were not defensively copied")
	}
	if err := request.MarkAwaitingClarification(now); err != nil {
		t.Fatalf("mark awaiting: %v", err)
	}
	if request.PipelineStatus != PipelineStatusAwaitingClarification || request.CompletedAt != nil {
		t.Fatalf("unexpected awaiting state: %+v", request)
	}
	if err := request.SubmitClarifications(
		[]ClarificationAnswer{{QuestionID: ClarificationIDCurrentLevel, SelectedValues: []string{"beginner"}}},
		"Linux", "Linux course", CourseLanguageFR, now.Add(time.Minute),
	); err != nil {
		t.Fatalf("submit clarifications: %v", err)
	}
	if request.PipelineStatus != PipelineStatusQueued || request.ConfirmedBrief == nil || request.ConfirmedBrief.CurrentLevel != LevelBeginner {
		t.Fatalf("unexpected resumed state: %+v", request)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("resumed request is invalid: %v", err)
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
	if err := request.MarkAwaitingClarification(time.Now()); !errors.Is(err, ErrGenerationRequestNotReady) {
		t.Fatalf("empty clarification error = %v", err)
	}
}

func TestClarificationQuestionAndAnswerValidation(t *testing.T) {
	t.Parallel()

	question := ClarificationQuestion{
		ID: ClarificationIDGoals, Question: "What is your goal?", AllowMultiple: true,
		Options: []ClarificationOption{{Value: "Admin", Label: "Administer"}, {Value: "DevOps", Label: "DevOps"}},
	}
	if err := question.Validate(); err != nil {
		t.Fatalf("valid question rejected: %v", err)
	}
	if err := ValidateClarificationAnswers(
		[]ClarificationQuestion{question},
		[]ClarificationAnswer{{QuestionID: ClarificationIDGoals, SelectedValues: []string{"Admin", "DevOps"}}},
	); err != nil {
		t.Fatalf("valid answers rejected: %v", err)
	}
	if err := ValidateClarificationAnswers(
		[]ClarificationQuestion{question},
		[]ClarificationAnswer{{QuestionID: ClarificationIDGoals, SelectedValues: []string{"Unknown"}}},
	); !errors.Is(err, ErrClarificationValueNotAllowed) {
		t.Fatalf("unexpected invalid answer error: %v", err)
	}
}
