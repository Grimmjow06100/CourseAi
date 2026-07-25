package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NewCourseParams struct {
	RequestID               uuid.UUID
	Language                CourseLanguage
	InitialUserPrompt       string
	Title                   string
	Synopsis                string
	TargetAudience          *string
	CurrentLevel            Level
	TargetLevel             Level
	Prerequisites           []string
	Goals                   []string
	AcquiredSkills          []string
	FinalProjectTitle       *string
	FinalProjectDescription *string
	FinalProjectConstraints []string
}

type Course struct {
	ID                      uuid.UUID
	RequestID               uuid.UUID
	Language                CourseLanguage
	Status                  CourseGenerationStatus
	InitialUserPrompt       string
	Title                   string
	Synopsis                string
	TargetAudience          *string
	CurrentLevel            Level
	TargetLevel             Level
	Prerequisites           []string
	Goals                   []string
	AcquiredSkills          []string
	FinalProjectTitle       *string
	FinalProjectDescription *string
	FinalProjectConstraints []string
	Modules                 []Module
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewCourse(params NewCourseParams) (Course, error) {
	return NewCourseAt(params, time.Now())
}

func NewCourseAt(params NewCourseParams, now time.Time) (Course, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Course{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	course := Course{
		ID:                      id,
		RequestID:               params.RequestID,
		Language:                params.Language,
		Status:                  CourseStatusAnalysisCompleted,
		InitialUserPrompt:       normalizeText(params.InitialUserPrompt),
		Title:                   normalizeText(params.Title),
		Synopsis:                normalizeText(params.Synopsis),
		TargetAudience:          trimOptionalString(params.TargetAudience),
		CurrentLevel:            params.CurrentLevel,
		TargetLevel:             params.TargetLevel,
		Prerequisites:           normalizeStringSlice(params.Prerequisites),
		Goals:                   normalizeStringSlice(params.Goals),
		AcquiredSkills:          normalizeStringSlice(params.AcquiredSkills),
		FinalProjectTitle:       trimOptionalString(params.FinalProjectTitle),
		FinalProjectDescription: trimOptionalString(params.FinalProjectDescription),
		FinalProjectConstraints: normalizeStringSlice(params.FinalProjectConstraints),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := course.Validate(); err != nil {
		return Course{}, err
	}
	return course, nil
}

func (c Course) Validate() error {
	if c.ID == uuid.Nil {
		return fmt.Errorf("%w: course id", ErrBlankField)
	}
	if c.RequestID == uuid.Nil {
		return fmt.Errorf("%w: request id", ErrBlankField)
	}
	if err := c.Language.Validate(); err != nil {
		return err
	}
	if err := c.Status.Validate(); err != nil {
		return err
	}
	if err := requireNotBlank("initial user prompt", c.InitialUserPrompt); err != nil {
		return err
	}
	if err := requireNotBlank("title", c.Title); err != nil {
		return err
	}
	if err := requireNotBlank("synopsis", c.Synopsis); err != nil {
		return err
	}
	if err := c.CurrentLevel.Validate(); err != nil {
		return err
	}
	if err := c.TargetLevel.Validate(); err != nil {
		return err
	}
	if err := validateUniqueModuleOrders(c.Modules); err != nil {
		return err
	}
	for _, module := range c.Modules {
		if err := module.Validate(); err != nil {
			return err
		}
	}
	if c.Status == CourseStatusCompleted && !c.HasCompleteContent() {
		return ErrMissingCourseContent
	}
	return nil
}

func (c *Course) AddModule(module Module) error {
	if module.CourseID == uuid.Nil {
		module.CourseID = c.ID
	}
	if module.CourseID != c.ID {
		return fmt.Errorf("%w: module course id does not match course id", ErrInvalidCollection)
	}
	for _, existing := range c.Modules {
		if existing.Order == module.Order {
			return ErrDuplicateModuleOrder
		}
	}
	if err := module.Validate(); err != nil {
		return err
	}
	c.Modules = append(c.Modules, module)
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Course) TransitionTo(next CourseGenerationStatus) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if !c.Status.CanTransitionTo(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, c.Status, next)
	}
	c.Status = next
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Course) MarkArchitectureGenerating() error {
	return c.TransitionTo(CourseStatusArchitectureGenerating)
}

func (c *Course) MarkArchitectureGenerated() error {
	return c.TransitionTo(CourseStatusStructureGenerated)
}

func (c *Course) MarkLessonsGenerating() error {
	return c.TransitionTo(CourseStatusLessonsGenerating)
}

func (c *Course) MarkLessonsGenerated() error {
	return c.TransitionTo(CourseStatusLessonsGenerated)
}

func (c *Course) MarkContentGenerating() error {
	return c.TransitionTo(CourseStatusContentGenerating)
}

func (c *Course) MarkCompleted() error {
	if !c.HasCompleteContent() {
		return ErrMissingCourseContent
	}
	return c.TransitionTo(CourseStatusCompleted)
}

func (c *Course) MarkFailed() error {
	return c.TransitionTo(CourseStatusFailed)
}

func (c Course) TotalDurationMinutes() int {
	total := 0
	for _, module := range c.Modules {
		total += module.TotalDurationMinutes()
	}
	return total
}

func (c Course) HasCompleteContent() bool {
	if len(c.Modules) == 0 {
		return false
	}
	for _, module := range c.Modules {
		if !module.HasCompleteLessons() {
			return false
		}
	}
	return true
}

func validateUniqueModuleOrders(modules []Module) error {
	seen := make(map[int]struct{}, len(modules))
	for _, module := range modules {
		if _, ok := seen[module.Order]; ok {
			return ErrDuplicateModuleOrder
		}
		seen[module.Order] = struct{}{}
	}
	return nil
}
