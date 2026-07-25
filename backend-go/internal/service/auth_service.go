package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
)

var ErrAuthentification = errors.New("nom d'utilisateur ou mot de passe invalide")

type ErrFailedSignup struct {
	Message string
	Err     error
}

func (e ErrFailedSignup) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e ErrFailedSignup) Unwrap() error {
	return e.Err
}

type SignupParams = contract.SignupParams

type AuthService struct {
	token    contract.TokenManager
	users    contract.UserRepository
	password contract.PasswordManager
}

func NewAuthService(token contract.TokenManager, users contract.UserRepository, password contract.PasswordManager) *AuthService {
	return &AuthService{
		token:    token,
		users:    users,
		password: password,
	}
}

func (a *AuthService) Login(ctx context.Context, username string, password string) (string, error) {
	normalizedUsername := domain.Username(username).Normalize()

	user, err := a.users.FindUserByUsername(ctx, normalizedUsername)
	if err != nil {
		return "", ErrAuthentification
	}

	if !a.password.CheckPassword(password, user.PasswordHash) {
		return "", ErrAuthentification
	}

	token, err := a.token.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate login token: %w", err)
	}

	return token, nil
}

func (a *AuthService) SignUp(ctx context.Context, params contract.SignupParams) (string, error) {
	password := domain.Password(params.Password)
	if err := password.Validate(); err != nil {
		return "", err
	}

	passwordHash, err := a.password.HashPassword(params.Password)
	if err != nil {
		return "", fmt.Errorf("hash signup password: %w", err)
	}

	newUser, err := domain.NewUser(domain.Username(params.Username), passwordHash)
	if err != nil {
		return "", err
	}

	persistedUser, err := a.users.SaveUser(ctx, newUser)
	if err != nil {
		if errors.Is(err, domain.ErrUsernameAlreadyExists) {
			return "", domain.ErrUsernameAlreadyExists
		}

		return "", ErrFailedSignup{
			Message: "signup a échoué",
			Err:     err,
		}
	}

	token, err := a.token.GenerateToken(persistedUser.ID)
	if err != nil {
		return "", fmt.Errorf("generate signup token: %w", err)
	}

	return token, nil
}
