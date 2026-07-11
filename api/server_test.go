package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/abydv/devlab/internal/engine"
	"github.com/abydv/devlab/internal/runtime"
	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service/factory"
	"github.com/abydv/devlab/internal/storage"
	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

// fakeExec is a runtime.Runtime test double that records every Command
// it receives and lets tests script the response.
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

func newTestApp(t *testing.T, fake *fakeExec) *fiber.App {
	t.Helper()

	templatesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(templatesDir, "kubernetes.json"),
		[]byte(`{"name":"kubernetes","description":"A Kubernetes workspace.","services":["kubernetes"]}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
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
	e := engine.New(workspaces, templates, services)

	return New(e)
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test(%s %s): %v", method, path, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestHealth(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodGet, "/healthz", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
}

func TestCreateAndGetWorkspace(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{
		Name: "demo", Description: "a demo workspace", Template: "kubernetes",
	})
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, fiber.StatusCreated)
	}
	var created workspace.Workspace
	decodeJSON(t, resp, &created)
	if created.Name != "demo" || len(created.Services) != 1 || created.Services[0] != "kubernetes" {
		t.Fatalf("created = %+v, want name=demo services=[kubernetes]", created)
	}

	resp = doJSON(t, app, http.MethodGet, "/api/workspaces/"+created.ID, nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("get status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	var got workspace.Workspace
	decodeJSON(t, resp, &got)
	if got.ID != created.ID {
		t.Errorf("got.ID = %q, want %q", got.ID, created.ID)
	}
}

func TestCreateWorkspaceUnknownTemplate(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{
		Name: "demo", Template: "does-not-exist",
	})
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

func TestCreateWorkspaceMalformedBody(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	req, err := http.NewRequest(http.MethodPost, "/api/workspaces", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusBadRequest)
	}
}

func TestGetWorkspaceNotFound(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodGet, "/api/workspaces/missing", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

func TestListWorkspaces(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{Name: "demo", Template: "kubernetes"})

	resp := doJSON(t, app, http.MethodGet, "/api/workspaces", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	var list []*workspace.Workspace
	decodeJSON(t, resp, &list)
	if len(list) != 1 {
		t.Fatalf("list = %v, want 1 workspace", list)
	}
}

func TestListAndGetTemplate(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodGet, "/api/templates", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	var list []*template.Template
	decodeJSON(t, resp, &list)
	if len(list) != 1 || list[0].Name != "kubernetes" {
		t.Fatalf("list = %v, want [kubernetes]", list)
	}

	resp = doJSON(t, app, http.MethodGet, "/api/templates/kubernetes", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}

	resp = doJSON(t, app, http.MethodGet, "/api/templates/missing", nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

// statefulClusterHandler simulates a real k3d/docker backend well
// enough for Status to reflect Start/Stop: it tracks whether the
// cluster's server node is "running" and answers "k3d cluster list"
// and "docker inspect" accordingly, matching how the Kubernetes
// Service actually queries state.
func statefulClusterHandler(clusterName string) func(cmd runtime.Command) (*runtime.Result, error) {
	running := false
	return func(cmd runtime.Command) (*runtime.Result, error) {
		switch {
		case cmd.Name == "k3d" && len(cmd.Args) >= 2 && cmd.Args[0] == "cluster" && (cmd.Args[1] == "create" || cmd.Args[1] == "start"):
			running = true
		case cmd.Name == "k3d" && len(cmd.Args) >= 2 && cmd.Args[0] == "cluster" && cmd.Args[1] == "stop":
			running = false
		case cmd.Name == "k3d" && strings.Join(cmd.Args, " ") == "cluster list --output json":
			return &runtime.Result{ExitCode: 0, Stdout: `[{"name":"` + clusterName + `"}]`}, nil
		case cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "inspect":
			if running {
				return &runtime.Result{ExitCode: 0, Stdout: "running\n"}, nil
			}
			return &runtime.Result{ExitCode: 0, Stdout: "exited\n"}, nil
		}
		return &runtime.Result{ExitCode: 0}, nil
	}
}

func TestWorkspaceStartStopStatusLifecycle(t *testing.T) {
	fake := &fakeExec{}
	app := newTestApp(t, fake)

	resp := doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{Name: "demo", Template: "kubernetes"})
	var created workspace.Workspace
	decodeJSON(t, resp, &created)

	fake.handle = statefulClusterHandler("devlab-" + created.ID + "-kubernetes")

	resp = doJSON(t, app, http.MethodPost, "/api/workspaces/"+created.ID+"/start", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("start status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	var started workspace.Workspace
	decodeJSON(t, resp, &started)
	if started.Status != workspace.StatusRunning {
		t.Errorf("Status after start = %q, want %q", started.Status, workspace.StatusRunning)
	}

	resp = doJSON(t, app, http.MethodGet, "/api/workspaces/"+created.ID+"/status", nil)
	var status statusResponse
	decodeJSON(t, resp, &status)
	if status.Status != workspace.StatusRunning {
		t.Errorf("status endpoint = %q, want %q", status.Status, workspace.StatusRunning)
	}

	resp = doJSON(t, app, http.MethodPost, "/api/workspaces/"+created.ID+"/stop", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("stop status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	var stopped workspace.Workspace
	decodeJSON(t, resp, &stopped)
	if stopped.Status != workspace.StatusStopped {
		t.Errorf("Status after stop = %q, want %q", stopped.Status, workspace.StatusStopped)
	}
}

func TestWorkspaceLogs(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if cmd.Name == "docker" && len(cmd.Args) > 0 && cmd.Args[0] == "logs" {
				return &runtime.Result{ExitCode: 0, Stdout: "server log line\n"}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	app := newTestApp(t, fake)

	resp := doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{Name: "demo", Template: "kubernetes"})
	var created workspace.Workspace
	decodeJSON(t, resp, &created)

	resp = doJSON(t, app, http.MethodGet, "/api/workspaces/"+created.ID+"/logs", nil)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "=== kubernetes ===") || !strings.Contains(string(body), "server log line") {
		t.Errorf("logs body = %q, want it to contain the labeled kubernetes service logs", body)
	}
}

func TestDeleteWorkspace(t *testing.T) {
	app := newTestApp(t, &fakeExec{})

	resp := doJSON(t, app, http.MethodPost, "/api/workspaces", createWorkspaceRequest{Name: "demo", Template: "kubernetes"})
	var created workspace.Workspace
	decodeJSON(t, resp, &created)

	resp = doJSON(t, app, http.MethodDelete, "/api/workspaces/"+created.ID, nil)
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}

	resp = doJSON(t, app, http.MethodGet, "/api/workspaces/"+created.ID, nil)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("get after delete status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}
