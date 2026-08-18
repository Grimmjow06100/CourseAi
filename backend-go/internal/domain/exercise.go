package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ExercisePayload map[string]any

type NewExerciseParams struct {
	LessonID             uuid.UUID
	Type                 ExerciseType
	Difficulty           Difficulty
	Title                string
	Objective            string
	InstructionsMarkdown string
	ContentMarkdown      string
	CorrectionMarkdown   string
	Payload              ExercisePayload
}

type Exercise struct {
	ID                   uuid.UUID
	LessonID             uuid.UUID
	Type                 ExerciseType
	Difficulty           Difficulty
	Title                string
	Objective            string
	InstructionsMarkdown string
	ContentMarkdown      string
	CorrectionMarkdown   string
	Payload              ExercisePayload
	RawAIOutput          json.RawMessage
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewExercise(params NewExerciseParams) (Exercise, error) {
	return NewExerciseAt(params, time.Now())
}

func NewExerciseAt(params NewExerciseParams, now time.Time) (Exercise, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Exercise{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	exercise := Exercise{
		ID:                   id,
		LessonID:             params.LessonID,
		Type:                 params.Type,
		Difficulty:           params.Difficulty,
		Title:                normalizeText(params.Title),
		Objective:            normalizeText(params.Objective),
		InstructionsMarkdown: normalizeMarkdown(params.InstructionsMarkdown),
		ContentMarkdown:      normalizeMarkdown(params.ContentMarkdown),
		CorrectionMarkdown:   normalizeMarkdown(params.CorrectionMarkdown),
		Payload:              normalizeExercisePayload(params.Payload),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := exercise.Validate(); err != nil {
		return Exercise{}, err
	}
	return exercise, nil
}

func (e Exercise) Validate() error {
	if e.ID == uuid.Nil {
		return fmt.Errorf("%w: exercise id", ErrBlankField)
	}
	if e.LessonID == uuid.Nil {
		return fmt.Errorf("%w: lesson id", ErrBlankField)
	}
	if err := e.Type.Validate(); err != nil {
		return err
	}
	if err := e.Difficulty.Validate(); err != nil {
		return err
	}
	if err := requireNotBlank("exercise title", e.Title); err != nil {
		return err
	}
	if err := requireNotBlank("exercise objective", e.Objective); err != nil {
		return err
	}
	if err := requireNotBlank("exercise instructions markdown", e.InstructionsMarkdown); err != nil {
		return err
	}
	if err := requireNotBlank("exercise content markdown", e.ContentMarkdown); err != nil {
		return err
	}
	if err := requireNotBlank("exercise correction markdown", e.CorrectionMarkdown); err != nil {
		return err
	}
	return nil
}

func (e *Exercise) AttachCorrection(correctionMarkdown string) error {
	correctionMarkdown = normalizeMarkdown(correctionMarkdown)
	if correctionMarkdown == "" {
		return fmt.Errorf("%w: exercise correction markdown", ErrBlankField)
	}
	e.CorrectionMarkdown = correctionMarkdown
	e.UpdatedAt = time.Now()
	return nil
}

func (e *Exercise) ReplacePayload(payload ExercisePayload) {
	e.Payload = normalizeExercisePayload(payload)
	e.UpdatedAt = time.Now()
}

func normalizeExercisePayload(payload ExercisePayload) ExercisePayload {
	if payload == nil {
		return ExercisePayload{}
	}
	return payload
}
