package dto

import (
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

func TestAnalysisResponseToDomain(t *testing.T) {
	t.Parallel()

	warning := "  Be precise  "
	response := AnalysisResponse{
		SuggestedTitle: " Linux ", ShortSynopsis: " Basics ", WarningMessage: &warning,
		DetectedCurrentLevel: "beginner", DetectedTargetLevel: "advanced",
		DetectedGoal: " ", DetectedLanguage: "fr",
		ClarificationQuestions: []clarificationQuestionPayload{{ID: "goals", Question: "Goal?", Options: []string{"Admin"}}},
	}
	summary, err := response.ToDomain()
	if err != nil {
		t.Fatalf("ToDomain() error = %v", err)
	}
	if summary.SuggestedTitle == nil || *summary.SuggestedTitle != "Linux" || summary.WarningMessage == nil || *summary.WarningMessage != "Be precise" {
		t.Fatalf("text fields were not normalized: %+v", summary)
	}
	if summary.DetectedGoal == nil || *summary.DetectedGoal != "unknown" || summary.DetectedLanguage == nil || *summary.DetectedLanguage != domain.CourseLanguageFR {
		t.Fatalf("unexpected detected values: %+v", summary)
	}
	if len(summary.ClarificationQuestions) != 1 {
		t.Fatalf("questions = %d, want 1", len(summary.ClarificationQuestions))
	}
}

func TestAnalysisResponseToDomainRejectsInvalidEnums(t *testing.T) {
	t.Parallel()

	response := AnalysisResponse{DetectedCurrentLevel: "novice", DetectedTargetLevel: "advanced", DetectedLanguage: "fr"}
	if _, err := response.ToDomain(); !errors.Is(err, domain.ErrInvalidLevel) {
		t.Fatalf("ToDomain() error = %v, want ErrInvalidLevel", err)
	}
}
