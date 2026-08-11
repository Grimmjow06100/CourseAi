package dto

import (
	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

type ArchitecturePromptInput struct {
	Title        string   `json:"title"`
	Synopsis     string   `json:"synopsis"`
	CurrentLevel string   `json:"currentLevel"`
	TargetLevel  string   `json:"targetLevel"`
	Goals        []string `json:"goals"`
	Language     string   `json:"language"`
}

func ArchitecturePromptInputFromContract(input contract.ArchitectureInput) ArchitecturePromptInput {
	return ArchitecturePromptInput{
		Title:        input.Title,
		Synopsis:     input.Synopsis,
		CurrentLevel: string(input.CurrentLevel),
		TargetLevel:  string(input.TargetLevel),
		Goals:        input.Goals,
		Language:     string(input.Language),
	}
}

type ArchitectureResponse struct {
	Title          string                   `json:"title"`
	Synopsis       string                   `json:"synopsis"`
	TargetAudience string                   `json:"targetAudience"`
	Prerequisites  []string                 `json:"prerequisites"`
	Goals          []string                 `json:"goals"`
	AcquiredSkills []string                 `json:"acquiredSkills"`
	Modules        []architectureModule     `json:"modules"`
	FinalProject   architectureFinalProject `json:"finalProject"`
}

type architectureModule struct {
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	KeyLearningPoints []string `json:"keyLearningPoints"`
}

type architectureFinalProject struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Constraints []string `json:"constraints"`
}

func (r ArchitectureResponse) ToDomain(input contract.ArchitectureInput) (domain.Course, error) {
	language := input.Language
	if language == "" {
		language = domain.CourseLanguageFR
	}
	currentLevel := input.CurrentLevel
	if currentLevel == "" {
		currentLevel = domain.LevelUnknown
	}
	targetLevel := input.TargetLevel
	if targetLevel == "" {
		targetLevel = domain.LevelUnknown
	}

	course := domain.Course{
		RequestID:               input.Request.ID,
		Language:                language,
		InitialUserPrompt:       input.Request.InitialUserPrompt,
		Title:                   r.Title,
		Synopsis:                r.Synopsis,
		TargetAudience:          stringPtr(r.TargetAudience),
		CurrentLevel:            currentLevel,
		TargetLevel:             targetLevel,
		Prerequisites:           r.Prerequisites,
		Goals:                   r.Goals,
		AcquiredSkills:          r.AcquiredSkills,
		FinalProjectTitle:       stringPtr(r.FinalProject.Title),
		FinalProjectDescription: stringPtr(r.FinalProject.Description),
		FinalProjectConstraints: r.FinalProject.Constraints,
		Modules:                 make([]domain.Module, 0, len(r.Modules)),
	}

	for _, module := range r.Modules {
		course.Modules = append(course.Modules, domain.Module{
			Order:             module.Order,
			Title:             module.Title,
			Description:       module.Description,
			KeyLearningPoints: module.KeyLearningPoints,
		})
	}

	return course, nil
}
