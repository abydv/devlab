package kubernetes

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service"
)

// fakeExec is a runtime.Runtime test double shared by the k3d and
// Docker Runtimes a kubernetes.Service composes over. It routes on the
// underlying binary (cmd.Name) so a single fake can script both.
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

func newTestService(t *testing.T, fake *fakeExec) *Service {
	t.Helper()
	return New(k3d.New(fake), docker.New(fake), "test-cluster")
}

func TestCreate(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(t, fake)

	if err := s.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	calls := fake.callsFor("k3d")
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "cluster create test-cluster" {
		t.Errorf("k3d calls = %v, want [cluster create test-cluster]", calls)
	}
}

func TestStartAndStop(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(t, fake)

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	calls := fake.callsFor("k3d")
	if len(calls) != 2 {
		t.Fatalf("k3d calls = %v, want 2 calls", calls)
	}
	if strings.Join(calls[0].Args, " ") != "cluster start test-cluster" {
		t.Errorf("first call = %v, want cluster start test-cluster", calls[0].Args)
	}
	if strings.Join(calls[1].Args, " ") != "cluster stop test-cluster" {
		t.Errorf("second call = %v, want cluster stop test-cluster", calls[1].Args)
	}
}

func TestDelete(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(t, fake)

	if err := s.Delete(context.Background()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	calls := fake.callsFor("k3d")
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "cluster delete test-cluster" {
		t.Errorf("k3d calls = %v, want [cluster delete test-cluster]", calls)
	}
}

func TestResetWhenClusterExists(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if cmd.Name == "k3d" && strings.Join(cmd.Args, " ") == "cluster list --output json" {
				return &runtime.Result{ExitCode: 0, Stdout: `[{"name":"test-cluster"}]`}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	s := newTestService(t, fake)

	if err := s.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	calls := fake.callsFor("k3d")
	var verbs []string
	for _, c := range calls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	want := []string{"cluster list --output json", "cluster delete test-cluster", "cluster create test-cluster"}
	if strings.Join(verbs, "; ") != strings.Join(want, "; ") {
		t.Errorf("k3d call sequence = %v, want %v", verbs, want)
	}
}

func TestResetWhenClusterDoesNotExist(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if cmd.Name == "k3d" && strings.Join(cmd.Args, " ") == "cluster list --output json" {
				return &runtime.Result{ExitCode: 0, Stdout: `[]`}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	s := newTestService(t, fake)

	if err := s.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	calls := fake.callsFor("k3d")
	var verbs []string
	for _, c := range calls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	want := []string{"cluster list --output json", "cluster create test-cluster"}
	if strings.Join(verbs, "; ") != strings.Join(want, "; ") {
		t.Errorf("k3d call sequence = %v, want %v (no delete call expected)", verbs, want)
	}
}

func TestStatusRunning(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "running\n"}, nil
		},
	}
	s := newTestService(t, fake)

	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != service.StatusRunning {
		t.Errorf("Status() = %q, want %q", status, service.StatusRunning)
	}

	calls := fake.callsFor("docker")
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "inspect --format {{.State.Status}} k3d-test-cluster-server-0" {
		t.Errorf("docker calls = %v, want an inspect of the server node", calls)
	}
}

func TestStatusStopped(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "exited\n"}, nil
		},
	}
	s := newTestService(t, fake)

	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != service.StatusStopped {
		t.Errorf("Status() = %q, want %q", status, service.StatusStopped)
	}
}

func TestStatusUnknownDockerStateMapsToError(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "restarting\n"}, nil
		},
	}
	s := newTestService(t, fake)

	status, err := s.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != service.StatusError {
		t.Errorf("Status() = %q, want %q", status, service.StatusError)
	}
}

func TestStatusNotFound(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: "Error response from daemon: No such container: k3d-test-cluster-server-0"}, nil
		},
	}
	s := newTestService(t, fake)

	if _, err := s.Status(context.Background()); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("Status() error = %v, want service.ErrNotFound", err)
	}
}

func TestLogs(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "server log line\n"}, nil
		},
	}
	s := newTestService(t, fake)

	logs, err := s.Logs(context.Background())
	if err != nil {
		t.Fatalf("Logs() error = %v", err)
	}
	if logs != "server log line\n" {
		t.Errorf("Logs() = %q, want %q", logs, "server log line\n")
	}

	calls := fake.callsFor("docker")
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "logs k3d-test-cluster-server-0" {
		t.Errorf("docker calls = %v, want logs of the server node", calls)
	}
}

func TestLogsNotFound(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: "Error response from daemon: No such container: k3d-test-cluster-server-0"}, nil
		},
	}
	s := newTestService(t, fake)

	if _, err := s.Logs(context.Background()); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("Logs() error = %v, want service.ErrNotFound", err)
	}
}

func TestKubeconfig(t *testing.T) {
	const kubeconfigYAML = "apiVersion: v1\nkind: Config\n"
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: kubeconfigYAML}, nil
		},
	}
	s := newTestService(t, fake)

	got, err := s.Kubeconfig(context.Background())
	if err != nil {
		t.Fatalf("Kubeconfig() error = %v", err)
	}
	if got != kubeconfigYAML {
		t.Errorf("Kubeconfig() = %q, want %q", got, kubeconfigYAML)
	}

	calls := fake.callsFor("k3d")
	if len(calls) != 1 || strings.Join(calls[0].Args, " ") != "kubeconfig get test-cluster" {
		t.Errorf("k3d calls = %v, want [kubeconfig get test-cluster]", calls)
	}
}

func TestServiceSatisfiesInterface(t *testing.T) {
	var _ service.Service = (*Service)(nil)
}
