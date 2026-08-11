package dto

import (
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type AnalysisResponse struct {
	IsOutOfScope           bool                           `json:"isOutOfScope"`
	ErrorMessage           *string                        `json:"errorMessage"`
	WarningMessage         *string                        `json:"warningMessage"`
	SuggestedTitle         string                         `json:"suggestedTitle"`
	ShortSynopsis          string                         `json:"shortSynopsis"`
	DetectedCurrentLevel   string                         `json:"detectedCurrentLevel"`
	DetectedTargetLevel    string                         `json:"detectedTargetLevel"`
	DetectedGoal           string                         `json:"detectedGoal"`
	DetectedLanguage       string                         `json:"detectedLanguage"`
	ClarificationQuestions []clarificationQuestionPayload `json:"clarificationQuestions"`
}

type clarificationQuestionPayload struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func (r AnalysisResponse) ToDomain() (domain.AnalysisSummary, error) {
	currentLevel, err := domain.ParseLevel(r.DetectedCurrentLevel)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}
	targetLevel, err := domain.ParseLevel(r.DetectedTargetLevel)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}
	language, err := domain.ParseCourseLanguage(r.DetectedLanguage)
	if err != nil {
		return domain.AnalysisSummary{}, err
	}

	questions := make([]domain.ClarificationQuestion, 0, len(r.ClarificationQuestions))
	for _, question := range r.ClarificationQuestions {
		questions = append(questions, domain.ClarificationQuestion{
			ID:       question.ID,
			Question: question.Question,
			Options:  question.Options,
		})
	}

	goal := strings.TrimSpace(r.DetectedGoal)
	if goal == "" {
		goal = "unknown"
	}

	return domain.AnalysisSummary{
		IsOutOfScope:           r.IsOutOfScope,
		ErrorMessage:           cleanStringPtr(r.ErrorMessage),
		WarningMessage:         cleanStringPtr(r.WarningMessage),
		SuggestedTitle:         stringPtr(r.SuggestedTitle),
		ShortSynopsis:          stringPtr(r.ShortSynopsis),
		DetectedCurrentLevel:   &currentLevel,
		DetectedTargetLevel:    &targetLevel,
		DetectedGoal:           &goal,
		DetectedLanguage:       &language,
		ClarificationQuestions: questions,
	}, nil
}
