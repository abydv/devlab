package docker

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

// The following are real docker CLI stderr strings, captured from a
// live docker (Engine 29.6.1) instance, used so error classification
// is grounded in actual output rather than a guess.
const (
	realNoSuchContainerStderr = `Error response from daemon: No such container: devlab-nonexistent-xyz`
	realNoSuchObjectStderr    = `error: no such object: devlab-nonexistent-xyz`
	realAlreadyExistsStderr   = `Error response from daemon: Conflict. The container name "/devlab-duptest" is already in use by container "bb4a8679713a859bb7685f583ebb317e9a936b97783ae9b4c8192e92e41d2124". You have to remove (or rename) that container to be able to reuse that name.`
)

func TestExecuteRejectsNonDockerCommands(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	_, err := r.Execute(context.Background(), runtime.Command{Name: "k3d"})
	if err == nil {
		t.Fatal("Execute() error = nil, want an error for a non-docker command")
	}
	if len(fake.calls) != 0 {
		t.Errorf("underlying runtime was called %d times, want 0", len(fake.calls))
	}
}

func TestExecutePassesThroughDockerCommands(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if _, err := r.Execute(context.Background(), runtime.Command{Name: "docker", Args: []string{"version"}}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("underlying runtime was called %d times, want 1", len(fake.calls))
	}
}

func TestCreateContainer(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	spec := ContainerSpec{
		Name:  "demo",
		Image: "busybox:stable",
		Env:   []string{"FOO=bar"},
		Ports: []PortMapping{{HostPort: "18080", ContainerPort: "80"}},
		Volumes: []VolumeMapping{
			{HostPath: "/host/data", ContainerPath: "/data"},
		},
		Command: []string{"sh", "-c", "sleep 30"},
	}
	if err := r.CreateContainer(context.Background(), spec); err != nil {
		t.Fatalf("CreateContainer() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{
		"create", "--name", "demo",
		"-e", "FOO=bar",
		"-p", "18080:80",
		"-v", "/host/data:/data",
		"busybox:stable", "sh", "-c", "sleep 30",
	}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestCreateContainerRequiresNameAndImage(t *testing.T) {
	r := New(&fakeRuntime{})

	if err := r.CreateContainer(context.Background(), ContainerSpec{Image: "busybox"}); err == nil {
		t.Fatal("CreateContainer() error = nil, want an error for a missing name")
	}
	if err := r.CreateContainer(context.Background(), ContainerSpec{Name: "demo"}); err == nil {
		t.Fatal("CreateContainer() error = nil, want an error for a missing image")
	}
}

func TestCreateContainerAlreadyExists(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realAlreadyExistsStderr}, nil
		},
	}
	r := New(fake)

	err := r.CreateContainer(context.Background(), ContainerSpec{Name: "demo", Image: "busybox"})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("CreateContainer() error = %v, want ErrAlreadyExists", err)
	}
}

func TestStartContainer(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.StartContainer(context.Background(), "demo"); err != nil {
		t.Fatalf("StartContainer() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"start", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestStartContainerNotFound(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realNoSuchContainerStderr}, nil
		},
	}
	r := New(fake)

	if err := r.StartContainer(context.Background(), "demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("StartContainer() error = %v, want ErrNotFound", err)
	}
}

func TestStopContainer(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.StopContainer(context.Background(), "demo"); err != nil {
		t.Fatalf("StopContainer() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"stop", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestStopContainerNotFound(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realNoSuchContainerStderr}, nil
		},
	}
	r := New(fake)

	if err := r.StopContainer(context.Background(), "demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("StopContainer() error = %v, want ErrNotFound", err)
	}
}

func TestRemoveContainer(t *testing.T) {
	fake := &fakeRuntime{}
	r := New(fake)

	if err := r.RemoveContainer(context.Background(), "demo"); err != nil {
		t.Fatalf("RemoveContainer() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"rm", "-f", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestRemoveContainerAlreadyAbsentIsNotAnError(t *testing.T) {
	// Real "docker rm -f" on a nonexistent container exits 0.
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stderr: realNoSuchContainerStderr}, nil
		},
	}
	r := New(fake)

	if err := r.RemoveContainer(context.Background(), "demo"); err != nil {
		t.Fatalf("RemoveContainer() error = %v, want nil", err)
	}
}

func TestContainerStatus(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "running\n"}, nil
		},
	}
	r := New(fake)

	status, err := r.ContainerStatus(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ContainerStatus() error = %v", err)
	}
	if status != "running" {
		t.Errorf("ContainerStatus() = %q, want %q", status, "running")
	}

	got := lastCall(fake)
	want := []string{"inspect", "--format", "{{.State.Status}}", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestContainerStatusNotFound(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realNoSuchObjectStderr}, nil
		},
	}
	r := New(fake)

	if _, err := r.ContainerStatus(context.Background(), "demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ContainerStatus() error = %v, want ErrNotFound", err)
	}
}

func TestContainerExists(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "exited\n"}, nil
		},
	}
	r := New(fake)

	exists, err := r.ContainerExists(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ContainerExists() error = %v", err)
	}
	if !exists {
		t.Error("ContainerExists() = false, want true")
	}
}

func TestContainerExistsFalseWhenNotFound(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realNoSuchObjectStderr}, nil
		},
	}
	r := New(fake)

	exists, err := r.ContainerExists(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ContainerExists() error = %v", err)
	}
	if exists {
		t.Error("ContainerExists() = true, want false")
	}
}

func TestContainerLogs(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 0, Stdout: "hello\n", Stderr: ""}, nil
		},
	}
	r := New(fake)

	logs, err := r.ContainerLogs(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ContainerLogs() error = %v", err)
	}
	if logs != "hello\n" {
		t.Errorf("ContainerLogs() = %q, want %q", logs, "hello\n")
	}

	got := lastCall(fake)
	want := []string{"logs", "demo"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestContainerLogsNotFound(t *testing.T) {
	fake := &fakeRuntime{
		handle: func(cmd runtime.Command) (*runtime.Result, error) {
			return &runtime.Result{ExitCode: 1, Stderr: realNoSuchContainerStderr}, nil
		},
	}
	r := New(fake)

	if _, err := r.ContainerLogs(context.Background(), "demo"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ContainerLogs() error = %v, want ErrNotFound", err)
	}
}

func TestValidationRequiresName(t *testing.T) {
	r := New(&fakeRuntime{})
	ctx := context.Background()

	if err := r.StartContainer(ctx, ""); err == nil {
		t.Error("StartContainer() error = nil, want an error for an empty name")
	}
	if err := r.StopContainer(ctx, ""); err == nil {
		t.Error("StopContainer() error = nil, want an error for an empty name")
	}
	if err := r.RemoveContainer(ctx, ""); err == nil {
		t.Error("RemoveContainer() error = nil, want an error for an empty name")
	}
	if _, err := r.ContainerStatus(ctx, ""); err == nil {
		t.Error("ContainerStatus() error = nil, want an error for an empty name")
	}
	if _, err := r.ContainerLogs(ctx, ""); err == nil {
		t.Error("ContainerLogs() error = nil, want an error for an empty name")
	}
}
