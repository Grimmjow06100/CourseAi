package postgres

import (
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func ensureBulkInsertCount(entity string, expected int, inserted int64) error {
	if inserted == int64(expected) {
		return nil
	}
	return fmt.Errorf("bulk insert %s: inserted %d of %d rows", entity, inserted, expected)
}

func mapNoRows(err error, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return err
}

func mapUserWriteError(err error) error {
	if isUniqueViolation(err, "users_username_key") {
		return domain.ErrUsernameAlreadyExists
	}
	return err
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
