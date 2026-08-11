package postgres

import (
	"context"
	"encoding/json"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type LessonRepository struct {
	queries *dbsqlc.Queries
}

func NewLessonRepository(db DBTX) *LessonRepository {
	return &LessonRepository{queries: dbsqlc.New(db)}
}

func (r *LessonRepository) SaveLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	if err := lesson.Validate(); err != nil {
		return domain.Lesson{}, err
	}
	params, err := createLessonParams(lesson)
	if err != nil {
		return domain.Lesson{}, err
	}
	row, err := r.queries.CreateLesson(ctx, params)
	if err != nil {
		return domain.Lesson{}, err
	}
	savedLesson, err := lessonFromSQLC(row)
	if err != nil {
		return domain.Lesson{}, err
	}
	savedLesson.Exercises = lesson.Exercises
	savedLesson.Quizzes = lesson.Quizzes
	return savedLesson, nil
}

func (r *LessonRepository) SaveLessons(ctx context.Context, lessons []domain.Lesson) ([]domain.Lesson, error) {
	if len(lessons) == 0 {
		return []domain.Lesson{}, nil
	}

	params := make([]dbsqlc.CreateLessonsParams, 0, len(lessons))
	for _, lesson := range lessons {
		if err := lesson.Validate(); err != nil {
			return nil, err
		}
		createParams, err := createLessonParams(lesson)
		if err != nil {
			return nil, err
		}
		params = append(params, dbsqlc.CreateLessonsParams(createParams))
	}

	inserted, err := r.queries.CreateLessons(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := ensureBulkInsertCount("lessons", len(lessons), inserted); err != nil {
		return nil, err
	}
	return lessons, nil
}

func (r *LessonRepository) UpdateLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	if err := lesson.Validate(); err != nil {
		return domain.Lesson{}, err
	}
	params, err := updateLessonParams(lesson)
	if err != nil {
		return domain.Lesson{}, err
	}
	row, err := r.queries.UpdateLesson(ctx, params)
	if err != nil {
		return domain.Lesson{}, mapNoRows(err, ErrLessonNotFound)
	}
	updatedLesson, err := lessonFromSQLC(row)
	if err != nil {
		return domain.Lesson{}, err
	}
	updatedLesson.Exercises = lesson.Exercises
	updatedLesson.Quizzes = lesson.Quizzes
	return updatedLesson, nil
}

func (r *LessonRepository) ReplaceLessonContent(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	if err := lesson.Validate(); err != nil {
		return domain.Lesson{}, err
	}
	row, err := r.queries.ReplaceLessonContent(ctx, dbsqlc.ReplaceLessonContentParams{
		ID:               lesson.ID,
		ContentMarkdown:  lesson.ContentMarkdown,
		RawContentOutput: rawJSONFromBytes(lesson.RawContentOutput),
		UpdatedAt:        lesson.UpdatedAt,
	})
	if err != nil {
		return domain.Lesson{}, mapNoRows(err, ErrLessonNotFound)
	}
	updatedLesson, err := lessonFromSQLC(row)
	if err != nil {
		return domain.Lesson{}, err
	}
	updatedLesson.Exercises = lesson.Exercises
	updatedLesson.Quizzes = lesson.Quizzes
	return updatedLesson, nil
}

func (r *LessonRepository) FindLessonByID(ctx context.Context, id uuid.UUID) (domain.Lesson, error) {
	row, err := r.queries.GetLessonByID(ctx, dbsqlc.GetLessonByIDParams{ID: id})
	if err != nil {
		return domain.Lesson{}, mapNoRows(err, ErrLessonNotFound)
	}
	lesson, err := lessonFromSQLC(row)
	if err != nil {
		return domain.Lesson{}, err
	}
	return r.hydrateLesson(ctx, lesson)
}

func (r *LessonRepository) ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	rows, err := r.queries.ListLessonsByModuleID(ctx, dbsqlc.ListLessonsByModuleIDParams{ModuleID: moduleID})
	if err != nil {
		return nil, err
	}
	lessons, err := lessonsFromSQLC(rows)
	if err != nil {
		return nil, err
	}
	return relationLoader{queries: r.queries}.hydrateLessons(ctx, lessons)
}

func (r *LessonRepository) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteLessonByID(ctx, dbsqlc.DeleteLessonByIDParams{ID: id})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrLessonNotFound
	}
	return nil
}

func (r *LessonRepository) hydrateLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	lessons, err := relationLoader{queries: r.queries}.hydrateLessons(ctx, []domain.Lesson{lesson})
	if err != nil {
		return domain.Lesson{}, err
	}
	return lessons[0], nil
}

func createLessonParams(lesson domain.Lesson) (dbsqlc.CreateLessonParams, error) {
	keywords, err := stringSliceJSON(lesson.TechnicalKeywords)
	if err != nil {
		return dbsqlc.CreateLessonParams{}, err
	}
	return dbsqlc.CreateLessonParams{
		ID:                       lesson.ID,
		ModuleID:                 lesson.ModuleID,
		LessonOrder:              int32(lesson.Order),
		Title:                    lesson.Title,
		Type:                     dbsqlc.LessonType(lesson.Type),
		EstimatedDurationMinutes: int32(lesson.EstimatedDurationMinutes),
		LearningGoal:             lesson.LearningGoal,
		RequiresDiagram:          lesson.RequiresDiagram,
		TechnicalKeywords:        json.RawMessage(keywords),
		ContentMarkdown:          lesson.ContentMarkdown,
		RawContentOutput:         rawJSONFromBytes(lesson.RawContentOutput),
		CreatedAt:                lesson.CreatedAt,
		UpdatedAt:                lesson.UpdatedAt,
	}, nil
}

func updateLessonParams(lesson domain.Lesson) (dbsqlc.UpdateLessonParams, error) {
	keywords, err := stringSliceJSON(lesson.TechnicalKeywords)
	if err != nil {
		return dbsqlc.UpdateLessonParams{}, err
	}
	return dbsqlc.UpdateLessonParams{
		ModuleID:                 lesson.ModuleID,
		LessonOrder:              int32(lesson.Order),
		Title:                    lesson.Title,
		Type:                     dbsqlc.LessonType(lesson.Type),
		EstimatedDurationMinutes: int32(lesson.EstimatedDurationMinutes),
		LearningGoal:             lesson.LearningGoal,
		RequiresDiagram:          lesson.RequiresDiagram,
		TechnicalKeywords:        json.RawMessage(keywords),
		ContentMarkdown:          lesson.ContentMarkdown,
		RawContentOutput:         rawJSONFromBytes(lesson.RawContentOutput),
		UpdatedAt:                lesson.UpdatedAt,
		ID:                       lesson.ID,
	}, nil
}
