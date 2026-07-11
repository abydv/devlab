package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/abydv/devlab/internal/storage"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "devlab.db"))
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m, err := NewManager(t.TempDir(), db)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return m
}

func TestManagerCreate(t *testing.T) {
	m := newTestManager(t)

	ws, err := m.Create("my-workspace", "a test workspace", "kubernetes", []string{"jenkins"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if ws.ID == "" {
		t.Error("ID is empty")
	}
	if ws.Name != "my-workspace" {
		t.Errorf("Name = %q, want %q", ws.Name, "my-workspace")
	}
	if ws.Status != StatusCreated {
		t.Errorf("Status = %q, want %q", ws.Status, StatusCreated)
	}
	if ws.CreatedAt.IsZero() || ws.UpdatedAt.IsZero() {
		t.Error("CreatedAt/UpdatedAt not set")
	}

	for _, sub := range []string{logsDir, dataDir, cacheDir} {
		info, err := os.Stat(filepath.Join(m.dir(ws.ID), sub))
		if err != nil {
			t.Fatalf("stat %s: %v", sub, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", sub)
		}
	}

	if _, err := os.Stat(filepath.Join(m.dir(ws.ID), manifestFile)); err != nil {
		t.Fatalf("stat manifest: %v", err)
	}
}

func TestManagerCreateRequiresName(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.Create("  ", "", "", nil); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("Create() error = %v, want ErrNameRequired", err)
	}
}

func TestManagerCreateRejectsDuplicateName(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.Create("dup", "", "", nil); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := m.Create("DUP", "", "", nil); !errors.Is(err, ErrNameExists) {
		t.Fatalf("Create() error = %v, want ErrNameExists", err)
	}
}

func TestManagerGet(t *testing.T) {
	m := newTestManager(t)

	created, err := m.Create("my-workspace", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := m.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("Get() = %+v, want %+v", got, created)
	}
}

func TestManagerGetNotFound(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestManagerSetStatus(t *testing.T) {
	m := newTestManager(t)

	created, err := m.Create("my-workspace", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := m.SetStatus(created.ID, StatusRunning)
	if err != nil {
		t.Fatalf("SetStatus() error = %v", err)
	}
	if updated.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", updated.Status, StatusRunning)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && updated.UpdatedAt != created.UpdatedAt {
		t.Errorf("UpdatedAt = %v, want it to have advanced from %v", updated.UpdatedAt, created.UpdatedAt)
	}

	got, err := m.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("persisted Status = %q, want %q", got.Status, StatusRunning)
	}
}

func TestManagerSetStatusNotFound(t *testing.T) {
	m := newTestManager(t)

	if _, err := m.SetStatus("missing", StatusRunning); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetStatus() error = %v, want ErrNotFound", err)
	}
}

func TestManagerDataDir(t *testing.T) {
	m := newTestManager(t)

	created, err := m.Create("my-workspace", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := filepath.Join(m.dir(created.ID), "data")
	if got := m.DataDir(created.ID); got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
	if info, err := os.Stat(m.DataDir(created.ID)); err != nil || !info.IsDir() {
		t.Errorf("DataDir() path does not exist as a directory: %v", err)
	}
}

func TestManagerListEmpty(t *testing.T) {
	m := newTestManager(t)

	got, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
}

func TestManagerListOrdersByCreatedAt(t *testing.T) {
	m := newTestManager(t)

	first, err := m.Create("first", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	second, err := m.Create("second", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List() returned %d workspaces, want 2", len(got))
	}
	if got[0].ID != first.ID || got[1].ID != second.ID {
		t.Errorf("List() order = [%s, %s], want [%s, %s]", got[0].ID, got[1].ID, first.ID, second.ID)
	}
}

func TestManagerDelete(t *testing.T) {
	m := newTestManager(t)

	ws, err := m.Create("to-delete", "", "", nil)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := m.Delete(ws.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := m.Get(ws.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after delete error = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(m.dir(ws.ID)); !os.IsNotExist(err) {
		t.Fatalf("workspace directory still exists after Delete()")
	}

	list, err := m.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List() after delete = %v, want empty (index should be cleared)", list)
	}
}

func TestManagerDeleteNotFound(t *testing.T) {
	m := newTestManager(t)

	if err := m.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNotFound", err)
	}
}
