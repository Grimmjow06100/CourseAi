package postgres

import (
	"context"
	"encoding/json"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type ModuleRepository struct {
	queries *dbsqlc.Queries
}

func NewModuleRepository(db DBTX) *ModuleRepository {
	return &ModuleRepository{queries: dbsqlc.New(db)}
}

func (r *ModuleRepository) SaveModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	if err := module.Validate(); err != nil {
		return domain.Module{}, err
	}
	params, err := createModuleParams(module)
	if err != nil {
		return domain.Module{}, err
	}
	row, err := r.queries.CreateModule(ctx, params)
	if err != nil {
		return domain.Module{}, err
	}
	savedModule, err := moduleFromSQLC(row)
	if err != nil {
		return domain.Module{}, err
	}
	savedModule.Lessons = module.Lessons
	return savedModule, nil
}

func (r *ModuleRepository) SaveModules(ctx context.Context, modules []domain.Module) ([]domain.Module, error) {
	if len(modules) == 0 {
		return []domain.Module{}, nil
	}

	params := make([]dbsqlc.CreateModulesParams, 0, len(modules))
	for _, module := range modules {
		if err := module.Validate(); err != nil {
			return nil, err
		}
		createParams, err := createModuleParams(module)
		if err != nil {
			return nil, err
		}
		params = append(params, dbsqlc.CreateModulesParams(createParams))
	}

	inserted, err := r.queries.CreateModules(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := ensureBulkInsertCount("modules", len(modules), inserted); err != nil {
		return nil, err
	}
	return modules, nil
}

func (r *ModuleRepository) UpdateModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	if err := module.Validate(); err != nil {
		return domain.Module{}, err
	}
	params, err := updateModuleParams(module)
	if err != nil {
		return domain.Module{}, err
	}
	row, err := r.queries.UpdateModule(ctx, params)
	if err != nil {
		return domain.Module{}, mapNoRows(err, ErrModuleNotFound)
	}
	updatedModule, err := moduleFromSQLC(row)
	if err != nil {
		return domain.Module{}, err
	}
	updatedModule.Lessons = module.Lessons
	return updatedModule, nil
}

func (r *ModuleRepository) FindModuleByID(ctx context.Context, id uuid.UUID) (domain.Module, error) {
	row, err := r.queries.GetModuleByID(ctx, dbsqlc.GetModuleByIDParams{ID: id})
	if err != nil {
		return domain.Module{}, mapNoRows(err, ErrModuleNotFound)
	}
	module, err := moduleFromSQLC(row)
	if err != nil {
		return domain.Module{}, err
	}
	return r.hydrateModule(ctx, module)
}

func (r *ModuleRepository) ListModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]domain.Module, error) {
	rows, err := r.queries.ListModulesByCourseID(ctx, dbsqlc.ListModulesByCourseIDParams{CourseID: courseID})
	if err != nil {
		return nil, err
	}
	modules, err := modulesFromSQLC(rows)
	if err != nil {
		return nil, err
	}
	return relationLoader{queries: r.queries}.hydrateModules(ctx, modules)
}

func (r *ModuleRepository) DeleteModule(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteModuleByID(ctx, dbsqlc.DeleteModuleByIDParams{ID: id})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrModuleNotFound
	}
	return nil
}

func (r *ModuleRepository) hydrateModule(ctx context.Context, module domain.Module) (domain.Module, error) {
	modules, err := relationLoader{queries: r.queries}.hydrateModules(ctx, []domain.Module{module})
	if err != nil {
		return domain.Module{}, err
	}
	return modules[0], nil
}

func createModuleParams(module domain.Module) (dbsqlc.CreateModuleParams, error) {
	points, err := stringSliceJSON(module.KeyLearningPoints)
	if err != nil {
		return dbsqlc.CreateModuleParams{}, err
	}
	return dbsqlc.CreateModuleParams{
		ID:                   module.ID,
		CourseID:             module.CourseID,
		ModuleOrder:          int32(module.Order),
		Title:                module.Title,
		Description:          module.Description,
		KeyLearningPoints:    json.RawMessage(points),
		RawLessonsPlanOutput: rawJSONFromBytes(module.RawLessonsPlanOutput),
		CreatedAt:            module.CreatedAt,
		UpdatedAt:            module.UpdatedAt,
	}, nil
}

func updateModuleParams(module domain.Module) (dbsqlc.UpdateModuleParams, error) {
	points, err := stringSliceJSON(module.KeyLearningPoints)
	if err != nil {
		return dbsqlc.UpdateModuleParams{}, err
	}
	return dbsqlc.UpdateModuleParams{
		CourseID:             module.CourseID,
		ModuleOrder:          int32(module.Order),
		Title:                module.Title,
		Description:          module.Description,
		KeyLearningPoints:    json.RawMessage(points),
		RawLessonsPlanOutput: rawJSONFromBytes(module.RawLessonsPlanOutput),
		UpdatedAt:            module.UpdatedAt,
		ID:                   module.ID,
	}, nil
}
