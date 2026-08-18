package prompts

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadReadsOnlyPromptFiles(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "analysis.prompt.md"), "analysis content")
	writeTestFile(t, filepath.Join(directory, "architecture.prompt.md"), "architecture content")
	writeTestFile(t, filepath.Join(directory, "README.md"), "ignored")
	if err := os.Mkdir(filepath.Join(directory, "nested.prompt.md"), 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}

	store, err := Load(directory)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, ok := store.Get("analysis"); !ok || got != "analysis content" {
		t.Fatalf("Get(analysis) = %q, %t", got, ok)
	}
	if _, ok := store.Get("README"); ok {
		t.Fatal("non-prompt file should not be loaded")
	}
	if got, want := store.Names(), []string{"analysis", "architecture"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestLoadReturnsDirectoryError(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing directory error")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
