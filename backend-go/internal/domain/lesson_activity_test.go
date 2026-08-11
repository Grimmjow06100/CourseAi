package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewExerciseValidatesRequiredFields(t *testing.T) {
	lessonID := uuid.New()

	exercise, err := NewExerciseAt(NewExerciseParams{
		LessonID:             lessonID,
		Type:                 ExerciseTypeGuidedLab,
		Difficulty:           DifficultyBeginner,
		Title:                "Docker lab",
		Objective:            "Run a container",
		InstructionsMarkdown: "Follow the steps.",
		ContentMarkdown:      "Run `docker ps`.",
		CorrectionMarkdown:   "A valid correction.",
		Payload: ExercisePayload{
			"expectedCommands": []string{"docker ps"},
		},
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("expected valid exercise, got: %v", err)
	}

	if exercise.LessonID != lessonID {
		t.Fatalf("expected lesson id %s, got %s", lessonID, exercise.LessonID)
	}
	if exercise.Payload == nil {
		t.Fatal("expected payload to be initialized")
	}
}

func TestQuizRejectsQuestionTypeMismatchWhenQuizIsNotMixed(t *testing.T) {
	answer := "B"
	_, err := NewQuizAt(NewQuizParams{
		LessonID:   uuid.New(),
		Type:       QuizTypeSingleChoice,
		Difficulty: DifficultyBeginner,
		Title:      "Docker quiz",
		Objective:  "Validate Docker basics",
		Questions: []QuizQuestion{
			{
				Order:    1,
				Type:     QuizQuestionTypeMultipleChoice,
				Question: "Select valid Docker commands.",
				Options: []QuizOption{
					{Order: 1, Text: "A"},
					{Order: 2, Text: "B"},
				},
				Answer:     QuizAnswer{Answer: &answer},
				Correction: "Docker has several commands.",
			},
		},
	}, time.Unix(0, 0))

	if !errors.Is(err, ErrInvalidQuizAnswer) && !errors.Is(err, ErrInvalidQuizQuestionType) {
		t.Fatalf("expected quiz validation error, got: %v", err)
	}
}

func TestMixedQuizAcceptsDifferentQuestionTypes(t *testing.T) {
	singleAnswer := "docker ps"
	multipleAnswers := []string{"FROM", "RUN"}

	quiz, err := NewQuizAt(NewQuizParams{
		LessonID:   uuid.New(),
		Type:       QuizTypeMixed,
		Difficulty: DifficultyIntermediate,
		Title:      "Docker mixed quiz",
		Objective:  "Validate Docker knowledge",
		Questions: []QuizQuestion{
			{
				Order:    1,
				Type:     QuizQuestionTypeSingleChoice,
				Question: "Which command lists running containers?",
				Options: []QuizOption{
					{Order: 1, Text: "docker images"},
					{Order: 2, Text: "docker ps"},
				},
				Answer:     QuizAnswer{Answer: &singleAnswer},
				Correction: "`docker ps` lists running containers.",
			},
			{
				Order:    2,
				Type:     QuizQuestionTypeMultipleChoice,
				Question: "Which Dockerfile instructions are valid?",
				Options: []QuizOption{
					{Order: 1, Text: "FROM"},
					{Order: 2, Text: "RUN"},
					{Order: 3, Text: "DOCKER_LOGIN"},
				},
				Answer:     QuizAnswer{Answers: multipleAnswers},
				Correction: "`FROM` and `RUN` are valid Dockerfile instructions.",
			},
		},
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("expected mixed quiz to be valid, got: %v", err)
	}

	if len(quiz.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(quiz.Questions))
	}
}

func TestQuizRejectsOptionCountsThatAreTooLow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		quizType  QuizType
		question  QuizQuestion
		wantError error
	}{
		{
			name:     "single choice requires at least two options",
			quizType: QuizTypeSingleChoice,
			question: QuizQuestion{
				Order:    1,
				Type:     QuizQuestionTypeSingleChoice,
				Question: "Quelle commande liste les fichiers ?",
				Options: []QuizOption{
					{Order: 1, Text: "ls"},
				},
				Answer:     QuizAnswer{Answer: stringPtr("ls")},
				Correction: "`ls` liste les fichiers.",
			},
			wantError: ErrInvalidCollection,
		},
		{
			name:     "multiple choice requires at least three options",
			quizType: QuizTypeMultipleChoice,
			question: QuizQuestion{
				Order:    1,
				Type:     QuizQuestionTypeMultipleChoice,
				Question: "Quelles commandes affichent des informations ?",
				Options: []QuizOption{
					{Order: 1, Text: "pwd"},
					{Order: 2, Text: "ls"},
				},
				Answer:     QuizAnswer{Answers: []string{"pwd", "ls"}},
				Correction: "`pwd` et `ls` affichent des informations utiles.",
			},
			wantError: ErrInvalidCollection,
		},
		{
			name:     "true false requires exactly two options",
			quizType: QuizTypeTrueFalse,
			question: QuizQuestion{
				Order:    1,
				Type:     QuizQuestionTypeTrueFalse,
				Question: "Linux est un noyau.",
				Options: []QuizOption{
					{Order: 1, Text: "Vrai"},
					{Order: 2, Text: "Faux"},
					{Order: 3, Text: "Cela depend"},
				},
				Answer:     QuizAnswer{Answer: stringPtr("Vrai")},
				Correction: "Linux designe le noyau du systeme.",
			},
			wantError: ErrInvalidCollection,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewQuizAt(NewQuizParams{
				LessonID:   uuid.New(),
				Type:       tc.quizType,
				Difficulty: DifficultyBeginner,
				Title:      "Quiz Linux",
				Objective:  "Verifier les fondamentaux Linux.",
				Questions:  []QuizQuestion{tc.question},
			}, time.Unix(0, 0))
			if !errors.Is(err, tc.wantError) {
				t.Fatalf("expected %v, got %v", tc.wantError, err)
			}
		})
	}
}

