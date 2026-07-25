package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NewModuleParams struct {
	CourseID          uuid.UUID
	Order             int
	Title             string
	Description       string
	KeyLearningPoints []string
}

type Module struct {
	ID                uuid.UUID
	CourseID          uuid.UUID
	Order             int
	Title             string
	Description       string
	KeyLearningPoints []string
	Lessons           []Lesson
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewModule(params NewModuleParams) (Module, error) {
	return NewModuleAt(params, time.Now())
}

func NewModuleAt(params NewModuleParams, now time.Time) (Module, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Module{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	module := Module{
		ID:                id,
		CourseID:          params.CourseID,
		Order:             params.Order,
		Title:             normalizeText(params.Title),
		Description:       normalizeText(params.Description),
		KeyLearningPoints: normalizeStringSlice(params.KeyLearningPoints),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := module.Validate(); err != nil {
		return Module{}, err
	}
	return module, nil
}

func (m Module) Validate() error {
	if m.ID == uuid.Nil {
		return fmt.Errorf("%w: module id", ErrBlankField)
	}
	if m.CourseID == uuid.Nil {
		return fmt.Errorf("%w: course id", ErrBlankField)
	}
	if err := validatePositiveInt("module order", m.Order); err != nil {
		return err
	}
	if err := requireNotBlank("module title", m.Title); err != nil {
		return err
	}
	if err := requireNotBlank("module description", m.Description); err != nil {
		return err
	}
	if err := validateUniqueLessonOrders(m.Lessons); err != nil {
		return err
	}
	for _, lesson := range m.Lessons {
		if err := lesson.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) AddLesson(lesson Lesson) error {
	if lesson.ModuleID == uuid.Nil {
		lesson.ModuleID = m.ID
	}
	if lesson.ModuleID != m.ID {
		return fmt.Errorf("%w: lesson module id does not match module id", ErrInvalidCollection)
	}
	for _, existing := range m.Lessons {
		if existing.Order == lesson.Order {
			return ErrDuplicateLessonOrder
		}
	}
	if err := lesson.Validate(); err != nil {
		return err
	}
	m.Lessons = append(m.Lessons, lesson)
	m.UpdatedAt = time.Now()
	return nil
}

func (m Module) TotalDurationMinutes() int {
	total := 0
	for _, lesson := range m.Lessons {
		total += lesson.EstimatedDurationMinutes
	}
	return total
}

func (m Module) HasCompleteLessons() bool {
	if len(m.Lessons) == 0 {
		return false
	}
	for _, lesson := range m.Lessons {
		if !lesson.HasContent() {
			return false
		}
	}
	return true
}

func validateUniqueLessonOrders(lessons []Lesson) error {
	seen := make(map[int]struct{}, len(lessons))
	for _, lesson := range lessons {
		if _, ok := seen[lesson.Order]; ok {
			return ErrDuplicateLessonOrder
		}
		seen[lesson.Order] = struct{}{}
	}
	return nil
}
