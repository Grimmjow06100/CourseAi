package service

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	"github.com/google/uuid"
)

func normalizeStructureParams(params contract.GenerateStructureParams) (contract.GenerateStructureParams, error) {
	if params.RequestID == uuid.Nil {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: generation request id", domain.ErrBlankField)
	}

	params.Title = strings.TrimSpace(params.Title)
	if params.Title == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: title", domain.ErrBlankField)
	}
	params.Synopsis = strings.TrimSpace(params.Synopsis)
	if params.Synopsis == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: synopsis", domain.ErrBlankField)
	}
	if params.CurrentLevel == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: current level", domain.ErrGenerationBriefIncomplete)
	}
	if err := params.CurrentLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if params.CurrentLevel == domain.LevelUnknown {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: current level", domain.ErrGenerationBriefIncomplete)
	}
	if params.TargetLevel == "" {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: target level", domain.ErrGenerationBriefIncomplete)
	}
	if err := params.TargetLevel.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}
	if params.TargetLevel == domain.LevelUnknown {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: target level", domain.ErrGenerationBriefIncomplete)
	}
	if params.Language == "" {
		params.Language = domain.CourseLanguageFR
	}
	if err := params.Language.Validate(); err != nil {
		return contract.GenerateStructureParams{}, err
	}

	params.Goals = textutil.TrimNonBlank(params.Goals)
	if len(params.Goals) == 0 {
		return contract.GenerateStructureParams{}, fmt.Errorf("%w: goals", domain.ErrInvalidCollection)
	}

	return params, nil
}

func requestHasAnalysis(request domain.GenerationRequest) bool {
	return request.AnalysisCompletedAt != nil
}

func generationBriefFromStructureParams(params contract.GenerateStructureParams) domain.GenerationBrief {
	return domain.GenerationBrief{
		Title:        params.Title,
		Synopsis:     params.Synopsis,
		CurrentLevel: params.CurrentLevel,
		TargetLevel:  params.TargetLevel,
		Goals:        params.Goals,
		Language:     params.Language,
	}
}

func generationBriefEqual(left, right domain.GenerationBrief) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func clarificationSubmissionMatches(request domain.GenerationRequest, params contract.SubmitClarificationsParams) bool {
	if request.ConfirmedBrief == nil || request.ClarificationsSubmittedAt == nil {
		return false
	}
	clone := request
	clone.PipelineStatus = domain.PipelineStatusAwaitingClarification
	clone.ConfirmedBrief = nil
	clone.BriefConfirmedAt = nil
	clone.ClarificationsSubmittedAt = nil
	clone.ClarificationAnswers = nil
	clone.ClarificationVersion = 0
	if err := clone.SubmitClarifications(params.Answers, params.Title, params.Synopsis, params.Language, clone.UpdatedAt); err != nil {
		return false
	}
	return generationBriefEqual(*request.ConfirmedBrief, *clone.ConfirmedBrief) &&
		clarificationAnswersEqual(request.ClarificationAnswers, clone.ClarificationAnswers)
}

func clarificationAnswersEqual(left, right []domain.ClarificationAnswer) bool {
	canonical := func(answers []domain.ClarificationAnswer) map[string]string {
		values := make(map[string]string, len(answers))
		for _, answer := range answers {
			selected := append([]string(nil), answer.SelectedValues...)
			sort.Strings(selected)
			values[strings.TrimSpace(answer.QuestionID)] = strings.Join(selected, "\x00")
		}
		return values
	}
	return reflect.DeepEqual(canonical(left), canonical(right))
}

func structureParamsFromBrief(requestID uuid.UUID, brief domain.GenerationBrief) contract.GenerateStructureParams {
	return contract.GenerateStructureParams{
		RequestID:    requestID,
		Title:        brief.Title,
		Synopsis:     brief.Synopsis,
		CurrentLevel: brief.CurrentLevel,
		TargetLevel:  brief.TargetLevel,
		Goals:        brief.Goals,
		Language:     brief.Language,
	}
}