func TestQuizRejectsDuplicateMultipleChoiceAnswers(t *testing.T) {
	t.Parallel()

	_, err := NewQuizAt(NewQuizParams{
		LessonID:   uuid.New(),
		Type:       QuizTypeMultipleChoice,
		Difficulty: DifficultyBeginner,
		Title:      "Quiz Linux",
		Objective:  "Verifier les commandes Linux.",
		Questions: []QuizQuestion{
			{
				Order:    1,
				Type:     QuizQuestionTypeMultipleChoice,
				Question: "Quelles commandes affichent des informations ?",
				Options: []QuizOption{
					{Order: 1, Text: "pwd"},
					{Order: 2, Text: "ls"},
					{Order: 3, Text: "cd"},
				},
				Answer:     QuizAnswer{Answers: []string{"pwd", " PWD "}},
				Correction: "`pwd` affiche le repertoire courant.",
			},
		},
	}, time.Unix(0, 0))
	if !errors.Is(err, ErrInvalidQuizAnswer) {
		t.Fatalf("expected invalid quiz answer error, got: %v", err)
	}
}

func TestQuizRejectsNonContiguousQuestionAndOptionOrders(t *testing.T) {
	t.Parallel()

	answer := "ls"
	_, err := NewQuizAt(NewQuizParams{
		LessonID:   uuid.New(),
		Type:       QuizTypeSingleChoice,
		Difficulty: DifficultyBeginner,
		Title:      "Quiz Linux",
		Objective:  "Verifier les commandes Linux.",
		Questions: []QuizQuestion{
			{
				Order:    1,
				Type:     QuizQuestionTypeSingleChoice,
				Question: "Quelle commande liste les fichiers ?",
				Options: []QuizOption{
					{Order: 1, Text: "pwd"},
					{Order: 2, Text: "ls"},
				},
				Answer:     QuizAnswer{Answer: &answer},
				Correction: "`ls` liste les fichiers.",
			},
			{
				Order:    3,
				Type:     QuizQuestionTypeSingleChoice,
				Question: "Quelle commande affiche le dossier courant ?",
				Options: []QuizOption{
					{Order: 1, Text: "pwd"},
					{Order: 2, Text: "cd"},
				},
				Answer:     QuizAnswer{Answer: stringPtr("pwd")},
				Correction: "`pwd` affiche le dossier courant.",
			},
		},
	}, time.Unix(0, 0))
	if !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("expected invalid collection for question order gap, got: %v", err)
	}

	_, err = NewQuizAt(NewQuizParams{
		LessonID:   uuid.New(),
		Type:       QuizTypeSingleChoice,
		Difficulty: DifficultyBeginner,
		Title:      "Quiz Linux",
		Objective:  "Verifier les commandes Linux.",
		Questions: []QuizQuestion{
			{
				Order:    1,
				Type:     QuizQuestionTypeSingleChoice,
				Question: "Quelle commande liste les fichiers ?",
				Options: []QuizOption{
					{Order: 1, Text: "pwd"},
					{Order: 3, Text: "ls"},
				},
				Answer:     QuizAnswer{Answer: stringPtr("ls")},
				Correction: "`ls` liste les fichiers.",
			},
		},
	}, time.Unix(0, 0))
	if !errors.Is(err, ErrInvalidCollection) {
		t.Fatalf("expected invalid collection for option order gap, got: %v", err)
	}
}

