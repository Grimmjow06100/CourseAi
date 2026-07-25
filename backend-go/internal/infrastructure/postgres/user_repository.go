package postgres

import (
	"context"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

type UserRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) SaveUser(ctx context.Context, user domain.User) (domain.User, error) {
	if err := user.Validate(); err != nil {
		return domain.User{}, err
	}

	savedUser, err := scanUser(r.db.QueryRow(ctx, `
		INSERT INTO users (id, username, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns+`
	`, user.ID, string(user.Username), user.PasswordHash, user.CreatedAt, user.UpdatedAt))
	if err != nil {
		return domain.User{}, mapUserWriteError(err)
	}
	return savedUser, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.User{}, mapNoRows(err, domain.ErrUserNotFound)
	}
	return user, nil
}

func (r *UserRepository) FindUserByUsername(ctx context.Context, username domain.Username) (domain.User, error) {
	normalizedUsername := username.Normalize()
	user, err := scanUser(r.db.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE username = $1
	`, string(normalizedUsername)))
	if err != nil {
		return domain.User{}, mapNoRows(err, domain.ErrUserNotFound)
	}
	return user, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
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
