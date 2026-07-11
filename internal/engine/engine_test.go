package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()

	templatesDir := t.TempDir()
	writeTemplate(t, templatesDir, "kubernetes.json", `{"name":"kubernetes","description":"A Kubernetes workspace.","services":["kubernetes"]}`)

	templates := template.NewRegistry(templatesDir)
	if err := templates.Load(); err != nil {
		t.Fatalf("templates.Load() error = %v", err)
	}

	return New(workspace.NewManager(t.TempDir()), templates)
}

func writeTemplate(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func TestEngineWorkspaceLifecycle(t *testing.T) {
	e := newTestEngine(t)

	created, err := e.CreateWorkspace("demo", "demo workspace", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if len(created.Services) != 1 || created.Services[0] != "kubernetes" {
		t.Errorf("Services = %v, want [kubernetes] (resolved from template)", created.Services)
	}

	got, err := e.GetWorkspace(created.ID)
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetWorkspace() ID = %q, want %q", got.ID, created.ID)
	}

	list, err := e.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListWorkspaces() returned %d workspaces, want 1", len(list))
	}

	if err := e.DeleteWorkspace(created.ID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}

	if _, err := e.GetWorkspace(created.ID); !errors.Is(err, workspace.ErrNotFound) {
		t.Fatalf("GetWorkspace() after delete error = %v, want ErrNotFound", err)
	}
}

func TestEngineCreateWorkspaceUnknownTemplate(t *testing.T) {
	e := newTestEngine(t)

	if _, err := e.CreateWorkspace("demo", "", "does-not-exist"); !errors.Is(err, template.ErrNotFound) {
		t.Fatalf("CreateWorkspace() error = %v, want ErrNotFound", err)
	}
}

func TestEngineListAndGetTemplate(t *testing.T) {
	e := newTestEngine(t)

	list := e.ListTemplates()
	if len(list) != 1 || list[0].Name != "kubernetes" {
		t.Fatalf("ListTemplates() = %v, want [kubernetes]", list)
	}

	got, err := e.GetTemplate("kubernetes")
	if err != nil {
		t.Fatalf("GetTemplate() error = %v", err)
	}
	if got.Name != "kubernetes" {
		t.Errorf("GetTemplate() Name = %q, want %q", got.Name, "kubernetes")
	}
}
