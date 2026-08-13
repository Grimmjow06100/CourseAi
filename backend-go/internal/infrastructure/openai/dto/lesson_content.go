package dto

import (
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/textutil"
	"github.com/google/uuid"
)

type LessonContentPromptInput struct {
	Course lessonContentCourse `json:"course"`
	Module lessonContentModule `json:"module"`
	Lesson lessonContentLesson `json:"lesson"`
}

type lessonContentCourse struct {
	ID        string   `json:"id"`
	Language  string   `json:"language"`
	Title     string   `json:"title"`
	Synopsis  string   `json:"synopsis"`
	Goals     []string `json:"goals"`
	LevelFrom string   `json:"currentLevel"`
	LevelTo   string   `json:"targetLevel"`
}

type lessonContentModule struct {
	ID                string   `json:"id"`
	Order             int      `json:"order"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	KeyLearningPoints []string `json:"keyLearningPoints"`
}

type lessonContentLesson struct {
	ID                       string   `json:"id"`
	Order                    int      `json:"order"`
	Title                    string   `json:"title"`
	Type                     string   `json:"type"`
	EstimatedDurationMinutes int      `json:"estimatedDurationMinutes"`
	LearningGoal             string   `json:"learningGoal"`
	RequiresDiagram          bool     `json:"requiresDiagram"`
	TechnicalKeywords        []string `json:"technicalKeywords"`
}

func LessonContentPromptInputFromDomain(course domain.Course, module domain.Module, lesson domain.Lesson) LessonContentPromptInput {
	return LessonContentPromptInput{
		Course: lessonContentCourse{
			ID:        course.ID.String(),
			Language:  string(course.Language),
			Title:     course.Title,
			Synopsis:  course.Synopsis,
			Goals:     course.Goals,
			LevelFrom: string(course.CurrentLevel),
			LevelTo:   string(course.TargetLevel),
		},
		Module: lessonContentModule{
			ID:                module.ID.String(),
			Order:             module.Order,
			Title:             module.Title,
			Description:       module.Description,
			KeyLearningPoints: module.KeyLearningPoints,
		},
		Lesson: lessonContentLesson{
			ID:                       lesson.ID.String(),
			Order:                    lesson.Order,
			Title:                    lesson.Title,
			Type:                     string(lesson.Type),
			EstimatedDurationMinutes: lesson.EstimatedDurationMinutes,
			LearningGoal:             lesson.LearningGoal,
			RequiresDiagram:          lesson.RequiresDiagram,
			TechnicalKeywords:        lesson.TechnicalKeywords,
		},
	}
}

type LessonContentResponse struct {
	ContentMarkdown string             `json:"contentMarkdown"`
	Exercises       []ExerciseResponse `json:"exercises"`
	Quizzes         []QuizResponse     `json:"quizzes"`
}

type ExerciseResponse struct {
	Type                 string                  `json:"type"`
	Difficulty           string                  `json:"difficulty"`
	Title                string                  `json:"title"`
	Objective            string                  `json:"objective"`
	InstructionsMarkdown string                  `json:"instructionsMarkdown"`
	ContentMarkdown      string                  `json:"contentMarkdown"`
	CorrectionMarkdown   string                  `json:"correctionMarkdown"`
	Payload              ExercisePayloadResponse `json:"payload"`
}

type ExercisePayloadResponse struct {
	Tasks          []string `json:"tasks"`
	Resources      []string `json:"resources"`
	StarterCode    *string  `json:"starterCode"`
	ExpectedOutput *string  `json:"expectedOutput"`
	Hints          []string `json:"hints"`
}

type QuizResponse struct {
	Type       string                 `json:"type"`
	Difficulty string                 `json:"difficulty"`
	Title      string                 `json:"title"`
	Objective  string                 `json:"objective"`
	Questions  []QuizQuestionResponse `json:"questions"`
}

type QuizQuestionResponse struct {
	Order      int                  `json:"order"`
	Type       string               `json:"type"`
	Question   string               `json:"question"`
	Options    []QuizOptionResponse `json:"options"`
	Answer     QuizAnswerResponse   `json:"answer"`
	Correction string               `json:"correction"`
}

type QuizOptionResponse struct {
	Order int    `json:"order"`
	Text  string `json:"text"`
}

type QuizAnswerResponse struct {
	Answer  *string  `json:"answer"`
	Answers []string `json:"answers"`
}

func (r LessonContentResponse) ExercisesToDomain(lessonID uuid.UUID) ([]domain.Exercise, error) {
	exercises := make([]domain.Exercise, 0, len(r.Exercises))
	for index, exercise := range r.Exercises {
		exerciseType, err := domain.ParseExerciseType(exercise.Type)
		if err != nil {
			return nil, fmt.Errorf("exercise %d type: %w", index+1, err)
		}
		difficulty, err := domain.ParseDifficulty(exercise.Difficulty)
		if err != nil {
			return nil, fmt.Errorf("exercise %d difficulty: %w", index+1, err)
		}

		domainExercise, err := domain.NewExercise(domain.NewExerciseParams{
			LessonID:             lessonID,
			Type:                 exerciseType,
			Difficulty:           difficulty,
			Title:                exercise.Title,
			Objective:            exercise.Objective,
			InstructionsMarkdown: exercise.InstructionsMarkdown,
			ContentMarkdown:      exercise.ContentMarkdown,
			CorrectionMarkdown:   exercise.CorrectionMarkdown,
			Payload:              exercise.Payload.ToDomain(),
		})
		if err != nil {
			return nil, fmt.Errorf("exercise %d: %w", index+1, err)
		}
		exercises = append(exercises, domainExercise)
	}
	return exercises, nil
}

func (r LessonContentResponse) QuizzesToDomain(lessonID uuid.UUID) ([]domain.Quiz, error) {
	quizzes := make([]domain.Quiz, 0, len(r.Quizzes))
	for index, quiz := range r.Quizzes {
		quizType, err := domain.ParseQuizType(quiz.Type)
		if err != nil {
			return nil, fmt.Errorf("quiz %d type: %w", index+1, err)
		}
		difficulty, err := domain.ParseDifficulty(quiz.Difficulty)
		if err != nil {
			return nil, fmt.Errorf("quiz %d difficulty: %w", index+1, err)
		}

		questions, err := quiz.QuestionsToDomain()
		if err != nil {
			return nil, fmt.Errorf("quiz %d questions: %w", index+1, err)
		}

		domainQuiz, err := domain.NewQuiz(domain.NewQuizParams{
			LessonID:   lessonID,
			Type:       quizType,
			Difficulty: difficulty,
			Title:      quiz.Title,
			Objective:  quiz.Objective,
			Questions:  questions,
		})
		if err != nil {
			return nil, fmt.Errorf("quiz %d: %w", index+1, err)
		}
		quizzes = append(quizzes, domainQuiz)
	}
	return quizzes, nil
}

func (p ExercisePayloadResponse) ToDomain() domain.ExercisePayload {
	return domain.ExercisePayload{
		"tasks":          p.Tasks,
		"resources":      p.Resources,
		"starterCode":    p.StarterCode,
		"expectedOutput": p.ExpectedOutput,
		"hints":          p.Hints,
	}
}

func (q QuizResponse) QuestionsToDomain() ([]domain.QuizQuestion, error) {
	questions := make([]domain.QuizQuestion, 0, len(q.Questions))
	for index, question := range q.Questions {
		questionType, err := domain.ParseQuizQuestionType(question.Type)
		if err != nil {
			return nil, fmt.Errorf("question %d type: %w", index+1, err)
		}

		questions = append(questions, domain.QuizQuestion{
			Order:      question.Order,
			Type:       questionType,
			Question:   question.Question,
			Options:    question.OptionsToDomain(),
			Answer:     question.Answer.ToDomain(),
			Correction: question.Correction,
		})
	}
	return questions, nil
}

func (q QuizQuestionResponse) OptionsToDomain() []domain.QuizOption {
	options := make([]domain.QuizOption, 0, len(q.Options))
	for _, option := range q.Options {
		options = append(options, domain.QuizOption{
			Order: option.Order,
			Text:  option.Text,
		})
	}
	return options
}

func (a QuizAnswerResponse) ToDomain() domain.QuizAnswer {
	return domain.QuizAnswer{
		Answer:  textutil.TrimmedPointerFrom(a.Answer),
		Answers: a.Answers,
	}
}
