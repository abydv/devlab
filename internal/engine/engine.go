// Package engine orchestrates Workspaces. It is the seam between the
// REST API and the Workspace Manager: the API layer holds no business
// logic and delegates every operation to the Engine.
package engine

import (
	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

// Engine orchestrates Workspace lifecycle operations.
type Engine struct {
	workspaces *workspace.Manager
	templates  *template.Registry
}

// New returns an Engine backed by the given Workspace Manager and
// Template Registry.
func New(workspaces *workspace.Manager, templates *template.Registry) *Engine {
	return &Engine{workspaces: workspaces, templates: templates}
}

// CreateWorkspace creates a new Workspace from the named Template. The
// Workspace's Services are resolved from the Template.
func (e *Engine) CreateWorkspace(name, description, templateName string) (*workspace.Workspace, error) {
	tmpl, err := e.templates.Get(templateName)
	if err != nil {
		return nil, err
	}

	services := append([]string(nil), tmpl.Services...)
	return e.workspaces.Create(name, description, tmpl.Name, services)
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

// ListTemplates returns every available Template a Workspace can be
// created from.
func (e *Engine) ListTemplates() []*template.Template {
	return e.templates.List()
}

// GetTemplate returns the Template with the given name.
func (e *Engine) GetTemplate(name string) (*template.Template, error) {
	return e.templates.Get(name)
}
