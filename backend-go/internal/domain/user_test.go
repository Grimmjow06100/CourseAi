package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewUserAtNormalizesUsername(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 13, 10, 0, 0, 0, time.UTC)
	user, err := NewUserAt(Username("  Alice   Martin "), " hash ", now)
	if err != nil {
		t.Fatalf("NewUserAt() error = %v", err)
	}
	if user.Username != Username("Alice Martin") || user.PasswordHash != "hash" {
		t.Fatalf("unexpected normalized user: %+v", user)
	}
	if user.ID == uuid.Nil || !user.CreatedAt.Equal(now) || !user.UpdatedAt.Equal(now) {
		t.Fatal("constructor did not initialize identity and timestamps")
	}
}

func TestNewUserAtRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	if _, err := NewUserAt(Username("ab"), "hash", time.Now()); !errors.Is(err, ErrInvalidUsername) {
		t.Fatalf("short username error = %v, want ErrInvalidUsername", err)
	}
	if _, err := NewUserAt(Username("alice"), "  ", time.Now()); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("blank hash error = %v, want ErrInvalidPasswordHash", err)
	}
}

func TestPasswordValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		password Password
		wantErr  bool
	}{
		{password: Password("Strong!1")},
		{password: Password("short!A"), wantErr: true},
		{password: Password("lowercase!"), wantErr: true},
		{password: Password("Uppercase1"), wantErr: true},
	}
	for _, test := range tests {
		if err := test.password.Validate(); (err != nil) != test.wantErr {
			t.Fatalf("Password(%q).Validate() error = %v, wantErr %t", test.password, err, test.wantErr)
		}
	}
}

func TestUserValidate(t *testing.T) {
	t.Parallel()

	valid := User{ID: uuid.New(), Username: Username("alice"), PasswordHash: "hash"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid user rejected: %v", err)
	}
	valid.ID = uuid.Nil
	if err := valid.Validate(); !errors.Is(err, ErrBlankField) {
		t.Fatalf("nil id error = %v, want ErrBlankField", err)
	}
}
