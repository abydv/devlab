package engine

import (
	"errors"
	"testing"

	"github.com/abydv/devlab/internal/workspace"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	return New(workspace.NewManager(t.TempDir()))
}

func TestEngineWorkspaceLifecycle(t *testing.T) {
	e := newTestEngine(t)

	created, err := e.CreateWorkspace("demo", "demo workspace", "kubernetes", []string{"kubernetes"})
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
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
