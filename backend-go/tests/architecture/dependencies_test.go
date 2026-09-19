package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Verify real production imports so relative package moves cannot silently
// reverse the dependency direction documented in AGENTS.md.
func TestProductionDependencyDirection(t *testing.T) {
	const internalPrefix = "github.com/Grimmjow06100/course-ai/backend-go/internal/"
	root := filepath.Join("..", "..", "internal")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		layer := strings.Split(filepath.ToSlash(relative), "/")[0]
		allowed := map[string][]string{
			"domain":   {"domain", "shared"},
			"contract": {"contract", "domain", "shared"},
			"service":  {"service", "contract", "domain", "shared"},
			"shared":   {"shared"},
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(importPath, internalPrefix) {
				// Inner layers may use standard Go packages and UUID values,
				// but must not acquire a framework or a vendor adapter.
				if _, inner := allowed[layer]; inner && strings.Contains(strings.Split(importPath, "/")[0], ".") && importPath != "github.com/google/uuid" {
					t.Errorf("%s imports external adapter %s", relative, importPath)
				}
				continue
			}
			target := strings.Split(strings.TrimPrefix(importPath, internalPrefix), "/")[0]
			if dependencies, inner := allowed[layer]; inner {
				found := false
				for _, dependency := range dependencies {
					found = found || dependency == target
				}
				if !found {
					t.Errorf("%s imports forbidden layer %s", relative, target)
				}
			}
			if layer == "infrastructure" && target == "service" {
				t.Errorf("%s imports service directly; depend on its contract", relative)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
