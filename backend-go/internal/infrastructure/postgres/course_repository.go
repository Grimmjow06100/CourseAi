package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type CourseRepository struct {
	queries *dbsqlc.Queries
}

func NewCourseRepository(db DBTX) *CourseRepository {
	return &CourseRepository{queries: dbsqlc.New(db)}
}

func (r *CourseRepository) SaveCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	if err := course.ValidateCourseOnly(); err != nil {
		return domain.Course{}, err
	}

	params, err := createCourseParams(course)
	if err != nil {
		return domain.Course{}, err
	}
	row, err := r.queries.CreateCourse(ctx, params)
	if err != nil {
		return domain.Course{}, err
	}
	savedCourse, err := courseFromSQLC(row)
	if err != nil {
		return domain.Course{}, err
	}
	savedCourse.Modules = course.Modules
	return savedCourse, nil
}

func (r *CourseRepository) UpdateCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	if err := course.ValidateCourseOnly(); err != nil {
		return domain.Course{}, err
	}

	params, err := updateCourseParams(course)
	if err != nil {
		return domain.Course{}, err
	}
	row, err := r.queries.UpdateCourse(ctx, params)
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	updatedCourse, err := courseFromSQLC(row)
	if err != nil {
		return domain.Course{}, err
	}
	updatedCourse.Modules = course.Modules
	return updatedCourse, nil
}

func (r *CourseRepository) FindCourseByID(ctx context.Context, id uuid.UUID) (domain.Course, error) {
	course, err := r.FindCourseStateByID(ctx, id)
	if err != nil {
		return domain.Course{}, err
	}
	return r.hydrateCourse(ctx, course)
}

func (r *CourseRepository) FindCourseStateByID(ctx context.Context, id uuid.UUID) (domain.Course, error) {
	row, err := r.queries.GetCourseByID(ctx, id)
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	return courseFromSQLC(row)
}

func (r *CourseRepository) FindCourseByRequestID(ctx context.Context, requestID uuid.UUID) (domain.Course, error) {
	course, err := r.FindCourseStateByRequestID(ctx, requestID)
	if err != nil {
		return domain.Course{}, err
	}
	return r.hydrateCourse(ctx, course)
}

func (r *CourseRepository) FindCourseStateByRequestID(ctx context.Context, requestID uuid.UUID) (domain.Course, error) {
	row, err := r.queries.GetCourseByRequestID(ctx, requestID)
	if err != nil {
		return domain.Course{}, mapNoRows(err, ErrCourseNotFound)
	}
	return courseFromSQLC(row)
}

func (r *CourseRepository) IsCourseContentComplete(ctx context.Context, id uuid.UUID) (bool, error) {
	isComplete, err := r.queries.IsCourseContentComplete(ctx, id)
	if err != nil {
		return false, err
	}
	if isComplete == nil {
		return false, fmt.Errorf("course content completeness query returned null")
	}
	return *isComplete, nil
}

func (r *CourseRepository) ListCourses(ctx context.Context, filters contract.CourseFilters) (contract.Page[domain.Course], error) {
	filters = normalizeCourseFilters(filters)
	if err := filters.OrderDirection.Validate(); err != nil {
		return contract.Page[domain.Course]{}, err
	}

	search := optionalSearch(filters.Search)
	totalItems, err := r.queries.CountCourses(ctx, dbsqlc.CountCoursesParams{
		Status:   sqlcStatusPtr(filters.Status),
		Language: sqlcLanguagePtr(filters.Language),
		Search:   search,
	})
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}

	pagination := filters.Pagination.Normalize()
	rows, err := r.listCourseRows(ctx, filters, search, pagination)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}
	courses, err := coursesFromSQLC(rows)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}
	courses, err = relationLoader{queries: r.queries}.hydrateCourses(ctx, courses)
	if err != nil {
		return contract.Page[domain.Course]{}, err
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(pagination.PageSize)))
	}
	return contract.Page[domain.Course]{
		Items:       courses,
		Page:        pagination.Page,
		PageSize:    pagination.PageSize,
		TotalItems:  int(totalItems),
		TotalPages:  totalPages,
		HasNext:     pagination.Page < totalPages,
		HasPrevious: pagination.Page > 1,
	}, nil
}

func (r *CourseRepository) DeleteCourseByRequestID(ctx context.Context, requestID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteCourseByRequestID(ctx, requestID)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *CourseRepository) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteCourseByID(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *CourseRepository) hydrateCourse(ctx context.Context, course domain.Course) (domain.Course, error) {
	courses, err := relationLoader{queries: r.queries}.hydrateCourses(ctx, []domain.Course{course})
	if err != nil {
		return domain.Course{}, err
	}
	return courses[0], nil
}

