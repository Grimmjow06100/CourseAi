package main

import (
	"testing"
	"time"
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
