package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ClarificationQuestion struct {
	ID       string
	Question string
	Options  []string
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
	ID                     uuid.UUID
	InitialUserPrompt      string
	PipelineStatus         GenerationPipelineStatus
	CurrentStep            *string
	ProgressPercent        int
	FailureMessage         *string
	StartedAt              *time.Time
	CompletedAt            *time.Time
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
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func NewGenerationRequest(prompt string) (GenerationRequest, error) {
	return NewGenerationRequestAt(prompt, time.Now())
}

func NewGenerationRequestAt(prompt string, now time.Time) (GenerationRequest, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return GenerationRequest{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	request := GenerationRequest{
		ID:                id,
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
	if r.ID == uuid.Nil {
		return fmt.Errorf("%w: generation request id", ErrBlankField)
	}
	if err := requireNotBlank("initial user prompt", r.InitialUserPrompt); err != nil {
		return err
	}
	if err := r.PipelineStatus.Validate(); err != nil {
		return err
	}
	if err := validateProgressPercent(r.ProgressPercent); err != nil {
		return err
	}
	if r.PipelineStatus == PipelineStatusFailed && r.FailureMessage == nil {
		return fmt.Errorf("%w: failure message", ErrBlankField)
	}
	if r.PipelineStatus.IsTerminal() && r.CompletedAt == nil {
		return fmt.Errorf("%w: completed at", ErrBlankField)
	}
	for _, question := range r.ClarificationQuestions {
		if err := question.Validate(); err != nil {
			return err
		}
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
	r.StartedAt = &now
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
	for _, question := range summary.ClarificationQuestions {
		if err := question.Validate(); err != nil {
			return err
		}
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
	r.UpdatedAt = now
	return nil
}

func (r *GenerationRequest) NeedsClarification(questions []ClarificationQuestion, now time.Time) error {
	if len(questions) == 0 {
		return fmt.Errorf("%w: clarification questions", ErrInvalidClarification)
	}
	for _, question := range questions {
		if err := question.Validate(); err != nil {
			return err
		}
	}
	r.ClarificationQuestions = cloneClarificationQuestions(questions)
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
	if err := requireNotBlank("clarification question id", q.ID); err != nil {
		return err
	}
	if err := requireNotBlank("clarification question", q.Question); err != nil {
		return err
	}
	if len(normalizeStringSlice(q.Options)) == 0 {
		return fmt.Errorf("%w: options", ErrInvalidClarification)
	}
	return nil
}

func cloneClarificationQuestions(questions []ClarificationQuestion) []ClarificationQuestion {
	if len(questions) == 0 {
		return nil
	}
	cloned := make([]ClarificationQuestion, 0, len(questions))
	for _, question := range questions {
		cloned = append(cloned, ClarificationQuestion{
			ID:       normalizeText(question.ID),
			Question: normalizeText(question.Question),
			Options:  normalizeStringSlice(question.Options),
		})
	}
	return cloned
}
