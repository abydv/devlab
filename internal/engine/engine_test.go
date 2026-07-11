package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service/factory"
	"github.com/abydv/devlab/internal/storage"
	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

// fakeExec is a runtime.Runtime test double that records every Command
// it receives and lets tests script the response by binary+args.
type fakeExec struct {
	calls  []runtime.Command
	handle func(cmd runtime.Command) (*runtime.Result, error)
}

func (f *fakeExec) Execute(_ context.Context, cmd runtime.Command) (*runtime.Result, error) {
	f.calls = append(f.calls, cmd)
	if f.handle != nil {
		return f.handle(cmd)
	}
	return &runtime.Result{ExitCode: 0}, nil
}

func (f *fakeExec) callsFor(binary string) []runtime.Command {
	var out []runtime.Command
	for _, c := range f.calls {
		if c.Name == binary {
			out = append(out, c)
		}
	}
	return out
}

func newTestEngineWithFake(t *testing.T, fake *fakeExec) *Engine {
	t.Helper()

	templatesDir := t.TempDir()
	writeTemplate(t, templatesDir, "kubernetes.json", `{"name":"kubernetes","description":"A Kubernetes workspace.","services":["kubernetes"]}`)

	templates := template.NewRegistry(templatesDir)
	if err := templates.Load(); err != nil {
		t.Fatalf("templates.Load() error = %v", err)
	}

	db, err := storage.Open(filepath.Join(t.TempDir(), "devlab.db"))
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	workspaces, err := workspace.NewManager(t.TempDir(), db)
	if err != nil {
		t.Fatalf("workspace.NewManager() error = %v", err)
	}

	services := factory.New(k3d.New(fake), docker.New(fake))

	return New(workspaces, templates, services)
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	return newTestEngineWithFake(t, &fakeExec{})
}

func writeTemplate(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func TestEngineWorkspaceCRUD(t *testing.T) {
	e := newTestEngine(t)

	created, err := e.CreateWorkspace("demo", "demo workspace", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}
	if len(created.Services) != 1 || created.Services[0] != "kubernetes" {
		t.Errorf("Services = %v, want [kubernetes] (resolved from template)", created.Services)
	}
	if created.Status != workspace.StatusCreated {
		t.Errorf("Status = %q, want %q (no Service resources provisioned yet)", created.Status, workspace.StatusCreated)
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

	if err := e.DeleteWorkspace(context.Background(), created.ID); err != nil {
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

// clusterListHandler scripts a fakeExec to answer "k3d cluster list"
// with a single cluster named clusterName, and succeed (exit 0) on
// everything else — enough for the Kubernetes Service's Status/Create/
// Start/Stop/Reset/Delete calls to behave like a real, running cluster.
func clusterListHandler(clusterName string) func(cmd runtime.Command) (*runtime.Result, error) {
	return func(cmd runtime.Command) (*runtime.Result, error) {
		if cmd.Name == "k3d" && strings.Join(cmd.Args, " ") == "cluster list --output json" {
			return &runtime.Result{ExitCode: 0, Stdout: `[{"name":"` + clusterName + `"}]`}, nil
		}
		if cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "inspect" {
			return &runtime.Result{ExitCode: 0, Stdout: "running\n"}, nil
		}
		return &runtime.Result{ExitCode: 0}, nil
	}
}

func TestEngineStartWorkspaceCreatesThenStarts(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			// First Status check (before Create) reports not found so
			// StartWorkspace provisions the cluster.
			if cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "inspect" {
				return &runtime.Result{ExitCode: 1, Stderr: "Error response from daemon: No such container: x"}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	e := newTestEngineWithFake(t, fake)

	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	if err := e.StartWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("StartWorkspace() error = %v", err)
	}

	k3dCalls := fake.callsFor("k3d")
	var verbs []string
	for _, c := range k3dCalls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	wantCreate := "cluster create devlab-" + created.ID + "-kubernetes"
	wantStart := "cluster start devlab-" + created.ID + "-kubernetes"
	if strings.Join(verbs, "; ") != wantCreate+"; "+wantStart {
		t.Errorf("k3d call sequence = %v, want [%s, %s]", verbs, wantCreate, wantStart)
	}
}

func TestEngineStartWorkspaceUpdatesStatus(t *testing.T) {
	fake := &fakeExec{}
	e := newTestEngineWithFake(t, fake)

	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	// Now that the workspace (and its derived cluster name) exists,
	// script the fake to report that cluster as running so
	// StartWorkspace's status sync observes it.
	clusterName := "devlab-" + created.ID + "-kubernetes"
	fake.handle = clusterListHandler(clusterName)

	if err := e.StartWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("StartWorkspace() error = %v", err)
	}

	got, err := e.GetWorkspace(created.ID)
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if got.Status != workspace.StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, workspace.StatusRunning)
	}
}

// stoppedClusterHandler scripts a fakeExec so the Kubernetes Service's
// Status/ClusterExists calls see a real, stopped cluster: "k3d cluster
// list" reports no clusters (satisfies ClusterExists's list-based
// check) and "docker inspect" reports the server node as exited.
func stoppedClusterHandler(cmd runtime.Command) (*runtime.Result, error) {
	if cmd.Name == "k3d" && strings.Join(cmd.Args, " ") == "cluster list --output json" {
		return &runtime.Result{ExitCode: 0, Stdout: "[]"}, nil
	}
	if cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "inspect" {
		return &runtime.Result{ExitCode: 0, Stdout: "exited\n"}, nil
	}
	return &runtime.Result{ExitCode: 0}, nil
}

func TestEngineStopWorkspace(t *testing.T) {
	fake := &fakeExec{handle: stoppedClusterHandler}
	e := newTestEngineWithFake(t, fake)

	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	if err := e.StopWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("StopWorkspace() error = %v", err)
	}

	got, err := e.GetWorkspace(created.ID)
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	if got.Status != workspace.StatusStopped {
		t.Errorf("Status = %q, want %q", got.Status, workspace.StatusStopped)
	}
}

func TestEngineResetWorkspace(t *testing.T) {
	fake := &fakeExec{handle: stoppedClusterHandler}
	e := newTestEngineWithFake(t, fake)

	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	if err := e.ResetWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("ResetWorkspace() error = %v", err)
	}
}

func TestEngineDeleteWorkspaceCleansUpServices(t *testing.T) {
	fake := &fakeExec{}
	e := newTestEngineWithFake(t, fake)

	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	if err := e.DeleteWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}

	k3dCalls := fake.callsFor("k3d")
	if len(k3dCalls) != 1 || strings.Join(k3dCalls[0].Args, " ") != "cluster delete devlab-"+created.ID+"-kubernetes" {
		t.Errorf("k3d calls = %v, want a single cluster delete", k3dCalls)
	}

	if _, err := e.GetWorkspace(created.ID); !errors.Is(err, workspace.ErrNotFound) {
		t.Fatalf("GetWorkspace() after delete error = %v, want ErrNotFound", err)
	}
}

func TestEngineWorkspaceLogs(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "logs" {
				return &runtime.Result{ExitCode: 0, Stdout: "server log line\n"}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	e := newTestEngineWithFake(t, fake)
	created, err := e.CreateWorkspace("demo", "", "kubernetes")
	if err != nil {
		t.Fatalf("CreateWorkspace() error = %v", err)
	}

	logs, err := e.WorkspaceLogs(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("WorkspaceLogs() error = %v", err)
	}
	if !strings.Contains(logs, "=== kubernetes ===") || !strings.Contains(logs, "server log line") {
		t.Errorf("WorkspaceLogs() = %q, want it to contain the labeled kubernetes service logs", logs)
	}
}
