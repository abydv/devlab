package workspace

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// Manager owns the lifecycle of Workspaces. workspace.json remains the
// source of truth for each Workspace, stored under rootDir/<id>/
// alongside its logs/, data/, and cache/ subdirectories. A SQLite index
// (db) provides fast, ordered lookups without scanning the filesystem.
type Manager struct {
	rootDir string
	db      *sql.DB
	mu      sync.RWMutex
}

// NewManager returns a Manager that stores Workspaces under rootDir and
// indexes them in db. rootDir need not exist yet; it is created on
// first use.
func NewManager(rootDir string, db *sql.DB) (*Manager, error) {
	if err := initSchema(db); err != nil {
		return nil, err
	}
	return &Manager{rootDir: rootDir, db: db}, nil
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

	taken, err := m.indexNameTaken(name)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameExists
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

	if err := m.indexInsert(ws); err != nil {
		return nil, err
	}

	dir := m.dir(id)
	for _, sub := range []string{logsDir, dataDir, cacheDir} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			_ = m.indexDelete(id)
			return nil, fmt.Errorf("workspace: create %s directory: %w", sub, err)
		}
	}

	if err := m.write(ws); err != nil {
		_ = m.indexDelete(id)
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

	return m.indexDelete(id)
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
	ids, err := m.indexList()
	if err != nil {
		return nil, err
	}

	workspaces := make([]*Workspace, 0, len(ids))
	for _, id := range ids {
		ws, err := m.read(id)
		if errors.Is(err, ErrNotFound) {
			// Indexed but missing on disk: index and filesystem have
			// drifted apart. Skip rather than fail the whole listing.
			continue
		}
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}

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
