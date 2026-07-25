package contract

import (
	"context"

	"github.com/google/uuid"
)

type SignupParams struct {
	Username string
	Password string
}

type AuthService interface {
	Login(ctx context.Context, username string, password string) (string, error)
	SignUp(ctx context.Context, params SignupParams) (string, error)
}

type TokenManager interface {
	GenerateToken(userID uuid.UUID) (string, error)
	VerifyToken(token string) (uuid.UUID, error)
}

type PasswordManager interface {
	CheckPassword(password string, hash string) bool
	HashPassword(password string) (string, error)
}
