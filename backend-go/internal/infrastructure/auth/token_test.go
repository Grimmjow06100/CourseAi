package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenManagerGenerateAndVerifyToken(t *testing.T) {
	manager, err := NewTokenManager("test-secret", time.Duration(60*time.Second))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time {
		return now
	}

	userID := uuid.New()

	token, err := manager.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	gotUserID, err := manager.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}

	if gotUserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, gotUserID)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := NewTokenManager("test-secret", time.Duration(60*time.Second))
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time {
		return now
	}

	token, err := manager.GenerateToken(uuid.New())
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	manager.now = func() time.Time {
		return now.Add(2 * time.Minute)
	}

	_, err = manager.VerifyToken(token)
	if err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestNewTokenManagerRequiresSecret(t *testing.T) {
	_, err := NewTokenManager("   ", time.Duration(60*time.Second))
	if !errors.Is(err, ErrMissingSecret) {
		t.Fatalf("expected ErrMissingSecret, got %v", err)
	}
}

func TestTokenManagerRequiresPositiveDuration(t *testing.T) {
	_, err := NewTokenManager("test-secret", time.Duration(0))
	if err == nil {
		t.Fatalf("expected duration error")
	}

}
