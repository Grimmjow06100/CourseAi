package auth

import (
	"testing"
)

func TestClerkConfigValidate(t *testing.T) {
	t.Parallel()

	valid := ClerkConfig{
		SecretKey:         "sk_test_example",
		AuthorizedParties: []string{"http://localhost:5173", "https://app.example.com"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	valid.AuthorizedParties = []string{"not-a-url"}
	if err := valid.Validate(); err == nil {
		t.Fatal("invalid authorized party accepted")
	}
}
