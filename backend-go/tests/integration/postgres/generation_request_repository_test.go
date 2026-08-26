//go:build integration

package postgresintegration_test

import (
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/postgres"
	"github.com/Grimmjow06100/course-ai/backend-go/tests/testkit"
)

func TestGenerationRequestClarificationRoundTrip(t *testing.T) {
	ctx, pool := testkit.OpenPostgres(t, 30*time.Second)
	tx := testkit.BeginRollback(t, ctx, pool)
	repository := postgres.NewGenerationRequestRepository(tx)
	now := time.Now().UTC().Truncate(time.Millisecond)

	request, err := domain.NewGenerationRequestAt("Build a Linux course", "user_test", now)
	if err != nil {
		t.Fatalf("new generation request: %v", err)
	}
	if err := request.MarkRunning("analysis", now); err != nil {
		t.Fatalf("mark analysis running: %v", err)
	}
	unknown := domain.LevelUnknown
	target := domain.LevelAdvanced
	language := domain.CourseLanguageEN
	title := "Linux administration"
	synopsis := "Learn to administer Linux in production"
	goal := "Administer Linux systems"
	if err := request.ApplyAnalysis(domain.AnalysisSummary{
		SuggestedTitle:       &title,
		ShortSynopsis:        &synopsis,
		DetectedCurrentLevel: &unknown,
		DetectedTargetLevel:  &target,
		DetectedGoal:         &goal,
		DetectedLanguage:     &language,
		ClarificationQuestions: []domain.ClarificationQuestion{{
			ID:       domain.ClarificationIDCurrentLevel,
			Question: "What is your current Linux level?",
			Options: []domain.ClarificationOption{
				{Value: "beginner", Label: "Beginner"},
				{Value: "intermediate", Label: "Intermediate"},
			},
		}},
	}, now); err != nil {
		t.Fatalf("apply analysis: %v", err)
	}
	if err := request.MarkAwaitingClarification(now); err != nil {
		t.Fatalf("mark awaiting clarification: %v", err)
	}
	if _, err := repository.SaveGenerationRequest(ctx, request); err != nil {
		t.Fatalf("save awaiting request: %v", err)
	}

	locked, err := repository.FindGenerationRequestForUpdate(ctx, request.ID)
	if err != nil {
		t.Fatalf("lock awaiting request: %v", err)
	}
	if locked.PipelineStatus != domain.PipelineStatusAwaitingClarification || len(locked.ClarificationQuestions) != 1 {
		t.Fatalf("unexpected awaiting request: %+v", locked)
	}
	if err := locked.SubmitClarifications([]domain.ClarificationAnswer{{
		QuestionID:     domain.ClarificationIDCurrentLevel,
		SelectedValues: []string{"beginner"},
	}}, title, synopsis, language, now.Add(time.Second)); err != nil {
		t.Fatalf("submit clarifications: %v", err)
	}
	if _, err := repository.UpdateGenerationRequest(ctx, locked); err != nil {
		t.Fatalf("persist clarifications: %v", err)
	}

	persisted, err := repository.FindGenerationRequestByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("reload clarified request: %v", err)
	}
	if persisted.PipelineStatus != domain.PipelineStatusQueued || persisted.ConfirmedBrief == nil {
		t.Fatalf("clarified request was not resumed: %+v", persisted)
	}
	if persisted.ConfirmedBrief.CurrentLevel != domain.LevelBeginner || persisted.ConfirmedBrief.TargetLevel != domain.LevelAdvanced {
		t.Fatalf("unexpected persisted brief: %+v", persisted.ConfirmedBrief)
	}
	if len(persisted.ClarificationAnswers) != 1 || persisted.ClarificationVersion != 1 || persisted.ClarificationsSubmittedAt == nil {
		t.Fatalf("clarification metadata was not persisted: %+v", persisted)
	}

	status, err := repository.FindGenerationStatusByID(ctx, request.ID)
	if err != nil {
		t.Fatalf("read generation status: %v", err)
	}
	if status.PipelineStatus != domain.PipelineStatusQueued || len(status.ClarificationQuestions) != 1 ||
		status.SuggestedTitle == nil || *status.SuggestedTitle != title ||
		status.DetectedCurrentLevel == nil || *status.DetectedCurrentLevel != domain.LevelUnknown ||
		status.DetectedLanguage == nil || *status.DetectedLanguage != language {
		t.Fatalf("unexpected generation status: %+v", status)
	}
}
