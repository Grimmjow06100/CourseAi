package postgres

import (
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/shared/errtrace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func ensureBulkInsertCount(entity string, expected int, inserted int64) error {
	if inserted == int64(expected) {
		return nil
	}
	return errtrace.Capture(fmt.Errorf("bulk insert %s: inserted %d of %d rows", entity, inserted, expected))
}

func mapNoRows(err error, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errtrace.Capture(notFound)
	}
	return errtrace.Capture(err)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && (constraint == "" || pgErr.ConstraintName == constraint)
}

var (
	ErrCourseNotFound            = contract.ErrCourseNotFound
	ErrGenerationRequestNotFound = contract.ErrGenerationRequestNotFound
	ErrGenerationJobNotFound     = contract.ErrGenerationJobNotFound
	ErrLessonNotFound            = contract.ErrLessonNotFound
	ErrModuleNotFound            = contract.ErrModuleNotFound
)
