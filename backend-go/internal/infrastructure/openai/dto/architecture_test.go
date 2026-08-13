package dto

import (
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestArchitecturePromptInputFromContract(t *testing.T) {
	t.Parallel()

	input := ArchitecturePromptInputFromContract(contract.ArchitectureInput{
		Title: "Linux", Synopsis: "Course", CurrentLevel: domain.LevelBeginner,
		TargetLevel: domain.LevelAdvanced, Goals: []string{"Admin"}, Language: domain.CourseLanguageEN,
	})
	if input.Title != "Linux" || input.CurrentLevel != "beginner" || input.TargetLevel != "advanced" || input.Language != "en" {
		t.Fatalf("unexpected prompt input: %+v", input)
	}
}

func TestArchitectureResponseToDomainAppliesDefaultsAndRelations(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	response := ArchitectureResponse{
		Title: "Linux", Synopsis: "Course", TargetAudience: " Developers ",
		Modules:      []architectureModule{{Order: 1, Title: "Basics", Description: "Commands", KeyLearningPoints: []string{"shell"}}},
		FinalProject: architectureFinalProject{Title: "Server", Description: "Deploy it", Constraints: []string{"Secure"}},
	}
	course, err := response.ToDomain(contract.ArchitectureInput{
		Request: domain.GenerationRequest{ID: requestID, InitialUserPrompt: "Learn Linux"},
	})
	if err != nil {
		t.Fatalf("ToDomain() error = %v", err)
	}
	if course.RequestID != requestID || course.Language != domain.CourseLanguageFR || course.CurrentLevel != domain.LevelUnknown || course.TargetLevel != domain.LevelUnknown {
		t.Fatalf("unexpected defaults: %+v", course)
	}
	if len(course.Modules) != 1 || course.Modules[0].Order != 1 || course.TargetAudience == nil || *course.TargetAudience != "Developers" {
		t.Fatalf("unexpected mapped course: %+v", course)
	}
}
