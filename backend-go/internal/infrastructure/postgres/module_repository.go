package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type ModuleRepository struct {
	db DBTX
}

func NewModuleRepository(db DBTX) *ModuleRepository {
	return &ModuleRepository{db: db}
}

func (r *ModuleRepository) SaveModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	if err := module.Validate(); err != nil {
		return domain.Module{}, err
	}

	pointsJSON, err := stringSliceJSON(module.KeyLearningPoints)
	if err != nil {
		return domain.Module{}, err
	}

	savedModule, err := scanModule(r.db.QueryRow(ctx, `
		INSERT INTO modules (
			id,
			course_id,
			module_order,
			title,
			description,
			key_learning_points,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		RETURNING `+moduleColumns+`
	`,
		module.ID,
		module.CourseID,
		module.Order,
		module.Title,
		module.Description,
		pointsJSON,
		module.CreatedAt,
		module.UpdatedAt,
	))
	if err != nil {
		return domain.Module{}, err
	}
	return r.hydrateModule(ctx, savedModule)
}

func (r *ModuleRepository) UpdateModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	if err := module.Validate(); err != nil {
		return domain.Module{}, err
	}

	pointsJSON, err := stringSliceJSON(module.KeyLearningPoints)
	if err != nil {
		return domain.Module{}, err
	}

	updatedModule, err := scanModule(r.db.QueryRow(ctx, `
		UPDATE modules
		SET
			course_id = $2,
			module_order = $3,
			title = $4,
			description = $5,
			key_learning_points = $6::jsonb,
			updated_at = $7
		WHERE id = $1
		RETURNING `+moduleColumns+`
	`,
		module.ID,
		module.CourseID,
		module.Order,
		module.Title,
		module.Description,
		pointsJSON,
		module.UpdatedAt,
	))
	if err != nil {
		return domain.Module{}, mapNoRows(err, ErrModuleNotFound)
	}
	return r.hydrateModule(ctx, updatedModule)
}

func (r *ModuleRepository) FindModuleByID(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	module, err := scanModule(r.db.QueryRow(ctx, `
		SELECT `+moduleColumns+`
		FROM modules
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.Module{}, mapNoRows(err, ErrModuleNotFound)
	}
	return r.hydrateModule(ctx, module)
}

func (r *ModuleRepository) ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+moduleColumns+`
		FROM modules
		WHERE course_id = $1
		ORDER BY module_order ASC
	`, courseID)
	if err != nil {
		return nil, err
	}

	modules, err := scanModules(rows)
	if err != nil {
		return nil, err
	}
	for index := range modules {
		hydratedModule, err := r.hydrateModule(ctx, modules[index])
		if err != nil {
			return nil, err
		}
		modules[index] = hydratedModule
	}
	return modules, nil
}

func (r *ModuleRepository) DeleteModule(ctx context.Context, id uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM modules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrModuleNotFound
	}
	return nil
}

func (r *ModuleRepository) hydrateModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	lessons, err := NewLessonRepository(r.db).ListLessonsByModuleID(ctx, module.ID)
	if err != nil {
		return domain.Module{}, err
	}
	module.Lessons = lessons
	if err := module.Validate(); err != nil {
		return domain.Module{}, err
	}
	return module, nil
}
