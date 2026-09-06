package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ClarificationIDGoals        = "goals"
	ClarificationIDCurrentLevel = "currentLevel"
	ClarificationIDTargetLevel  = "targetLevel"
)

type ClarificationOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ClarificationQuestion struct {
	ID            string                `json:"id"`
	Question      string                `json:"question"`
	Options       []ClarificationOption `json:"options"`
	AllowMultiple bool                  `json:"allowMultiple"`
}

type ClarificationAnswer struct {
	QuestionID     string   `json:"questionId"`
	SelectedValues []string `json:"selectedValues"`
}

type GenerationBrief struct {
	Title        string
	Synopsis     string
	CurrentLevel Level
	TargetLevel  Level
	Goals        []string
	Language     CourseLanguage
}

type AnalysisSummary struct {
	IsOutOfScope           bool
	ErrorMessage           *string
	WarningMessage         *string
	SuggestedTitle         *string
	ShortSynopsis          *string
	DetectedCurrentLevel   *Level
	DetectedTargetLevel    *Level
	DetectedGoal           *string
	DetectedLanguage       *CourseLanguage
	ClarificationQuestions []ClarificationQuestion
}

type GenerationRequest struct {
	ID                        uuid.UUID
	ClerkUserID               string
	InitialUserPrompt         string
	PipelineStatus            GenerationPipelineStatus
	CurrentStep               *string
	ProgressPercent           int
	FailureMessage            *string
	StartedAt                 *time.Time
	CompletedAt               *time.Time
	IsOutOfScope              bool
	ErrorMessage              *string
	WarningMessage            *string
	SuggestedTitle            *string
	ShortSynopsis             *string
	DetectedCurrentLevel      *Level
	DetectedTargetLevel       *Level
	DetectedGoal              *string
	DetectedLanguage          *CourseLanguage
	ClarificationQuestions    []ClarificationQuestion
	ClarificationAnswers      []ClarificationAnswer
	ConfirmedBrief            *GenerationBrief
	AnalysisCompletedAt       *time.Time
	BriefConfirmedAt          *time.Time
	ClarificationsSubmittedAt *time.Time
	ClarificationVersion      int
	RawAnalysisOutput         json.RawMessage
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

func NewGenerationRequest(prompt string, clerkUserID string) (GenerationRequest, error) {
	return NewGenerationRequestAt(prompt, clerkUserID, time.Now())
}

func NewGenerationRequestAt(prompt string, clerkUserID string, now time.Time) (GenerationRequest, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return GenerationRequest{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	request := GenerationRequest{
		ID:                id,
		ClerkUserID:       normalizeText(clerkUserID),
		InitialUserPrompt: normalizeText(prompt),
		PipelineStatus:    PipelineStatusQueued,
		ProgressPercent:   0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := request.Validate(); err != nil {
		return GenerationRequest{}, err
	}
	return request, nil
}

func (r GenerationRequest) Validate() error {
	if err := r.validateIdentity(); err != nil {
		return err
	}
	if err := r.validateLifecycle(); err != nil {
		return err
	}
	return r.validateClarificationState()
}

func (r GenerationRequest) validateIdentity() error {
	if r.ID == uuid.Nil {
		return fmt.Errorf("%w: generation request id", ErrBlankField)
	}
	if err := validateClerkUserID(r.ClerkUserID); err != nil {
		return fmt.Errorf("%w: generation request clerk user id", err)
	}
	if err := requireNotBlank("initial user prompt", r.InitialUserPrompt); err != nil {
		return err
	}
	if err := r.PipelineStatus.Validate(); err != nil {
		return err
	}
	return validateProgressPercent(r.ProgressPercent)
}

func (r GenerationRequest) validateLifecycle() error {
	if r.PipelineStatus == PipelineStatusFailed && r.FailureMessage == nil {
		return fmt.Errorf("%w: failure message", ErrBlankField)
	}
	if r.PipelineStatus.IsTerminal() && r.CompletedAt == nil {
		return fmt.Errorf("%w: completed at", ErrBlankField)
	}
	if !r.PipelineStatus.IsTerminal() && r.CompletedAt != nil {
		return ErrGenerationRequestNotReady
	}
	return nil
}

func (r GenerationRequest) validateClarificationState() error {
	if err := validateClarificationQuestions(r.ClarificationQuestions); err != nil {
		return err
	}
	if err := r.validateAnalysisState(); err != nil {
		return err
	}
	if r.ConfirmedBrief == nil {
		return r.validateUnconfirmedBriefState()
	}
	return r.validateConfirmedBriefState()
}

func (r GenerationRequest) validateAnalysisState() error {
	if r.AnalysisCompletedAt == nil && r.hasPostAnalysisState() {
		return fmt.Errorf("%w: analysis completed at", ErrBlankField)
	}
	if r.PipelineStatus == PipelineStatusAwaitingClarification {
		if len(r.ClarificationQuestions) == 0 || r.IsOutOfScope || r.ConfirmedBrief != nil {
			return ErrGenerationRequestNotReady
		}
	}
	return nil
}

func (r GenerationRequest) hasPostAnalysisState() bool {
	return len(r.ClarificationQuestions) > 0 ||
		r.ConfirmedBrief != nil ||
		r.PipelineStatus == PipelineStatusAwaitingClarification
}

func (r GenerationRequest) validateUnconfirmedBriefState() error {
	if r.BriefConfirmedAt != nil || r.ClarificationsSubmittedAt != nil || r.ClarificationVersion != 0 || len(r.ClarificationAnswers) > 0 {
		return ErrGenerationBriefIncomplete
	}
	return nil
}

func (r GenerationRequest) validateConfirmedBriefState() error {
	if err := r.ConfirmedBrief.Validate(); err != nil {
		return err
	}
	if r.BriefConfirmedAt == nil || r.ClarificationVersion <= 0 {
		return ErrGenerationBriefIncomplete
	}
	if len(r.ClarificationAnswers) > 0 {
		if err := ValidateClarificationAnswers(r.ClarificationQuestions, r.ClarificationAnswers); err != nil {
			return err
		}
	}
	if r.ClarificationsSubmittedAt != nil && len(r.ClarificationAnswers) == 0 {
		return ErrInvalidClarificationAnswer
	}
	return nil
}

func (b GenerationBrief) Validate() error {
	if err := requireNotBlank("confirmed title", b.Title); err != nil {
		return fmt.Errorf("%w: %v", ErrGenerationBriefIncomplete, err)
	}
	if err := requireNotBlank("confirmed synopsis", b.Synopsis); err != nil {
		return fmt.Errorf("%w: %v", ErrGenerationBriefIncomplete, err)
	}
	if err := b.CurrentLevel.Validate(); err != nil || b.CurrentLevel == LevelUnknown {
		return fmt.Errorf("%w: current level", ErrGenerationBriefIncomplete)
	}
	if err := b.TargetLevel.Validate(); err != nil || b.TargetLevel == LevelUnknown {
		return fmt.Errorf("%w: target level", ErrGenerationBriefIncomplete)
	}
	if len(normalizeStringSlice(b.Goals)) == 0 {
		return fmt.Errorf("%w: goals", ErrGenerationBriefIncomplete)
	}
	if err := b.Language.Validate(); err != nil {
		return fmt.Errorf("%w: language", ErrGenerationBriefIncomplete)
	}
	return nil
}

func (r *GenerationRequest) MarkRunning(step string, now time.Time) error {
	if !r.PipelineStatus.CanTransitionTo(PipelineStatusRunning) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, r.PipelineStatus, PipelineStatusRunning)
	}
	step = normalizeText(step)
	if step == "" {
		return fmt.Errorf("%w: current step", ErrBlankField)
	}

	r.PipelineStatus = PipelineStatusRunning
	r.CurrentStep = &step
	if r.StartedAt == nil {
		r.StartedAt = &now
	}
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) UpdateProgress(step string, percent int, now time.Time) error {
	if r.PipelineStatus != PipelineStatusRunning {
		return ErrGenerationRequestNotReady
	}
	if err := validateProgressPercent(percent); err != nil {
		return err
	}
	step = normalizeText(step)
	if step == "" {
		return fmt.Errorf("%w: current step", ErrBlankField)
	}

	r.CurrentStep = &step
	r.ProgressPercent = percent
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) ApplyAnalysis(summary AnalysisSummary, now time.Time) error {
	if summary.DetectedCurrentLevel != nil {
		if err := summary.DetectedCurrentLevel.Validate(); err != nil {
			return err
		}
	}
	if summary.DetectedTargetLevel != nil {
		if err := summary.DetectedTargetLevel.Validate(); err != nil {
			return err
		}
	}
	if summary.DetectedLanguage != nil {
		if err := summary.DetectedLanguage.Validate(); err != nil {
			return err
		}
	}
	if err := validateAnalysisClarifications(summary); err != nil {
		return err
	}

	r.IsOutOfScope = summary.IsOutOfScope
	r.ErrorMessage = trimOptionalString(summary.ErrorMessage)
	r.WarningMessage = trimOptionalString(summary.WarningMessage)
	r.SuggestedTitle = trimOptionalString(summary.SuggestedTitle)
	r.ShortSynopsis = trimOptionalString(summary.ShortSynopsis)
	r.DetectedCurrentLevel = summary.DetectedCurrentLevel
	r.DetectedTargetLevel = summary.DetectedTargetLevel
	r.DetectedGoal = trimOptionalString(summary.DetectedGoal)
	r.DetectedLanguage = summary.DetectedLanguage
	r.ClarificationQuestions = cloneClarificationQuestions(summary.ClarificationQuestions)
	r.AnalysisCompletedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r GenerationRequest) NeedsClarification() bool {
	return !r.IsOutOfScope && len(r.ClarificationQuestions) > 0
}

func (r *GenerationRequest) MarkAwaitingClarification(now time.Time) error {
	if r.PipelineStatus != PipelineStatusRunning || r.AnalysisCompletedAt == nil || !r.NeedsClarification() || r.ConfirmedBrief != nil {
		return ErrGenerationRequestNotReady
	}
	if !r.PipelineStatus.CanTransitionTo(PipelineStatusAwaitingClarification) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, r.PipelineStatus, PipelineStatusAwaitingClarification)
	}
	step := "awaiting_clarification"
	r.PipelineStatus = PipelineStatusAwaitingClarification
	r.CurrentStep = &step
	r.ProgressPercent = 25
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) ConfirmDetectedBrief(now time.Time) error {
	if r.PipelineStatus != PipelineStatusRunning || r.AnalysisCompletedAt == nil || r.NeedsClarification() || r.IsOutOfScope {
		return ErrGenerationRequestNotReady
	}
	brief, err := r.detectedBrief()
	if err != nil {
		return err
	}
	r.confirmBrief(brief, nil, false, now)
	return nil
}

