package docker

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
	dockerruntime "github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/service"
)

// fakeExec is a runtime.Runtime test double that records every
// Command it receives and lets tests script the response.
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

func lastCall(f *fakeExec) runtime.Command {
	return f.calls[len(f.calls)-1]
}

func testSpec() dockerruntime.ContainerSpec {
	return dockerruntime.ContainerSpec{
		Name:  "devlab-test",
		Image: "busybox:stable",
	}
}

func newTestService(fake *fakeExec, spec dockerruntime.ContainerSpec) *Service {
	return New(dockerruntime.New(fake), spec)
}

func TestCreate(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(fake, testSpec())

	if err := s.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"create", "--name", "devlab-test", "busybox:stable"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestStartAndStop(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(fake, testSpec())

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if len(fake.calls) != 2 {
		t.Fatalf("calls = %v, want 2 calls", fake.calls)
	}
	if strings.Join(fake.calls[0].Args, " ") != "start devlab-test" {
		t.Errorf("first call = %v, want start devlab-test", fake.calls[0].Args)
	}
	if strings.Join(fake.calls[1].Args, " ") != "stop devlab-test" {
		t.Errorf("second call = %v, want stop devlab-test", fake.calls[1].Args)
	}
}

func TestDelete(t *testing.T) {
	fake := &fakeExec{}
	s := newTestService(fake, testSpec())

	if err := s.Delete(context.Background()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"rm", "-f", "devlab-test"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestResetWhenContainerExists(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if strings.Join(cmd.Args, " ") == "inspect --format {{.State.Status}} devlab-test" {
				return &runtime.Result{ExitCode: 0, Stdout: "running\n"}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	s := newTestService(fake, testSpec())

	if err := s.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	var verbs []string
	for _, c := range fake.calls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	want := []string{
		"inspect --format {{.State.Status}} devlab-test",
		"rm -f devlab-test",
		"create --name devlab-test busybox:stable",
	}
	if strings.Join(verbs, "; ") != strings.Join(want, "; ") {
		t.Errorf("call sequence = %v, want %v", verbs, want)
	}
}

func TestResetWhenContainerDoesNotExist(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			if strings.Join(cmd.Args, " ") == "inspect --format {{.State.Status}} devlab-test" {
				return &runtime.Result{ExitCode: 1, Stderr: "Error response from daemon: No such container: devlab-test"}, nil
			}
			return &runtime.Result{ExitCode: 0}, nil
		},
	}
	s := newTestService(fake, testSpec())

	if err := s.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	var verbs []string
	for _, c := range fake.calls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	want := []string{
		"inspect --format {{.State.Status}} devlab-test",
		"create --name devlab-test busybox:stable",
	}
	if strings.Join(verbs, "; ") != strings.Join(want, "; ") {
		t.Errorf("call sequence = %v, want %v (no rm call expected)", verbs, want)
	}
}

func TestStatusMapping(t *testing.T) {
	tests := []struct {
		dockerState string
		want        service.Status
	}{
		{"created", service.StatusCreated},
		{"running", service.StatusRunning},
		{"exited", service.StatusStopped},
		{"paused", service.StatusError},
		{"restarting", service.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.dockerState, func(t *testing.T) {
			fake := &fakeExec{
				handle: func(cmd runtime.Command) (*runtime.Result, error) {
					return &runtime.Result{ExitCode: 0, Stdout: tt.dockerState + "\n"}, nil
				},
			}
			s := newTestService(fake, testSpec())

			got, err := s.Status(context.Background())
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusNotFound(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: "error: no such object: devlab-test"}, nil
		},
	}
	s := newTestService(fake, testSpec())

	if _, err := s.Status(context.Background()); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("Status() error = %v, want service.ErrNotFound", err)
	}
}

func TestLogs(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "hello\n"}, nil
		},
	}
	s := newTestService(fake, testSpec())

	logs, err := s.Logs(context.Background())
	if err != nil {
		t.Fatalf("Logs() error = %v", err)
	}
	if logs != "hello\n" {
		t.Errorf("Logs() = %q, want %q", logs, "hello\n")
	}

	got := lastCall(fake)
	want := []string{"logs", "devlab-test"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestLogsNotFound(t *testing.T) {
	fake := &fakeExec{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: "Error response from daemon: No such container: devlab-test"}, nil
		},
	}
	s := newTestService(fake, testSpec())

	if _, err := s.Logs(context.Background()); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("Logs() error = %v, want service.ErrNotFound", err)
	}
}

func TestServiceSatisfiesInterface(t *testing.T) {
	var _ service.Service = (*Service)(nil)
}
