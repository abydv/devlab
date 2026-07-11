package jenkins

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
	"github.com/abydv/devlab/internal/runtime/docker"
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

func TestCreateBuildsCorrectContainerAndDataDir(t *testing.T) {
	fake := &fakeExec{}
	dataDir := filepath.Join(t.TempDir(), "jenkins-home")
	s := New(docker.New(fake), Config{Name: "devlab-jenkins", HostPort: "18080", DataDir: dataDir})

	if err := s.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	info, err := os.Stat(dataDir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("data dir is not a directory")
	}

	got := lastCall(fake)
	want := []string{
		"create", "--name", "devlab-jenkins",
		"-p", "18080:8080",
		"-v", dataDir + ":/var/jenkins_home",
		"jenkins/jenkins:lts",
	}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestCreateRequiresDataDir(t *testing.T) {
	s := New(docker.New(&fakeExec{}), Config{Name: "devlab-jenkins", HostPort: "18080"})

	if err := s.Create(context.Background()); err == nil {
		t.Fatal("Create() error = nil, want an error for a missing data directory")
	}
}

func TestStartDelegatesToDockerService(t *testing.T) {
	fake := &fakeExec{}
	s := New(docker.New(fake), Config{Name: "devlab-jenkins", HostPort: "18080", DataDir: t.TempDir()})

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	got := lastCall(fake)
	want := []string{"start", "devlab-jenkins"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestInitialAdminPassword(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "secrets"), 0o755); err != nil {
		t.Fatalf("mkdir secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, initialPasswordFile), []byte("3c95a53eb6bd42d0be4bf83d4c39fc38\n"), 0o640); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	s := New(docker.New(&fakeExec{}), Config{Name: "devlab-jenkins", HostPort: "18080", DataDir: dataDir})

	got, err := s.InitialAdminPassword()
	if err != nil {
		t.Fatalf("InitialAdminPassword() error = %v", err)
	}
	if got != "3c95a53eb6bd42d0be4bf83d4c39fc38" {
		t.Errorf("InitialAdminPassword() = %q, want %q", got, "3c95a53eb6bd42d0be4bf83d4c39fc38")
	}
}

func TestInitialAdminPasswordMissing(t *testing.T) {
	s := New(docker.New(&fakeExec{}), Config{Name: "devlab-jenkins", HostPort: "18080", DataDir: t.TempDir()})

	if _, err := s.InitialAdminPassword(); err == nil {
		t.Fatal("InitialAdminPassword() error = nil, want an error when the file doesn't exist (setup not complete)")
	}
}

func TestServiceSatisfiesInterface(t *testing.T) {
	var _ service.Service = (*Service)(nil)
}
