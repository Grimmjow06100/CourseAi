package postgres

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type CourseRepository struct {
	db DBTX
}

func NewCourseRepository(db DBTX) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) SaveCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	if err := course.Validate(); err != nil {
		return domain.Course{}, err
	}

	prerequisitesJSON, err := stringSliceJSON(course.Prerequisites)
	if err != nil {
		return domain.Course{}, err
	}
	goalsJSON, err := stringSliceJSON(course.Goals)
	if err != nil {
		return domain.Course{}, err
	}
	acquiredSkillsJSON, err := stringSliceJSON(course.AcquiredSkills)
	if err != nil {
		return domain.Course{}, err
	}
	finalProjectConstraintsJSON, err := stringSliceJSON(course.FinalProjectConstraints)
	if err != nil {
		return domain.Course{}, err
	}

	savedCourse, err := scanCourse(r.db.QueryRow(ctx, `
		INSERT INTO courses (
			id,
			request_id,
			language,
			status,
			initial_user_prompt,
			title,
			synopsis,
			target_audience,
			current_level,
			target_level,
			prerequisites,
			goals,
			acquired_skills,
			final_project_title,
			final_project_description,
			final_project_constraints,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15, $16::jsonb, $17, $18)
		RETURNING `+courseColumns+`
	`,
		course.ID,
		course.RequestID,
		string(course.Language),
		string(course.Status),
		course.InitialUserPrompt,
		course.Title,
		course.Synopsis,
		textValue(course.TargetAudience),
		string(course.CurrentLevel),
		string(course.TargetLevel),
		prerequisitesJSON,
		goalsJSON,
		acquiredSkillsJSON,
		textValue(course.FinalProjectTitle),
		textValue(course.FinalProjectDescription),
		finalProjectConstraintsJSON,
		course.CreatedAt,
		course.UpdatedAt,
	))
	if err != nil {
		return domain.Course{}, err
	}
	return savedCourse, nil
}

func (r *CourseRepository) UpdateCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	if err := course.Validate(); err != nil {
		return domain.Course{}, err
	}

	prerequisitesJSON, err := stringSliceJSON(course.Prerequisites)
	if err != nil {
		return domain.Course{}, err
	}
	goalsJSON, err := stringSliceJSON(course.Goals)
	if err != nil {
		return domain.Course{}, err
	}
	acquiredSkillsJSON, err := stringSliceJSON(course.AcquiredSkills)
	if err != nil {
		return domain.Course{}, err
	}
	finalProjectConstraintsJSON, err := stringSliceJSON(course.FinalProjectConstraints)
	if err != nil {
		return domain.Course{}, err
	}

	updatedCourse, err := scanCourse(r.db.QueryRow(ctx, `
		UPDATE courses
		SET
			request_id = $2,
			language = $3,
			status = $4,
			initial_user_prompt = $5,
			title = $6,
			synopsis = $7,
			target_audience = $8,
			current_level = $9,
			target_level = $10,
			prerequisites = $11::jsonb,
			goals = $12::jsonb,
			acquired_skills = $13::jsonb,
			final_project_title = $14,
			final_project_description = $15,
			final_project_constraints = $16::jsonb,
			updated_at = $17
		WHERE id = $1
		RETURNING `+courseColumns+`
	`,
		course.ID,
		course.RequestID,
		string(course.Language),
		string(course.Status),
		course.InitialUserPrompt,
		course.Title,
		course.Synopsis,
		textValue(course.TargetAudience),
		string(course.CurrentLevel),
		string(course.TargetLevel),
		prerequisitesJSON,
		goalsJSON,
		acquiredSkillsJSON,
		textValue(course.FinalProjectTitle),
		textValue(course.FinalProjectDescription),
		finalProjectConstraintsJSON,
		course.UpdatedAt,
	))
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	return r.hydrateCourse(ctx, updatedCourse)
}

func (r *CourseRepository) FindCourseByID(ctx context.Context, id uuid.UUID) (domain.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx, `
		SELECT `+courseColumns+`
		FROM courses
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	return r.hydrateCourse(ctx, course)
}

func (r *CourseRepository) FindCourseByRequestID(ctx context.Context, requestID uuid.UUID) (domain.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx, `
		SELECT `+courseColumns+`
		FROM courses
		WHERE request_id = $1
	`, requestID))
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	return r.hydrateCourse(ctx, course)
}

func (r *CourseRepository) ListCourses(ctx context.Context, filters contract.CourseFilters) (contract.Page[domain.Course], error) {
	filters = normalizeCourseFilters(filters)
	whereSQL, args := buildCourseWhere(filters)

	var totalItems int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM courses `+whereSQL, args...).Scan(&totalItems); err != nil {
		return contract.Page[domain.Course]{}, err
	}

	orderBy, err := courseOrderBySQL(filters.OrderBy)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}
	if err := filters.OrderDirection.Validate(); err != nil {
		return contract.Page[domain.Course]{}, err
	}

	pagination := filters.Pagination.Normalize()
	args = append(args, pagination.PageSize, (pagination.Page-1)*pagination.PageSize)
	rows, err := r.db.Query(ctx, `
		SELECT `+courseColumns+`
		FROM courses
		`+whereSQL+`
		ORDER BY `+orderBy+` `+string(filters.OrderDirection)+`
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args))+`
	`, args...)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}

	courses, err := scanCourses(rows)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}
	for index := range courses {
		hydratedCourse, err := r.hydrateCourse(ctx, courses[index])
		if err != nil {
			return contract.Page[domain.Course]{}, err
		}
		courses[index] = hydratedCourse
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pagination.PageSize)))
	}

	return contract.Page[domain.Course]{
		Items:       courses,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		HasNext:     pagination.Page < totalPages,
		HasPrevious: pagination.Page > 1,
	}, nil
}

func (r *CourseRepository) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *CourseRepository) hydrateCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	modules, err := NewModuleRepository(r.db).ListModulesByCourseID(ctx, course.ID)
	if err != nil {
		return domain.Course{}, err
	}
	course.Modules = modules
	if err := course.Validate(); err != nil {
		return domain.Course{}, err
	}
	return course, nil
}

func normalizeCourseFilters(filters contract.CourseFilters) contract.CourseFilters {
	filters.Pagination = filters.Pagination.Normalize()
	if filters.OrderBy == "" {
		filters.OrderBy = contract.CourseOrderByCreatedAt
	}
	if filters.OrderDirection == "" {
		filters.OrderDirection = contract.SortDescending
	}
	filters.Search = strings.TrimSpace(filters.Search)
	return filters
}

func buildCourseWhere(filters contract.CourseFilters) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	if filters.Status != nil {
		args = append(args, string(*filters.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if filters.Language != nil {
		args = append(args, string(*filters.Language))
		clauses = append(clauses, fmt.Sprintf("language = $%d", len(args)))
	}
	if filters.Search != "" {
		args = append(args, "%"+filters.Search+"%")
		clauses = append(clauses, fmt.Sprintf("(title ILIKE $%d OR synopsis ILIKE $%d)", len(args), len(args)))
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func courseOrderBySQL(field contract.CourseOrderField) (string, error) {
	switch field {
	case contract.CourseOrderByCreatedAt:
		return "created_at", nil
	case contract.CourseOrderByUpdatedAt:
		return "updated_at", nil
	case contract.CourseOrderByTitle:
		return "title", nil
	case contract.CourseOrderByStatus:
		return "status", nil
	default:
		return "", fmt.Errorf("invalid course order field: %s", field)
	}
}
