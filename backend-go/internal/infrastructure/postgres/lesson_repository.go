package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type LessonRepository struct {
	db DBTX
}

func NewLessonRepository(db DBTX) *LessonRepository {
	return &LessonRepository{db: db}
}

func (r *LessonRepository) SaveLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	if err := lesson.Validate(); err != nil {
		return domain.Lesson{}, err
	}

	keywordsJSON, err := stringSliceJSON(lesson.TechnicalKeywords)
	if err != nil {
		return domain.Lesson{}, err
	}

	savedLesson, err := scanLesson(r.db.QueryRow(ctx, `
		INSERT INTO lessons (
			id,
			module_id,
			lesson_order,
			title,
			type,
			estimated_duration_minutes,
			learning_goal,
			requires_diagram,
			technical_keywords,
			content_markdown,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12)
		RETURNING `+lessonColumns+`
	`,
		lesson.ID,
		lesson.ModuleID,
		lesson.Order,
		lesson.Title,
		string(lesson.Type),
		lesson.EstimatedDurationMinutes,
		lesson.LearningGoal,
		lesson.RequiresDiagram,
		keywordsJSON,
		textValue(lesson.ContentMarkdown),
		lesson.CreatedAt,
		lesson.UpdatedAt,
	))
	if err != nil {
		return domain.Lesson{}, err
	}
	return savedLesson, nil
}

func (r *LessonRepository) SaveLessons(ctx context.Context, lessons []domain.Lesson) ([]domain.Lesson, error) {
	savedLessons := make([]domain.Lesson, 0, len(lessons))
	for _, lesson := range lessons {
		savedLesson, err := r.SaveLesson(ctx, lesson)
		if err != nil {
			return nil, err
		}
		savedLessons = append(savedLessons, savedLesson)
	}
	return savedLessons, nil
}

func (r *LessonRepository) UpdateLesson(ctx context.Context, lesson domain.Lesson) (domain.Lesson, error) {
	if err := lesson.Validate(); err != nil {
		return domain.Lesson{}, err
	}

	keywordsJSON, err := stringSliceJSON(lesson.TechnicalKeywords)
	if err != nil {
		return domain.Lesson{}, err
	}

	updatedLesson, err := scanLesson(r.db.QueryRow(ctx, `
		UPDATE lessons
		SET
			module_id = $2,
			lesson_order = $3,
			title = $4,
			type = $5,
			estimated_duration_minutes = $6,
			learning_goal = $7,
			requires_diagram = $8,
			technical_keywords = $9::jsonb,
			content_markdown = $10,
			updated_at = $11
		WHERE id = $1
		RETURNING `+lessonColumns+`
	`,
		lesson.ID,
		lesson.ModuleID,
		lesson.Order,
		lesson.Title,
		string(lesson.Type),
		lesson.EstimatedDurationMinutes,
		lesson.LearningGoal,
		lesson.RequiresDiagram,
		keywordsJSON,
		textValue(lesson.ContentMarkdown),
		lesson.UpdatedAt,
	))
	if err != nil {
		return domain.Lesson{}, mapNoRows(err, ErrLessonNotFound)
	}
	return updatedLesson, nil
}

func (r *LessonRepository) FindLessonByID(ctx context.Context, id uuid.UUID) (domain.Lesson, error) {
	lesson, err := scanLesson(r.db.QueryRow(ctx, `
		SELECT `+lessonColumns+`
		FROM lessons
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.Lesson{}, mapNoRows(err, ErrLessonNotFound)
	}
	return lesson, nil
}

func (r *LessonRepository) ListLessonsByModuleID(ctx context.Context, moduleID uuid.UUID) ([]domain.Lesson, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+lessonColumns+`
		FROM lessons
		WHERE module_id = $1
		ORDER BY lesson_order ASC
	`, moduleID)
	if err != nil {
		return nil, err
	}
	return scanLessons(rows)
}

func (r *LessonRepository) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM lessons WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrLessonNotFound
	}
	return nil
}
