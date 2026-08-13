package postgres

import (
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresErrorMapping(t *testing.T) {
	t.Parallel()

	if err := ensureBulkInsertCount("lessons", 2, 2); err != nil {
		t.Fatalf("matching bulk count rejected: %v", err)
	}
	if err := ensureBulkInsertCount("lessons", 2, 1); err == nil {
		t.Fatal("expected mismatched bulk count error")
	}
	if got := mapNoRows(pgx.ErrNoRows, contract.ErrCourseNotFound); !errors.Is(got, contract.ErrCourseNotFound) {
		t.Fatalf("mapNoRows() = %v", got)
	}
	cause := errors.New("query failed")
	if got := mapNoRows(cause, contract.ErrCourseNotFound); !errors.Is(got, cause) {
		t.Fatalf("non-no-rows error changed: %v", got)
	}

	unique := &pgconn.PgError{Code: "23505", ConstraintName: "users_username_key"}
	if !isUniqueViolation(unique, "users_username_key") || isUniqueViolation(unique, "other") {
		t.Fatal("unique violation constraint matching is incorrect")
	}
	if got := mapUserWriteError(unique); !errors.Is(got, domain.ErrUsernameAlreadyExists) {
		t.Fatalf("mapUserWriteError() = %v", got)
	}
	if isUniqueViolation(cause, "") {
		t.Fatal("ordinary errors must not be unique violations")
	}
}

func TestNormalizeCourseFilters(t *testing.T) {
	t.Parallel()

	filters := normalizeCourseFilters(contract.CourseFilters{Search: "  linux  "})
	if filters.Search != "linux" || filters.OrderBy != contract.CourseOrderByCreatedAt ||
		filters.OrderDirection != contract.SortDescending || filters.Pagination.Page != 1 || filters.Pagination.PageSize != 20 {
		t.Fatalf("unexpected normalized filters: %+v", filters)
	}
	if optionalSearch("") != nil {
		t.Fatal("empty search should map to nil")
	}
	if value := optionalSearch("linux"); value == nil || *value != "linux" {
		t.Fatalf("optionalSearch() = %v", value)
	}
}

func TestNewRepositoriesExposesAllImplementations(t *testing.T) {
	t.Parallel()

	repositories := NewRepositories(&copyFromRecorder{})
	if repositories.Users() == nil || repositories.GenerationRequests() == nil || repositories.GenerationJobs() == nil ||
		repositories.Courses() == nil || repositories.Modules() == nil || repositories.Lessons() == nil ||
		repositories.Exercises() == nil || repositories.Quizzes() == nil {
		t.Fatal("repository registry contains a nil implementation")
	}
}
