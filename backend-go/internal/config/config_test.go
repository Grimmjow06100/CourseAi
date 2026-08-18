package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadDotEnvLoadsValuesWithoutOverwritingEnvironment(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	content := strings.Join([]string{
		"# comment",
		"COURSE_AI_TEST_DOTENV=from-file",
		"COURSE_AI_TEST_QUOTED=\"quoted value\"",
		"COURSE_AI_TEST_EXISTING=from-file",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}
	t.Setenv("COURSE_AI_TEST_DOTENV", "")
	if err := os.Unsetenv("COURSE_AI_TEST_DOTENV"); err != nil {
		t.Fatalf("unset dotenv key: %v", err)
	}
	t.Setenv("COURSE_AI_TEST_QUOTED", "")
	if err := os.Unsetenv("COURSE_AI_TEST_QUOTED"); err != nil {
		t.Fatalf("unset quoted key: %v", err)
	}
	t.Setenv("COURSE_AI_TEST_EXISTING", "process")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv() error = %v", err)
	}
	if got := os.Getenv("COURSE_AI_TEST_DOTENV"); got != "from-file" {
		t.Fatalf("loaded value = %q", got)
	}
	if got := os.Getenv("COURSE_AI_TEST_QUOTED"); got != "quoted value" {
		t.Fatalf("quoted value = %q", got)
	}
	if got := os.Getenv("COURSE_AI_TEST_EXISTING"); got != "process" {
		t.Fatalf("existing value was overwritten: %q", got)
	}
}

func TestLoadDotEnvAcceptsMissingFile(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("missing dotenv should be optional: %v", err)
	}
}

func TestLoadDotEnvRejectsMalformedLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "missing separator", content: "INVALID"},
		{name: "empty key", content: " =value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".env")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatalf("write dotenv: %v", err)
			}
			if err := loadDotEnv(path); err == nil {
				t.Fatal("expected malformed dotenv error")
			}
		})
	}
}

func TestGetEnvRequiresValue(t *testing.T) {
	t.Setenv("COURSE_AI_TEST_REQUIRED", "")
	if _, err := GetEnv[string]("COURSE_AI_TEST_REQUIRED"); err == nil {
		t.Fatal("expected missing required value error")
	}
}

func TestGetEnvParsesSupportedTypes(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "string", run: func(t *testing.T) { t.Setenv("TEST_VALUE", "value"); assertEnvValue(t, "value") }},
		{name: "integer", run: func(t *testing.T) { t.Setenv("TEST_VALUE", "42"); assertEnvValue(t, 42) }},
		{name: "boolean", run: func(t *testing.T) { t.Setenv("TEST_VALUE", "true"); assertEnvValue(t, true) }},
		{name: "float", run: func(t *testing.T) { t.Setenv("TEST_VALUE", "2.5"); assertEnvValue(t, 2.5) }},
		{name: "duration", run: func(t *testing.T) { t.Setenv("TEST_VALUE", "45s"); assertEnvValue(t, 45*time.Second) }},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func assertEnvValue[T EnvParsable](t *testing.T, want T) {
	t.Helper()
	got, err := GetEnv[T]("TEST_VALUE")
	if err != nil {
		t.Fatalf("GetEnv() error = %v", err)
	}
	if got != want {
		t.Fatalf("GetEnv() = %v, want %v", got, want)
	}
}

func TestGetEnvWithDefaultUsesFallbackWhenVariableIsMissing(t *testing.T) {
	t.Setenv("COURSE_AI_TEST_MISSING_DURATION", "")

	value, err := GetEnvWithDefault("COURSE_AI_TEST_MISSING_DURATION", 30*time.Second)
	if err != nil {
		t.Fatalf("get environment with default: %v", err)
	}
	if value != 30*time.Second {
		t.Fatalf("value = %s, want 30s", value)
	}
}

func TestGetEnvWithDefaultParsesConfiguredValue(t *testing.T) {
	t.Setenv("COURSE_AI_TEST_CONCURRENCY", "4")

	value, err := GetEnvWithDefault("COURSE_AI_TEST_CONCURRENCY", 1)
	if err != nil {
		t.Fatalf("get environment with default: %v", err)
	}
	if value != 4 {
		t.Fatalf("value = %d, want 4", value)
	}
}

func TestGetEnvWithDefaultRejectsMalformedValue(t *testing.T) {
	t.Setenv("COURSE_AI_TEST_CONCURRENCY", "many")

	if _, err := GetEnvWithDefault("COURSE_AI_TEST_CONCURRENCY", 1); err == nil {
		t.Fatal("expected malformed environment value to be rejected")
	}
}

func TestLoadReadsDotEnvFromWorkingDirectory(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	if err := os.WriteFile(path, []byte("COURSE_AI_TEST_LOAD=value\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}
	t.Setenv("COURSE_AI_TEST_LOAD", "")
	if err := os.Unsetenv("COURSE_AI_TEST_LOAD"); err != nil {
		t.Fatalf("unset environment: %v", err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })

	if err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := os.Getenv("COURSE_AI_TEST_LOAD"); got != "value" {
		t.Fatalf("loaded value = %q", got)
	}
}
