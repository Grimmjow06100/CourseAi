package postgres

import (
	"context"
	"encoding/json"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type QuizRepository struct {
	queries *dbsqlc.Queries
}

func NewQuizRepository(db DBTX) *QuizRepository {
	return &QuizRepository{queries: dbsqlc.New(db)}
}

func (r *QuizRepository) SaveQuiz(ctx context.Context, quiz domain.Quiz) (domain.Quiz, error) {
	if err := quiz.Validate(); err != nil {
		return domain.Quiz{}, err
	}
	params, err := createQuizParams(quiz)
	if err != nil {
		return domain.Quiz{}, err
	}
	row, err := r.queries.CreateLessonQuiz(ctx, params)
	if err != nil {
		return domain.Quiz{}, err
	}
	return quizFromSQLC(row)
}

func (r *QuizRepository) SaveQuizzes(ctx context.Context, quizzes []domain.Quiz) ([]domain.Quiz, error) {
	if len(quizzes) == 0 {
		return []domain.Quiz{}, nil
	}

	params := make([]dbsqlc.CreateLessonQuizzesParams, 0, len(quizzes))
	for _, quiz := range quizzes {
		if err := quiz.Validate(); err != nil {
			return nil, err
		}
		createParams, err := createQuizParams(quiz)
		if err != nil {
			return nil, err
		}
		params = append(params, dbsqlc.CreateLessonQuizzesParams(createParams))
	}

	inserted, err := r.queries.CreateLessonQuizzes(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := ensureBulkInsertCount("lesson quizzes", len(quizzes), inserted); err != nil {
		return nil, err
	}
	return quizzes, nil
}

func (r *QuizRepository) ListQuizzesByLessonID(ctx context.Context, lessonID uuid.UUID) ([]domain.Quiz, error) {
	rows, err := r.queries.ListLessonQuizzesByLessonID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	return quizzesFromSQLC(rows)
}

func (r *QuizRepository) DeleteQuizzesByLessonID(ctx context.Context, lessonID uuid.UUID) error {
	return r.queries.DeleteLessonQuizzesByLessonID(ctx, lessonID)
}

func createQuizParams(quiz domain.Quiz) (dbsqlc.CreateLessonQuizParams, error) {
	questions, err := quizQuestionsJSON(quiz.Questions)
	if err != nil {
		return dbsqlc.CreateLessonQuizParams{}, err
	}
	return dbsqlc.CreateLessonQuizParams{
		ID:          quiz.ID,
		LessonID:    quiz.LessonID,
		Type:        dbsqlc.QuizType(quiz.Type),
		Difficulty:  dbsqlc.ActivityDifficulty(quiz.Difficulty),
		Title:       quiz.Title,
		Objective:   quiz.Objective,
		Questions:   json.RawMessage(questions),
		RawAiOutput: rawJSONFromBytes(quiz.RawAIOutput),
		CreatedAt:   quiz.CreatedAt,
		UpdatedAt:   quiz.UpdatedAt,
	}, nil
}
