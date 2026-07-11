package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abydv/devlab/internal/utils"
)

const (
	manifestFile = "workspace.json"
	logsDir      = "logs"
	dataDir      = "data"
	cacheDir     = "cache"
)

// Manager owns the lifecycle of Workspaces on disk. Every Workspace is
// stored under rootDir/<id>/, with its manifest at workspace.json and
// logs/, data/, and cache/ subdirectories.
type Manager struct {
	rootDir string
	mu      sync.RWMutex
}

// NewManager returns a Manager that stores Workspaces under rootDir.
// rootDir need not exist yet; it is created on first use.
func NewManager(rootDir string) *Manager {
	return &Manager{rootDir: rootDir}
}

// Create creates a new Workspace and persists it to disk. name must be
// unique (case-insensitive) among existing Workspaces.
func (m *Manager) Create(name, description, template string, services []string) (*Workspace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, err := m.list()
	if err != nil {
		return nil, err
	}
	for _, ws := range existing {
		if strings.EqualFold(ws.Name, name) {
			return nil, ErrNameExists
		}
	}

	id, err := utils.NewID()
	if err != nil {
		return nil, fmt.Errorf("workspace: generate id: %w", err)
	}

	if services == nil {
		services = []string{}
	}

	now := time.Now().UTC()
	ws := &Workspace{
		ID:          id,
		Name:        name,
		Description: description,
		Template:    template,
		Services:    services,
		Status:      StatusCreated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dir := m.dir(id)
	for _, sub := range []string{logsDir, dataDir, cacheDir} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, fmt.Errorf("workspace: create %s directory: %w", sub, err)
		}
	}

	if err := m.write(ws); err != nil {
		return nil, err
	}

	return ws, nil
}

// Get returns the Workspace with the given ID.
func (m *Manager) Get(id string) (*Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.read(id)
}

// List returns every known Workspace, ordered by creation time.
func (m *Manager) List() ([]*Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.list()
}

// Delete permanently removes a Workspace and all of its on-disk data.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := m.dir(id)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("workspace: stat %s: %w", id, err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("workspace: delete %s: %w", id, err)
	}
	return nil
}

func (m *Manager) dir(id string) string {
	return filepath.Join(m.rootDir, id)
}

func (m *Manager) read(id string) (*Workspace, error) {
	path := filepath.Join(m.dir(id), manifestFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("workspace: read manifest %s: %w", id, err)
	}

	var ws Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("workspace: parse manifest %s: %w", id, err)
	}
	return &ws, nil
}

func (m *Manager) list() ([]*Workspace, error) {
	entries, err := os.ReadDir(m.rootDir)
	if errors.Is(err, os.ErrNotExist) {
		return []*Workspace{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("workspace: read root directory: %w", err)
	}

	workspaces := make([]*Workspace, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ws, err := m.read(entry.Name())
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}

	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].CreatedAt.Before(workspaces[j].CreatedAt)
	})

	return workspaces, nil
}

func (m *Manager) write(ws *Workspace) error {
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return fmt.Errorf("workspace: marshal manifest %s: %w", ws.ID, err)
	}

	path := filepath.Join(m.dir(ws.ID), manifestFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("workspace: write manifest %s: %w", ws.ID, err)
	}
	return nil
}