func (r *GenerationRequest) SubmitClarifications(
	answers []ClarificationAnswer,
	title string,
	synopsis string,
	language CourseLanguage,
	now time.Time,
) error {
	if r.PipelineStatus != PipelineStatusAwaitingClarification || r.AnalysisCompletedAt == nil || r.ConfirmedBrief != nil {
		return ErrGenerationRequestNotReady
	}
	if err := ValidateClarificationAnswers(r.ClarificationQuestions, answers); err != nil {
		return err
	}

	brief, err := r.briefFromAnswers(answers, title, synopsis, language)
	if err != nil {
		return err
	}
	if !r.PipelineStatus.CanTransitionTo(PipelineStatusQueued) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, r.PipelineStatus, PipelineStatusQueued)
	}
	r.confirmBrief(brief, answers, true, now)
	step := "clarifications_completed"
	r.PipelineStatus = PipelineStatusQueued
	r.CurrentStep = &step
	r.ProgressPercent = 25
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) ConfirmBrief(brief GenerationBrief, now time.Time) error {
	if r.PipelineStatus != PipelineStatusRunning && r.PipelineStatus != PipelineStatusAwaitingClarification {
		return ErrGenerationRequestNotReady
	}
	if err := brief.Validate(); err != nil {
		return err
	}
	if r.PipelineStatus == PipelineStatusAwaitingClarification {
		if !r.PipelineStatus.CanTransitionTo(PipelineStatusQueued) {
			return ErrGenerationRequestNotReady
		}
		r.PipelineStatus = PipelineStatusQueued
	}
	answers := cloneClarificationAnswers(r.ClarificationAnswers)
	submitted := r.ClarificationsSubmittedAt != nil
	r.confirmBrief(brief, answers, submitted, now)
	step := "brief_confirmed"
	r.CurrentStep = &step
	r.ProgressPercent = 25
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) MarkCompleted(now time.Time) error {
	if !r.PipelineStatus.CanTransitionTo(PipelineStatusCompleted) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, r.PipelineStatus, PipelineStatusCompleted)
	}
	r.PipelineStatus = PipelineStatusCompleted
	r.ProgressPercent = 100
	r.CompletedAt = &now
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) RestartFromFailure(step string, percent int, now time.Time) error {
	if r.PipelineStatus != PipelineStatusFailed {
		return ErrGenerationRequestNotReady
	}
	if err := validateProgressPercent(percent); err != nil {
		return err
	}
	step = normalizeText(step)
	if step == "" {
		return fmt.Errorf("%w: current step", ErrBlankField)
	}

	r.PipelineStatus = PipelineStatusRunning
	r.CurrentStep = &step
	r.ProgressPercent = percent
	r.FailureMessage = nil
	r.CompletedAt = nil
	if r.StartedAt == nil {
		r.StartedAt = &now
	}
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) MarkFailed(message string, now time.Time) error {
	message = normalizeText(message)
	if message == "" {
		return fmt.Errorf("%w: failure message", ErrBlankField)
	}
	if !r.PipelineStatus.CanTransitionTo(PipelineStatusFailed) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, r.PipelineStatus, PipelineStatusFailed)
	}
	r.PipelineStatus = PipelineStatusFailed
	r.FailureMessage = &message
	r.CompletedAt = &now
	r.UpdatedAt = now
	return nil
}

