package postgres

import (
	"context"
	"encoding/json"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type ExerciseRepository struct {
	queries *dbsqlc.Queries
}

func NewExerciseRepository(db DBTX) *ExerciseRepository {
	return &ExerciseRepository{queries: dbsqlc.New(db)}
}

func (r *ExerciseRepository) SaveExercise(ctx context.Context, exercise domain.Exercise) (domain.Exercise, error) {
	if err := exercise.Validate(); err != nil {
		return domain.Exercise{}, err
	}
	params, err := createExerciseParams(exercise)
	if err != nil {
		return domain.Exercise{}, err
	}
	row, err := r.queries.CreateLessonExercise(ctx, params)
	if err != nil {
		return domain.Exercise{}, err
	}
	return exerciseFromSQLC(row)
}

func (r *ExerciseRepository) SaveExercises(ctx context.Context, exercises []domain.Exercise) ([]domain.Exercise, error) {
	if len(exercises) == 0 {
		return []domain.Exercise{}, nil
	}

	params := make([]dbsqlc.CreateLessonExercisesParams, 0, len(exercises))
	for _, exercise := range exercises {
		if err := exercise.Validate(); err != nil {
			return nil, err
		}
		createParams, err := createExerciseParams(exercise)
		if err != nil {
			return nil, err
		}
		params = append(params, dbsqlc.CreateLessonExercisesParams(createParams))
	}

	inserted, err := r.queries.CreateLessonExercises(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := ensureBulkInsertCount("lesson exercises", len(exercises), inserted); err != nil {
		return nil, err
	}
	return exercises, nil
}

func (r *ExerciseRepository) ListExercisesByLessonID(ctx context.Context, lessonID uuid.UUID) ([]domain.Exercise, error) {
	rows, err := r.queries.ListLessonExercisesByLessonID(ctx, dbsqlc.ListLessonExercisesByLessonIDParams{LessonID: lessonID})
	if err != nil {
		return nil, err
	}
	return exercisesFromSQLC(rows)
}

func (r *ExerciseRepository) DeleteExercisesByLessonID(ctx context.Context, lessonID uuid.UUID) error {
	return r.queries.DeleteLessonExercisesByLessonID(ctx, dbsqlc.DeleteLessonExercisesByLessonIDParams{LessonID: lessonID})
}

func createExerciseParams(exercise domain.Exercise) (dbsqlc.CreateLessonExerciseParams, error) {
	payload, err := exercisePayloadJSON(exercise.Payload)
	if err != nil {
		return dbsqlc.CreateLessonExerciseParams{}, err
	}
	return dbsqlc.CreateLessonExerciseParams{
		ID:                   exercise.ID,
		LessonID:             exercise.LessonID,
		Type:                 dbsqlc.ExerciseType(exercise.Type),
		Difficulty:           dbsqlc.ActivityDifficulty(exercise.Difficulty),
		Title:                exercise.Title,
		Objective:            exercise.Objective,
		InstructionsMarkdown: exercise.InstructionsMarkdown,
		ContentMarkdown:      exercise.ContentMarkdown,
		CorrectionMarkdown:   exercise.CorrectionMarkdown,
		Payload:              json.RawMessage(payload),
		RawAiOutput:          rawJSONFromBytes(exercise.RawAIOutput),
		CreatedAt:            exercise.CreatedAt,
		UpdatedAt:            exercise.UpdatedAt,
	}, nil
}
