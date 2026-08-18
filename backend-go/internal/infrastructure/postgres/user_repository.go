package postgres

import (
	"context"

	dbsqlc "github.com/Grimmjow06100/course-ai/backend-go/internal/db/sqlc"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type UserRepository struct {
	queries *dbsqlc.Queries
}

func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{queries: dbsqlc.New(db)}
}

func (r *UserRepository) SaveUser(ctx context.Context, user domain.User) (domain.User, error) {
	if err := user.Validate(); err != nil {
		return domain.User{}, err
	}

	row, err := r.queries.CreateUser(ctx, dbsqlc.CreateUserParams{
		ID:        user.ID,
		Username:  string(user.Username),
		Password:  user.PasswordHash,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
	if err != nil {
		return domain.User{}, mapUserWriteError(err)
	}
	return userFromSQLC(row)
}

func (r *UserRepository) FindUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, mapNoRows(err, domain.ErrUserNotFound)
	}
	return userFromSQLC(row)
}

func (r *UserRepository) FindUserByUsername(ctx context.Context, username domain.Username) (domain.User, error) {
	row, err := r.queries.GetUserByUsername(ctx, string(username.Normalize()))
	if err != nil {
		return domain.User{}, mapNoRows(err, domain.ErrUserNotFound)
	}
	return userFromSQLC(row)
}

func (r *UserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteUserByID(ctx, id)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) PersistUser(ctx context.Context, user domain.User) (domain.User, error) {
	return r.SaveUser(ctx, user)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.FindUserByUsername(ctx, domain.Username(email))
}