func TestValidateLessonActivitiesForType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		lesson    LessonType
		exercises []Exercise
		quizzes   []Quiz
		wantError bool
	}{
		{
			name:      "practice requires an exercise",
			lesson:    LessonTypePractice,
			exercises: nil,
			quizzes:   nil,
			wantError: true,
		},
		{
			name:      "practice rejects quizzes",
			lesson:    LessonTypePractice,
			exercises: []Exercise{{}},
			quizzes:   []Quiz{{}},
			wantError: true,
		},
		{
			name:      "quiz requires a quiz",
			lesson:    LessonTypeQuiz,
			exercises: nil,
			quizzes:   nil,
			wantError: true,
		},
		{
			name:      "quiz rejects exercises",
			lesson:    LessonTypeQuiz,
			exercises: []Exercise{{}},
			quizzes:   []Quiz{{}},
			wantError: true,
		},
		{
			name:      "mixed requires exercise and quiz",
			lesson:    LessonTypeMixed,
			exercises: []Exercise{{}},
			quizzes:   nil,
			wantError: true,
		},
		{
			name:      "theory rejects exercises",
			lesson:    LessonTypeTheory,
			exercises: []Exercise{{}},
			quizzes:   nil,
			wantError: true,
		},
		{
			name:      "theory accepts optional quiz recap",
			lesson:    LessonTypeTheory,
			exercises: nil,
			quizzes:   []Quiz{{}},
			wantError: false,
		},
		{
			name:      "mixed accepts exercise and quiz",
			lesson:    LessonTypeMixed,
			exercises: []Exercise{{}},
			quizzes:   []Quiz{{}},
			wantError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLessonActivitiesForType(tc.lesson, tc.exercises, tc.quizzes)
			if tc.wantError && !errors.Is(err, ErrInvalidCollection) {
				t.Fatalf("expected invalid collection error, got: %v", err)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected valid activity policy, got: %v", err)
			}
		})
	}
}

func TestLessonHasContentWithStructuredActivity(t *testing.T) {
	lesson, err := NewLessonAt(NewLessonParams{
		ModuleID:                 uuid.New(),
		Order:                    1,
		Title:                    "Practice Docker",
		Type:                     LessonTypePractice,
		EstimatedDurationMinutes: 30,
		LearningGoal:             "Practice Docker commands",
		TechnicalKeywords:        []string{"docker"},
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("expected lesson to be valid: %v", err)
	}

	exercise, err := NewExerciseAt(NewExerciseParams{
		LessonID:             lesson.ID,
		Type:                 ExerciseTypeCommandLine,
		Difficulty:           DifficultyBeginner,
		Title:                "Run Docker ps",
		Objective:            "List containers",
		InstructionsMarkdown: "Run the command.",
		ContentMarkdown:      "Use the terminal.",
		CorrectionMarkdown:   "`docker ps` is expected.",
	}, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("expected exercise to be valid: %v", err)
	}

	if err := lesson.AddExercise(exercise); err != nil {
		t.Fatalf("expected exercise to be added: %v", err)
	}
	if !lesson.HasContent() {
		t.Fatal("expected structured activity to count as lesson content")
	}
}

func stringPtr(value string) *string {
	return &value
}
