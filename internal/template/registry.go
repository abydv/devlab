package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Registry loads Template definitions from a directory of JSON files
// and serves them from memory.
type Registry struct {
	dir string

	mu        sync.RWMutex
	templates map[string]*Template
}

// NewRegistry returns a Registry that loads Template definitions from
// dir. Call Load before Get or List.
func NewRegistry(dir string) *Registry {
	return &Registry{dir: dir, templates: make(map[string]*Template)}
}

// Load reads every *.json file in the Registry's directory, validates
// it as a Template definition, and replaces the in-memory catalog. A
// missing directory is treated as an empty catalog, not an error.
func (r *Registry) Load() error {
	entries, err := os.ReadDir(r.dir)
	if os.IsNotExist(err) {
		r.mu.Lock()
		r.templates = make(map[string]*Template)
		r.mu.Unlock()
		return nil
	}
	if err != nil {
		return fmt.Errorf("template: read directory: %w", err)
	}

	loaded := make(map[string]*Template)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(r.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("template: read %s: %w", entry.Name(), err)
		}

		var t Template
		if err := json.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("template: parse %s: %w", entry.Name(), err)
		}
		if err := t.validate(); err != nil {
			return fmt.Errorf("template: %s: %w", entry.Name(), err)
		}
		if _, exists := loaded[t.Name]; exists {
			return fmt.Errorf("template: %s: %w", t.Name, ErrNameExists)
		}
		loaded[t.Name] = &t
	}

	r.mu.Lock()
	r.templates = loaded
	r.mu.Unlock()
	return nil
}

// Get returns the Template with the given name.
func (r *Registry) Get(name string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.templates[name]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

// List returns every loaded Template, ordered by name.
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Template, 0, len(r.templates))
	for _, t := range r.templates {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}
