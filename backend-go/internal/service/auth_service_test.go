package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/contract"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/domain"
	"github.com/google/uuid"
)

func TestAuthServiceLogin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	users := &authUserRepository{findUser: domain.User{ID: userID, Username: "Alice", PasswordHash: "hash"}}
	tokens := &authTokenManager{token: "jwt-token"}
	passwords := &authPasswordManager{passwordMatches: true}
	service := NewAuthService(tokens, users, passwords)

	token, err := service.Login(context.Background(), "  Alice  Martin ", "Strong!1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token != "jwt-token" || users.findUsername != domain.Username("Alice Martin") || tokens.generatedFor != userID {
		t.Fatalf("unexpected login result: token=%q username=%q userID=%s", token, users.findUsername, tokens.generatedFor)
	}
}

func TestAuthServiceLoginDoesNotLeakCredentialFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		users     *authUserRepository
		passwords *authPasswordManager
	}{
		{name: "unknown user", users: &authUserRepository{findErr: domain.ErrUserNotFound}, passwords: &authPasswordManager{}},
		{name: "wrong password", users: &authUserRepository{findUser: domain.User{ID: uuid.New(), PasswordHash: "hash"}}, passwords: &authPasswordManager{passwordMatches: false}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewAuthService(&authTokenManager{}, test.users, test.passwords).Login(context.Background(), "alice", "wrong")
			if !errors.Is(err, ErrAuthentification) {
				t.Fatalf("Login() error = %v, want ErrAuthentification", err)
			}
		})
	}
}

func TestAuthServiceSignUp(t *testing.T) {
	t.Parallel()

	persistedID := uuid.New()
	users := &authUserRepository{saveUser: domain.User{ID: persistedID}}
	tokens := &authTokenManager{token: "signup-token"}
	passwords := &authPasswordManager{hash: "password-hash"}
	service := NewAuthService(tokens, users, passwords)

	token, err := service.SignUp(context.Background(), contract.SignupParams{Username: " alice ", Password: "Strong!1"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	if token != "signup-token" || users.savedUser.Username != "alice" || users.savedUser.PasswordHash != "password-hash" {
		t.Fatalf("unexpected signup state: token=%q user=%+v", token, users.savedUser)
	}
	if tokens.generatedFor != persistedID {
		t.Fatalf("token generated for %s, want %s", tokens.generatedFor, persistedID)
	}
}

func TestAuthServiceSignUpErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("weak password", func(t *testing.T) {
		t.Parallel()
		_, err := NewAuthService(&authTokenManager{}, &authUserRepository{}, &authPasswordManager{}).
			SignUp(context.Background(), contract.SignupParams{Username: "alice", Password: "weak"})
		if !errors.Is(err, domain.ErrInvalidPassword) {
			t.Fatalf("SignUp() error = %v, want ErrInvalidPassword", err)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		t.Parallel()
		_, err := NewAuthService(&authTokenManager{}, &authUserRepository{saveErr: domain.ErrUsernameAlreadyExists}, &authPasswordManager{hash: "hash"}).
			SignUp(context.Background(), contract.SignupParams{Username: "alice", Password: "Strong!1"})
		if !errors.Is(err, domain.ErrUsernameAlreadyExists) {
			t.Fatalf("SignUp() error = %v, want ErrUsernameAlreadyExists", err)
		}
	})

	t.Run("repository failure", func(t *testing.T) {
		t.Parallel()
		repositoryErr := errors.New("database unavailable")
		_, err := NewAuthService(&authTokenManager{}, &authUserRepository{saveErr: repositoryErr}, &authPasswordManager{hash: "hash"}).
			SignUp(context.Background(), contract.SignupParams{Username: "alice", Password: "Strong!1"})
		var signupErr ErrFailedSignup
		if !errors.As(err, &signupErr) || !errors.Is(err, repositoryErr) {
			t.Fatalf("SignUp() error = %v, want wrapped ErrFailedSignup", err)
		}
	})
}

func TestErrFailedSignupMethods(t *testing.T) {
	t.Parallel()

	cause := errors.New("database")
	err := ErrFailedSignup{Message: "signup failed", Err: cause}
	if err.Error() != "signup failed: database" || !errors.Is(err, cause) {
		t.Fatalf("unexpected signup error behavior: %v", err)
	}
}

type authUserRepository struct {
	contract.UserRepository
	findUser     domain.User
	findErr      error
	findUsername domain.Username
	saveUser     domain.User
	saveErr      error
	savedUser    domain.User
}

func (r *authUserRepository) FindUserByUsername(_ context.Context, username domain.Username) (domain.User, error) {
	r.findUsername = username
	return r.findUser, r.findErr
}

func (r *authUserRepository) SaveUser(_ context.Context, user domain.User) (domain.User, error) {
	r.savedUser = user
	if r.saveErr != nil {
		return domain.User{}, r.saveErr
	}
	if r.saveUser.ID != uuid.Nil {
		return r.saveUser, nil
	}
	return user, nil
}

type authTokenManager struct {
	token        string
	err          error
	generatedFor uuid.UUID
}

func (m *authTokenManager) GenerateToken(userID uuid.UUID) (string, error) {
	m.generatedFor = userID
	return m.token, m.err
}

func (*authTokenManager) VerifyToken(string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

type authPasswordManager struct {
	passwordMatches bool
	hash            string
	hashErr         error
}

func (m *authPasswordManager) CheckPassword(string, string) bool {
	return m.passwordMatches
}

func (m *authPasswordManager) HashPassword(string) (string, error) {
	return m.hash, m.hashErr
}
