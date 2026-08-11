package dto

import (
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestLessonContentResponseMapsStructuredActivitiesToDomain(t *testing.T) {
	t.Parallel()

	lessonID := uuid.New()
	answer := "Un conteneur isole un processus avec ses dependances."
	response := LessonContentResponse{
		ContentMarkdown: "# Introduction aux conteneurs",
		Exercises: []ExerciseResponse{
			{
				Type:                 "command_line",
				Difficulty:           "beginner",
				Title:                "Inspecter un conteneur",
				Objective:            "Manipuler les commandes de base Docker.",
				InstructionsMarkdown: "Execute les commandes sans ouvrir la correction.",
				ContentMarkdown:      "Liste les conteneurs actifs puis inspecte leurs ports exposes.",
				CorrectionMarkdown:   "Utilise `docker ps`, puis `docker inspect <container>`.",
				Payload: ExercisePayloadResponse{
					Tasks:          []string{"Lister les conteneurs", "Inspecter un conteneur"},
					Resources:      []string{"docker ps", "docker inspect"},
					StarterCode:    nil,
					ExpectedOutput: stringPtr("Les ports exposes sont identifies."),
					Hints:          []string{"Commence par lister les conteneurs actifs."},
				},
			},
		},
		Quizzes: []QuizResponse{
			{
				Type:       "short_answer",
				Difficulty: "beginner",
				Title:      "Validation rapide",
				Objective:  "Verifier la comprehension du role d'un conteneur.",
				Questions: []QuizQuestionResponse{
					{
						Order:    1,
						Type:     "short_answer",
						Question: "A quoi sert un conteneur ?",
						Options:  []QuizOptionResponse{},
						Answer: QuizAnswerResponse{
							Answer:  &answer,
							Answers: []string{},
						},
						Correction: "Un conteneur fournit un environnement d'execution isole et reproductible.",
					},
				},
			},
		},
	}

	exercises, err := response.ExercisesToDomain(lessonID)
	if err != nil {
		t.Fatalf("expected exercises to map to domain: %v", err)
	}
	if len(exercises) != 1 {
		t.Fatalf("expected one exercise, got %d", len(exercises))
	}
	if exercises[0].LessonID != lessonID {
		t.Fatalf("expected exercise lesson id %s, got %s", lessonID, exercises[0].LessonID)
	}

	quizzes, err := response.QuizzesToDomain(lessonID)
	if err != nil {
		t.Fatalf("expected quizzes to map to domain: %v", err)
	}
	if len(quizzes) != 1 {
		t.Fatalf("expected one quiz, got %d", len(quizzes))
	}
	if quizzes[0].Questions[0].Answer.Answer == nil {
		t.Fatal("expected short answer to be present")
	}
}

func TestLessonContentResponseRejectsWeakQuizShape(t *testing.T) {
	t.Parallel()

	answer := "ls"
	response := LessonContentResponse{
		ContentMarkdown: "# Commandes Linux",
		Quizzes: []QuizResponse{
			{
				Type:       "single_choice",
				Difficulty: "beginner",
				Title:      "Quiz shell",
				Objective:  "Verifier les commandes de base.",
				Questions: []QuizQuestionResponse{
					{
						Order:    1,
						Type:     "single_choice",
						Question: "Quelle commande liste les fichiers ?",
						Options: []QuizOptionResponse{
							{Order: 1, Text: "ls"},
						},
						Answer:     QuizAnswerResponse{Answer: &answer, Answers: []string{}},
						Correction: "`ls` liste les fichiers.",
					},
				},
			},
		},
	}

	_, err := response.QuizzesToDomain(uuid.New())
	if !errors.Is(err, domain.ErrInvalidCollection) {
		t.Fatalf("expected invalid collection from weak quiz shape, got: %v", err)
	}
}
