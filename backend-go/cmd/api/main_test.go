package main

import (
	"testing"
	"time"

	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/auth"
	"github.com/Grimmjow06100/course-ai/backend-go/internal/infrastructure/jobs"
)

func TestLoadHTTPServerConfigUsesRailwayPort(t *testing.T) {
	clearHTTPConfigEnvironment(t)
	t.Setenv("PORT", "9090")

	cfg, err := loadHTTPServerConfig()
	if err != nil {
		t.Fatalf("load HTTP config: %v", err)
	}
	if cfg.Address != ":9090" {
		t.Fatalf("address = %q, want :9090", cfg.Address)
	}
}

func TestValidateProductionConfigAcceptsHardenedConfiguration(t *testing.T) {
	worker := jobs.DefaultWorkerConfig()
	guardrails := apiGuardrailConfig{MaxBodyBytes: 64 * 1024, OpenAIMaxRetries: 0}
	err := validateProductionConfig(
		"production",
		[]string{"https://app.example.com"},
		auth.ClerkConfig{AuthorizedParties: []string{"https://app.example.com"}},
		worker,
		"gpt-5.6-luna",
		12000,
		guardrails,
	)
	if err != nil {
		t.Fatalf("production config: %v", err)
	}
}

func TestValidateProductionConfigRejectsUnsafeOrigin(t *testing.T) {
	err := validateProductionConfig(
		"production",
		[]string{"http://localhost:5173"},
		auth.ClerkConfig{AuthorizedParties: []string{"https://app.example.com"}},
		jobs.DefaultWorkerConfig(),
		"gpt-5.6-luna",
		12000,
		apiGuardrailConfig{MaxBodyBytes: 64 * 1024},
	)
	if err == nil {
		t.Fatal("expected an HTTP production origin to be rejected")
	}
}

func TestLoadAPIGuardrailConfigRejectsInvalidRetryBudget(t *testing.T) {
	clearGuardrailEnvironment(t)
	t.Setenv("OPENAI_MAX_RETRIES", "2")
	if _, err := loadAPIGuardrailConfig(); err == nil {
		t.Fatal("expected excessive SDK retries to be rejected")
	}
}

func TestLoadHTTPServerConfigPrefersExplicitAddress(t *testing.T) {
	clearHTTPConfigEnvironment(t)
	t.Setenv("PORT", "9090")
	t.Setenv("HTTP_ADDR", ":7070")

	cfg, err := loadHTTPServerConfig()
	if err != nil {
		t.Fatalf("load HTTP config: %v", err)
	}
	if cfg.Address != ":7070" {
		t.Fatalf("address = %q, want :7070", cfg.Address)
	}
}

func clearGuardrailEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HTTP_MAX_BODY_BYTES", "GENERATION_RATE_LIMIT_REQUESTS", "GENERATION_RATE_LIMIT_WINDOW",
		"GENERATION_MAX_ACTIVE_PER_USER", "GENERATION_MAX_DAILY_PER_USER", "GENERATION_MAX_PENDING_JOBS", "OPENAI_MAX_RETRIES",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadHTTPServerConfigRejectsInvalidTimeout(t *testing.T) {
	clearHTTPConfigEnvironment(t)
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "never")

	if _, err := loadHTTPServerConfig(); err == nil {
		t.Fatal("expected malformed shutdown timeout to be rejected")
	}
}

func TestLoadHTTPServerConfigLoadsProductionTimeouts(t *testing.T) {
	clearHTTPConfigEnvironment(t)
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "3s")
	t.Setenv("HTTP_READ_TIMEOUT", "10s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "90s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "45s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "15s")

	cfg, err := loadHTTPServerConfig()
	if err != nil {
		t.Fatalf("load HTTP config: %v", err)
	}
	if cfg.ReadHeaderTimeout != 3*time.Second || cfg.WriteTimeout != 90*time.Second || cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("unexpected HTTP config: %+v", cfg)
	}
}

func clearHTTPConfigEnvironment(t *testing.T) {
	t.Helper()
	keys := []string{
		"PORT",
		"HTTP_ADDR",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