func (q ClarificationQuestion) Validate() error {
	if err := q.validateMetadata(); err != nil {
		return err
	}
	return q.validateOptions()
}

func (q ClarificationQuestion) validateMetadata() error {
	if err := requireNotBlank("clarification question id", q.ID); err != nil {
		return err
	}
	if err := requireNotBlank("clarification question", q.Question); err != nil {
		return err
	}
	if len(q.Options) < 2 || len(q.Options) > 4 {
		return fmt.Errorf("%w: options", ErrInvalidClarification)
	}
	if q.ID != ClarificationIDGoals && q.ID != ClarificationIDCurrentLevel && q.ID != ClarificationIDTargetLevel {
		return fmt.Errorf("%w: id %s", ErrInvalidClarification, q.ID)
	}
	if q.AllowMultiple != (q.ID == ClarificationIDGoals) {
		return fmt.Errorf("%w: allow multiple", ErrInvalidClarification)
	}
	return nil
}

func (q ClarificationQuestion) validateOptions() error {
	seen := make(map[string]struct{}, len(q.Options))
	for _, option := range q.Options {
		value := normalizeText(option.Value)
		if value == "" || normalizeText(option.Label) == "" {
			return fmt.Errorf("%w: option", ErrInvalidClarification)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%w: duplicate option %s", ErrInvalidClarification, value)
		}
		seen[value] = struct{}{}
		if err := q.validateLevelOption(value); err != nil {
			return err
		}
	}
	return nil
}

