package postgres

import (
	"errors"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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
	ErrLessonNotFound            = contract.ErrLessonNotFound
	ErrModuleNotFound            = contract.ErrModuleNotFound
)
