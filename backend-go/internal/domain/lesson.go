package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NewLessonParams struct {
	ModuleID                 uuid.UUID
	Order                    int
	Title                    string
	Type                     LessonType
	EstimatedDurationMinutes int
	LearningGoal             string
	RequiresDiagram          bool
	TechnicalKeywords        []string
}

type Lesson struct {
	ID                       uuid.UUID
	ModuleID                 uuid.UUID
	Order                    int
	Title                    string
	Type                     LessonType
	EstimatedDurationMinutes int
	LearningGoal             string
	RequiresDiagram          bool
	TechnicalKeywords        []string
	ContentMarkdown          *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func NewLesson(params NewLessonParams) (Lesson, error) {
	return NewLessonAt(params, time.Now())
}

func NewLessonAt(params NewLessonParams, now time.Time) (Lesson, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Lesson{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	lesson := Lesson{
		ID:                       id,
		ModuleID:                 params.ModuleID,
		Order:                    params.Order,
		Title:                    normalizeText(params.Title),
		Type:                     params.Type,
		EstimatedDurationMinutes: params.EstimatedDurationMinutes,
		LearningGoal:             normalizeText(params.LearningGoal),
		RequiresDiagram:          params.RequiresDiagram,
		TechnicalKeywords:        normalizeStringSlice(params.TechnicalKeywords),
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := lesson.Validate(); err != nil {
		return Lesson{}, err
	}
	return lesson, nil
}

func (l Lesson) Validate() error {
	if l.ID == uuid.Nil {
		return fmt.Errorf("%w: lesson id", ErrBlankField)
	}
	if l.ModuleID == uuid.Nil {
		return fmt.Errorf("%w: module id", ErrBlankField)
	}
	if err := validatePositiveInt("lesson order", l.Order); err != nil {
		return err
	}
	if err := requireNotBlank("lesson title", l.Title); err != nil {
		return err
	}
	if err := l.Type.Validate(); err != nil {
		return err
	}
	if l.EstimatedDurationMinutes <= 0 {
		return ErrInvalidDuration
	}
	if err := requireNotBlank("learning goal", l.LearningGoal); err != nil {
		return err
	}
	if l.ContentMarkdown != nil && strings.TrimSpace(*l.ContentMarkdown) == "" {
		return fmt.Errorf("%w: content markdown", ErrBlankField)
	}
	return nil
}

func (l *Lesson) AttachContent(markdown string) error {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return fmt.Errorf("%w: content markdown", ErrBlankField)
	}
	l.ContentMarkdown = &markdown
	l.UpdatedAt = time.Now()
	return nil
}

func (l Lesson) HasContent() bool {
	return l.ContentMarkdown != nil && strings.TrimSpace(*l.ContentMarkdown) != ""
}