func (q ClarificationQuestion) validateLevelOption(value string) error {
	if q.ID != ClarificationIDCurrentLevel && q.ID != ClarificationIDTargetLevel {
		return nil
	}
	level, err := ParseLevel(value)
	if err != nil || level == LevelUnknown || (q.ID == ClarificationIDCurrentLevel && level == LevelExpert) {
		return fmt.Errorf("%w: level option %s", ErrInvalidClarification, value)
	}
	return nil
}

func ValidateClarificationAnswers(questions []ClarificationQuestion, answers []ClarificationAnswer) error {
	if len(questions) == 0 || len(answers) != len(questions) {
		return ErrClarificationAnswerMissing
	}
	questionByID := make(map[string]ClarificationQuestion, len(questions))
	for _, question := range questions {
		questionByID[question.ID] = question
	}
	seenAnswers := make(map[string]struct{}, len(answers))
	for _, answer := range answers {
		questionID := normalizeText(answer.QuestionID)
		question, exists := questionByID[questionID]
		if !exists {
			return fmt.Errorf("%w: %s", ErrClarificationAnswerUnknown, questionID)
		}
		if _, duplicate := seenAnswers[questionID]; duplicate {
			return fmt.Errorf("%w: duplicate question %s", ErrInvalidClarificationAnswer, questionID)
		}
		seenAnswers[questionID] = struct{}{}
		values := normalizeStringSlice(answer.SelectedValues)
		if len(values) == 0 || (!question.AllowMultiple && len(values) != 1) {
			return fmt.Errorf("%w: %s", ErrInvalidClarificationAnswer, questionID)
		}
		allowed := make(map[string]struct{}, len(question.Options))
		for _, option := range question.Options {
			allowed[normalizeText(option.Value)] = struct{}{}
		}
		selected := make(map[string]struct{}, len(values))
		for _, value := range values {
			if _, duplicate := selected[value]; duplicate {
				return fmt.Errorf("%w: duplicate value %s", ErrInvalidClarificationAnswer, value)
			}
			if _, exists := allowed[value]; !exists {
				return fmt.Errorf("%w: %s", ErrClarificationValueNotAllowed, value)
			}
			selected[value] = struct{}{}
		}
	}
	return nil
}

