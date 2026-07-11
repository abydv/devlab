package k3d

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
)

// fakeRuntime is a runtime.Runtime test double that records every
// Command it receives and lets tests script the response.
type fakeRuntime struct {
	calls  []runtime.Command
	handle func(cmd runtime.Command) (*runtime.Result, error)
}

func (f *fakeRuntime) Execute(_ context.Context, cmd runtime.Command) (*runtime.Result, error) {
	f.calls = append(f.calls, cmd)
	if f.handle != nil {
		return f.handle(cmd)
	}
	return &runtime.Result{ExitCode: 0}, nil
}

func lastCall(f *fakeRuntime) runtime.Command {
	return f.calls[len(f.calls)-1]
}

func TestExecuteRejectsNonK3dCommands(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	_, err := r.Execute(context.Background(), runtime.Command{Name: "docker"})
	if err == nil {
		t.Fatal("Execute() error = nil, want an error for a non-k3d command")
	}
	if len(fake.calls) != 0 {
		t.Errorf("underlying runtime was called %d times, want 0", len(fake.calls))
	}
}

func TestExecutePassesThroughK3dCommands(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if _, err := r.Execute(context.Background(), runtime.Command{Name: "k3d", Args: []string{"version"}}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("underlying runtime was called %d times, want 1", len(fake.calls))
	}
}

func TestCreateCluster(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.CreateCluster(context.Background(), "demo"); err != nil {
		t.Fatalf("CreateCluster() error = %v", err)
	}

	got := lastCall(fake)
	if got.Name != "k3d" {
		t.Errorf("Name = %q, want %q", got.Name, "k3d")
	}
	wantArgs := []string{"cluster", "create", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("Args = %v, want %v", got.Args, wantArgs)
	}
}

func TestCreateClusterRequiresName(t *testing.T) {
	r := New(&fakeRuntime{})

	if err := r.CreateCluster(context.Background(), "  "); err == nil {
		t.Fatal("CreateCluster() error = nil, want an error for an empty name")
	}
}

func TestCreateClusterFailsOnNonZeroExit(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: "cluster already exists"}, nil
		},
	}
	r := New(fake)

	err := r.CreateCluster(context.Background(), "demo")
	if err == nil {
		t.Fatal("CreateCluster() error = nil, want an error for a nonzero exit code")
	}
	if !strings.Contains(err.Error(), "cluster already exists") {
		t.Errorf("error = %v, want it to contain stderr", err)
	}
}

func TestCreateClusterPropagatesExecuteError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return nil, wantErr
		},
	}
	r := New(fake)

	if err := r.CreateCluster(context.Background(), "demo"); !errors.Is(err, wantErr) {
		t.Fatalf("CreateCluster() error = %v, want %v", err, wantErr)
	}
}

func TestDeleteCluster(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.DeleteCluster(context.Background(), "demo"); err != nil {
		t.Fatalf("DeleteCluster() error = %v", err)
	}

	got := lastCall(fake)
	wantArgs := []string{"cluster", "delete", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("Args = %v, want %v", got.Args, wantArgs)
	}
}

func TestStartCluster(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.StartCluster(context.Background(), "demo"); err != nil {
		t.Fatalf("StartCluster() error = %v", err)
	}

	got := lastCall(fake)
	wantArgs := []string{"cluster", "start", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("Args = %v, want %v", got.Args, wantArgs)
	}
}

func TestStopCluster(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.StopCluster(context.Background(), "demo"); err != nil {
		t.Fatalf("StopCluster() error = %v", err)
	}

	got := lastCall(fake)
	wantArgs := []string{"cluster", "stop", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("Args = %v, want %v", got.Args, wantArgs)
	}
}

// realClusterListJSON is a trimmed real "k3d cluster list --output
// json" response (captured from a live k3d v5.9.0 cluster), used to
// verify ListClusters only depends on the "name" field and safely
// ignores the rest of k3d's schema.
const realClusterListJSON = `[{"name":"devlab-smoketest","network":{"name":"k3d-devlab-smoketest"},"nodes":[{"name":"k3d-devlab-smoketest-server-0","role":"server"}],"serversRunning":1,"serversCount":1,"agentsRunning":0,"agentsCount":0}]`

func TestListClustersParsesRealShape(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: realClusterListJSON}, nil
		},
	}
	r := New(fake)

	got, err := r.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("ListClusters() error = %v", err)
	}
	if len(got) != 1 || got[0] != "devlab-smoketest" {
		t.Errorf("ListClusters() = %v, want [devlab-smoketest]", got)
	}
}

func TestListClustersEmpty(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "[]"}, nil
		},
	}
	r := New(fake)

	got, err := r.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("ListClusters() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListClusters() = %v, want empty", got)
	}
}

func TestListClustersMalformedJSON(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "not json"}, nil
		},
	}
	r := New(fake)

	if _, err := r.ListClusters(context.Background()); err == nil {
		t.Fatal("ListClusters() error = nil, want an error for malformed JSON")
	}
}

func TestClusterExists(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: realClusterListJSON}, nil
		},
	}
	r := New(fake)

	exists, err := r.ClusterExists(context.Background(), "devlab-smoketest")
	if err != nil {
		t.Fatalf("ClusterExists() error = %v", err)
	}
	if !exists {
		t.Error("ClusterExists() = false, want true")
	}

	exists, err = r.ClusterExists(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("ClusterExists() error = %v", err)
	}
	if exists {
		t.Error("ClusterExists() = true, want false")
	}
}

func TestClusterExistsRequiresName(t *testing.T) {
	r := New(&fakeRuntime{})

	if _, err := r.ClusterExists(context.Background(), ""); err == nil {
		t.Fatal("ClusterExists() error = nil, want an error for an empty name")
	}
}
