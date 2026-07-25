package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Store struct {
	prompts map[string]string
}

func Load(directory string) (*Store, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	store := &Store{
		prompts: make(map[string]string),
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".prompt.md") {
			continue
		}

		path := filepath.Join(directory, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read prompt %s: %w", entry.Name(), err)
		}

		name := strings.TrimSuffix(entry.Name(), ".prompt.md")
		store.prompts[name] = string(content)
	}

	return store, nil
}

func (s *Store) Get(name string) (string, bool) {
	prompt, ok := s.prompts[name]
	return prompt, ok
}

func (s *Store) Names() []string {
	names := make([]string, 0, len(s.prompts))
	for name := range s.prompts {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}
