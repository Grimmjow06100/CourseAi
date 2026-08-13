package dto

import (
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/pointer"
	"github.com/google/uuid"
)

func TestLessonContentPromptInputFromDomain(t *testing.T) {
	t.Parallel()

	course := domain.Course{ID: uuid.New(), Language: domain.CourseLanguageEN, Title: "Linux", CurrentLevel: domain.LevelBeginner, TargetLevel: domain.LevelAdvanced}
	module := domain.Module{ID: uuid.New(), Order: 1, Title: "Shell"}
	lesson := domain.Lesson{ID: uuid.New(), Order: 2, Title: "Pipes", Type: domain.LessonTypePractice, EstimatedDurationMinutes: 25}
	input := LessonContentPromptInputFromDomain(course, module, lesson)
	if input.Course.ID != course.ID.String() || input.Course.LevelFrom != "beginner" || input.Module.ID != module.ID.String() ||
		input.Lesson.ID != lesson.ID.String() || input.Lesson.Type != "practice" {
		t.Fatalf("unexpected lesson content input: %+v", input)
	}
}

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
					ExpectedOutput: pointer.To("Les ports exposes sont identifies."),
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

func TestLessonContentResponseRejectsInvalidActivityEnums(t *testing.T) {
	t.Parallel()

	response := LessonContentResponse{Exercises: []ExerciseResponse{{Type: "unknown", Difficulty: "beginner"}}}
	if _, err := response.ExercisesToDomain(uuid.New()); !errors.Is(err, domain.ErrInvalidExerciseType) {
		t.Fatalf("invalid exercise type error = %v", err)
	}
	response = LessonContentResponse{Quizzes: []QuizResponse{{Type: "unknown", Difficulty: "beginner"}}}
	if _, err := response.QuizzesToDomain(uuid.New()); !errors.Is(err, domain.ErrInvalidQuizType) {
		t.Fatalf("invalid quiz type error = %v", err)
	}
}

func TestQuizResponseHelpers(t *testing.T) {
	t.Parallel()

	answer := " yes "
	quiz := QuizResponse{Questions: []QuizQuestionResponse{{
		Order: 1, Type: "short_answer", Question: "Ready?",
		Options: []QuizOptionResponse{}, Answer: QuizAnswerResponse{Answer: &answer}, Correction: "Yes",
	}}}
	questions, err := quiz.QuestionsToDomain()
	if err != nil {
		t.Fatalf("QuestionsToDomain() error = %v", err)
	}
	if len(questions) != 1 || questions[0].Answer.Answer == nil || *questions[0].Answer.Answer != "yes" {
		t.Fatalf("unexpected questions: %+v", questions)
	}
	quiz.Questions[0].Type = "essay"
	if _, err := quiz.QuestionsToDomain(); !errors.Is(err, domain.ErrInvalidQuizQuestionType) {
		t.Fatalf("invalid question type error = %v", err)
	}
}
