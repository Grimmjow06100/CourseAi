package auth

import (
	"errors"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	manager := new(PasswordManager)
	hash, err := manager.HashPassword("Password!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "Password!" {
		t.Fatal("expected hash to differ from raw password")
	}

	if !manager.CheckPassword("Password!", hash) {
		t.Fatal("expected password to match hash")
	}

	if manager.CheckPassword("WrongPassword!", hash) {
		t.Fatal("expected wrong password to be rejected")
	}
}

func TestHashPasswordRejectsBlankPassword(t *testing.T) {
	manager := new(PasswordManager)
	_, err := manager.HashPassword("   ")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestCheckPasswordRejectsEmptyInputs(t *testing.T) {
	manager := new(PasswordManager)

	if manager.CheckPassword("", "hash") {
		t.Fatal("expected empty password to be rejected")
	}

	if manager.CheckPassword("Password!", "") {
		t.Fatal("expected empty hash to be rejected")
	}
}