func (r GenerationRequest) detectedBrief() (GenerationBrief, error) {
	if r.SuggestedTitle == nil || r.ShortSynopsis == nil || r.DetectedCurrentLevel == nil || r.DetectedTargetLevel == nil || r.DetectedGoal == nil || r.DetectedLanguage == nil {
		return GenerationBrief{}, ErrGenerationBriefIncomplete
	}
	brief := GenerationBrief{
		Title:        normalizeText(*r.SuggestedTitle),
		Synopsis:     normalizeText(*r.ShortSynopsis),
		CurrentLevel: *r.DetectedCurrentLevel,
		TargetLevel:  *r.DetectedTargetLevel,
		Goals:        []string{normalizeText(*r.DetectedGoal)},
		Language:     *r.DetectedLanguage,
	}
	if isUnknownGenerationValue(brief.Goals[0]) {
		return GenerationBrief{}, ErrGenerationBriefIncomplete
	}
	if err := brief.Validate(); err != nil {
		return GenerationBrief{}, err
	}
	return brief, nil
}

func (r GenerationRequest) briefFromAnswers(answers []ClarificationAnswer, title, synopsis string, language CourseLanguage) (GenerationBrief, error) {
	brief := GenerationBrief{Title: normalizeText(title), Synopsis: normalizeText(synopsis), Language: language}
	if r.DetectedCurrentLevel != nil {
		brief.CurrentLevel = *r.DetectedCurrentLevel
	}
	if r.DetectedTargetLevel != nil {
		brief.TargetLevel = *r.DetectedTargetLevel
	}
	if r.DetectedGoal != nil && !isUnknownGenerationValue(*r.DetectedGoal) {
		brief.Goals = []string{*r.DetectedGoal}
	}
	if brief.Language == "" && r.DetectedLanguage != nil {
		brief.Language = *r.DetectedLanguage
	}
	for _, answer := range answers {
		values := normalizeStringSlice(answer.SelectedValues)
		switch normalizeText(answer.QuestionID) {
		case ClarificationIDCurrentLevel:
			level, err := ParseLevel(values[0])
			if err != nil {
				return GenerationBrief{}, err
			}
			brief.CurrentLevel = level
		case ClarificationIDTargetLevel:
			level, err := ParseLevel(values[0])
			if err != nil {
				return GenerationBrief{}, err
			}
			brief.TargetLevel = level
		case ClarificationIDGoals:
			brief.Goals = values
		}
	}
	if err := brief.Validate(); err != nil {
		return GenerationBrief{}, err
	}
	return brief, nil
}