func (r *CourseRepository) listCourseRows(
	ctx context.Context,
	filters contract.CourseFilters,
	search *string,
	pagination contract.Pagination,
) ([]dbsqlc.Course, error) {
	offset := int32((pagination.Page - 1) * pagination.PageSize)
	limit := int32(pagination.PageSize)
	status := sqlcStatusPtr(filters.Status)
	language := sqlcLanguagePtr(filters.Language)

	switch filters.OrderBy {
	case contract.CourseOrderByCreatedAt:
		if filters.OrderDirection == contract.SortAscending {
			return r.queries.ListCoursesCreatedAtAsc(ctx, dbsqlc.ListCoursesCreatedAtAscParams{
				Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
			})
		}
		return r.queries.ListCoursesCreatedAtDesc(ctx, dbsqlc.ListCoursesCreatedAtDescParams{
			Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
		})
	case contract.CourseOrderByUpdatedAt:
		if filters.OrderDirection == contract.SortAscending {
			return r.queries.ListCoursesUpdatedAtAsc(ctx, dbsqlc.ListCoursesUpdatedAtAscParams{
				Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
			})
		}
		return r.queries.ListCoursesUpdatedAtDesc(ctx, dbsqlc.ListCoursesUpdatedAtDescParams{
			Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
		})
	case contract.CourseOrderByTitle:
		if filters.OrderDirection == contract.SortAscending {
			return r.queries.ListCoursesTitleAsc(ctx, dbsqlc.ListCoursesTitleAscParams{
				Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
			})
		}
		return r.queries.ListCoursesTitleDesc(ctx, dbsqlc.ListCoursesTitleDescParams{
			Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
		})
	case contract.CourseOrderByStatus:
		if filters.OrderDirection == contract.SortAscending {
			return r.queries.ListCoursesStatusAsc(ctx, dbsqlc.ListCoursesStatusAscParams{
				Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
			})
		}
		return r.queries.ListCoursesStatusDesc(ctx, dbsqlc.ListCoursesStatusDescParams{
			Status: status, Language: language, Search: search, OffsetRows: offset, LimitRows: limit,
		})
	default:
		return nil, fmt.Errorf("invalid course order field: %s", filters.OrderBy)
	}
}

func createCourseParams(course domain.Course) (dbsqlc.CreateCourseParams, error) {
	prerequisites, goals, skills, constraints, err := courseJSONFields(course)
	if err != nil {
		return dbsqlc.CreateCourseParams{}, err
	}
	return dbsqlc.CreateCourseParams{
		ID:                      course.ID,
		RequestID:               course.RequestID,
		Language:                dbsqlc.CourseLanguage(course.Language),
		Status:                  dbsqlc.CourseGenerationStatus(course.Status),
		InitialUserPrompt:       course.InitialUserPrompt,
		Title:                   course.Title,
		Synopsis:                course.Synopsis,
		TargetAudience:          course.TargetAudience,
		CurrentLevel:            dbsqlc.Level(course.CurrentLevel),
		TargetLevel:             dbsqlc.Level(course.TargetLevel),
		Prerequisites:           prerequisites,
		Goals:                   goals,
		AcquiredSkills:          skills,
		FinalProjectTitle:       course.FinalProjectTitle,
		FinalProjectDescription: course.FinalProjectDescription,
		FinalProjectConstraints: constraints,
		RawArchitectureOutput:   rawJSONFromBytes(course.RawArchitectureOutput),
		CreatedAt:               course.CreatedAt,
		UpdatedAt:               course.UpdatedAt,
	}, nil
}

func updateCourseParams(course domain.Course) (dbsqlc.UpdateCourseParams, error) {
	prerequisites, goals, skills, constraints, err := courseJSONFields(course)
	if err != nil {
		return dbsqlc.UpdateCourseParams{}, err
	}
	return dbsqlc.UpdateCourseParams{
		RequestID:               course.RequestID,
		Language:                dbsqlc.CourseLanguage(course.Language),
		Status:                  dbsqlc.CourseGenerationStatus(course.Status),
		InitialUserPrompt:       course.InitialUserPrompt,
		Title:                   course.Title,
		Synopsis:                course.Synopsis,
		TargetAudience:          course.TargetAudience,
		CurrentLevel:            dbsqlc.Level(course.CurrentLevel),
		TargetLevel:             dbsqlc.Level(course.TargetLevel),
		Prerequisites:           prerequisites,
		Goals:                   goals,
		AcquiredSkills:          skills,
		FinalProjectTitle:       course.FinalProjectTitle,
		FinalProjectDescription: course.FinalProjectDescription,
		FinalProjectConstraints: constraints,
		RawArchitectureOutput:   rawJSONFromBytes(course.RawArchitectureOutput),
		UpdatedAt:               course.UpdatedAt,
		ID:                      course.ID,
	}, nil
}

func courseJSONFields(course domain.Course) (json.RawMessage, json.RawMessage, json.RawMessage, json.RawMessage, error) {
	prerequisites, err := stringSliceJSON(course.Prerequisites)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	goals, err := stringSliceJSON(course.Goals)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	skills, err := stringSliceJSON(course.AcquiredSkills)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	constraints, err := stringSliceJSON(course.FinalProjectConstraints)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return json.RawMessage(prerequisites), json.RawMessage(goals), json.RawMessage(skills), json.RawMessage(constraints), nil
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

func optionalSearch(search string) *string {
	if search == "" {
		return nil
	}
	return &search
}
