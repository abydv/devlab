// Package engine orchestrates Workspaces. It is the seam between the
// REST API and the Workspace Manager: the API layer holds no business
// logic and delegates every operation to the Engine.
package engine

import "github.com/abydv/devlab/internal/workspace"

// Engine orchestrates Workspace lifecycle operations.
type Engine struct {
	workspaces *workspace.Manager
}

// New returns an Engine backed by the given Workspace Manager.
func New(workspaces *workspace.Manager) *Engine {
	return &Engine{workspaces: workspaces}
}

// CreateWorkspace creates a new Workspace.
func (e *Engine) CreateWorkspace(name, description, template string, services []string) (*workspace.Workspace, error) {
	return e.workspaces.Create(name, description, template, services)
}

// GetWorkspace returns the Workspace with the given ID.
func (e *Engine) GetWorkspace(id string) (*workspace.Workspace, error) {
	return e.workspaces.Get(id)
}

// ListWorkspaces returns every known Workspace.
func (e *Engine) ListWorkspaces() ([]*workspace.Workspace, error) {
	return e.workspaces.List()
}

// DeleteWorkspace permanently removes a Workspace.
func (e *Engine) DeleteWorkspace(id string) error {
	return e.workspaces.Delete(id)
}