func (r *GenerationRequest) confirmBrief(brief GenerationBrief, answers []ClarificationAnswer, submitted bool, now time.Time) {
	normalized := GenerationBrief{
		Title:        normalizeText(brief.Title),
		Synopsis:     normalizeText(brief.Synopsis),
		CurrentLevel: brief.CurrentLevel,
		TargetLevel:  brief.TargetLevel,
		Goals:        normalizeStringSlice(brief.Goals),
		Language:     brief.Language,
	}
	r.ConfirmedBrief = &normalized
	r.ClarificationAnswers = cloneClarificationAnswers(answers)
	r.ClarificationVersion++
	r.BriefConfirmedAt = &now
	if submitted {
		if r.ClarificationsSubmittedAt == nil {
			r.ClarificationsSubmittedAt = &now
		}
	}
}

func validateAnalysisClarifications(summary AnalysisSummary) error {
	if err := validateClarificationQuestions(summary.ClarificationQuestions); err != nil {
		return err
	}
	if summary.IsOutOfScope {
		if len(summary.ClarificationQuestions) != 0 {
			return ErrInvalidClarification
		}
		return nil
	}

	required := map[string]bool{
		ClarificationIDCurrentLevel: summary.DetectedCurrentLevel == nil || *summary.DetectedCurrentLevel == LevelUnknown,
		ClarificationIDTargetLevel:  summary.DetectedTargetLevel == nil || *summary.DetectedTargetLevel == LevelUnknown,
		ClarificationIDGoals:        summary.DetectedGoal == nil || isUnknownGenerationValue(*summary.DetectedGoal),
	}
	provided := make(map[string]bool, len(summary.ClarificationQuestions))
	for _, question := range summary.ClarificationQuestions {
		if !required[question.ID] {
			return fmt.Errorf("%w: unnecessary question %s", ErrInvalidClarification, question.ID)
		}
		provided[question.ID] = true
	}
	for id, isRequired := range required {
		if isRequired && !provided[id] {
			return fmt.Errorf("%w: missing question %s", ErrInvalidClarification, id)
		}
	}
	return nil
}

func validateClarificationQuestions(questions []ClarificationQuestion) error {
	seen := make(map[string]struct{}, len(questions))
	for _, question := range questions {
		if err := question.Validate(); err != nil {
			return err
		}
		if _, exists := seen[question.ID]; exists {
			return fmt.Errorf("%w: duplicate id %s", ErrInvalidClarification, question.ID)
		}
		seen[question.ID] = struct{}{}
	}
	return nil
}

func cloneClarificationQuestions(questions []ClarificationQuestion) []ClarificationQuestion {
	if len(questions) == 0 {
		return nil
	}
	cloned := make([]ClarificationQuestion, 0, len(questions))
	for _, question := range questions {
		options := make([]ClarificationOption, 0, len(question.Options))
		for _, option := range question.Options {
			options = append(options, ClarificationOption{Value: normalizeText(option.Value), Label: normalizeText(option.Label)})
		}
		cloned = append(cloned, ClarificationQuestion{
			ID:            normalizeText(question.ID),
			Question:      normalizeText(question.Question),
			Options:       options,
			AllowMultiple: question.AllowMultiple,
		})
	}
	return cloned
}

func cloneClarificationAnswers(answers []ClarificationAnswer) []ClarificationAnswer {
	if len(answers) == 0 {
		return nil
	}
	cloned := make([]ClarificationAnswer, 0, len(answers))
	for _, answer := range answers {
		cloned = append(cloned, ClarificationAnswer{
			QuestionID:     normalizeText(answer.QuestionID),
			SelectedValues: normalizeStringSlice(answer.SelectedValues),
		})
	}
	return cloned
}

func isUnknownGenerationValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "" || value == "unknown" || value == "unknow"
}
