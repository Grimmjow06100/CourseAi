package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     Username
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Password string
type Username string

func NewUser(username Username, passwordHash string) (User, error) {
	return NewUserAt(username, passwordHash, time.Now())
}

func NewUserAt(username Username, passwordHash string, now time.Time) (User, error) {
	username = username.Normalize()
	passwordHash = strings.TrimSpace(passwordHash)

	if err := username.Validate(); err != nil {
		return User{}, err
	}

	if passwordHash == "" {
		return User{}, ErrInvalidPasswordHash
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	return User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (u User) Validate() error {
	if u.ID == uuid.Nil {
		return fmt.Errorf("%w: user id", ErrBlankField)
	}
	if err := u.Username.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(u.PasswordHash) == "" {
		return ErrInvalidPasswordHash
	}
	return nil
}

func (p Password) Validate() error {
	value := string(p)

	if utf8.RuneCountInString(value) < 8 {
		return ErrInvalidPassword
	}

	var hasUpper bool
	var hasSpecial bool

	for _, char := range value {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasSpecial {
		return ErrInvalidPassword
	}

	return nil
}

func (p Username) Normalize() Username {
	return Username(normalizeText(string(p)))
}

func (p Username) Validate() error {
	value := string(p.Normalize())
	length := utf8.RuneCountInString(value)

	if length < 3 || length > 50 {
		return ErrInvalidUsername
	}

	return nil
}
