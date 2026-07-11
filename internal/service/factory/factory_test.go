package factory

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/abydv/devlab/internal/runtime"
	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service"
	"github.com/abydv/devlab/internal/utils"
)

// fakeExec is a runtime.Runtime test double that records every Command
// it receives.
type fakeExec struct {
	calls []runtime.Command
}

func (f *fakeExec) Execute(_ context.Context, cmd runtime.Command) (*runtime.Result, error) {
	f.calls = append(f.calls, cmd)
	return &runtime.Result{ExitCode: 0}, nil
}

func (f *fakeExec) lastCall() runtime.Command {
	return f.calls[len(f.calls)-1]
}

func newTestFactory(fake *fakeExec) *Factory {
	return New(k3d.New(fake), docker.New(fake))
}

func TestBuildKubernetes(t *testing.T) {
	fake := &fakeExec{}
	f := newTestFactory(fake)

	svc, err := f.Build(service.TypeKubernetes, "ws1", "/data/ws1")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := svc.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := fake.lastCall()
	want := []string{"cluster", "create", "devlab-ws1-kubernetes"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestBuildDocker(t *testing.T) {
	fake := &fakeExec{}
	f := newTestFactory(fake)

	svc, err := f.Build(service.TypeDocker, "ws1", "/data/ws1")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := svc.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := fake.lastCall()
	// A named Docker volume, not a host bind-mount path — see
	// dindService in factory.go for why.
	want := []string{"create", "--name", "devlab-ws1-docker", "--privileged", "-v", "devlab-ws1-docker-data:/var/lib/docker", "docker:dind"}
	if strings.Join(got.Args, " ") != strings.Join(want, " ") {
		t.Errorf("Args = %v, want %v", got.Args, want)
	}
}

func TestBuildDockerDeleteAlsoRemovesVolume(t *testing.T) {
	fake := &fakeExec{}
	f := newTestFactory(fake)

	svc, err := f.Build(service.TypeDocker, "ws1", "/data/ws1")
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := svc.Delete(context.Background()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	var verbs []string
	for _, c := range fake.calls {
		verbs = append(verbs, strings.Join(c.Args, " "))
	}
	want := []string{"rm -f devlab-ws1-docker", "volume rm -f devlab-ws1-docker-data"}
	if strings.Join(verbs, "; ") != strings.Join(want, "; ") {
		t.Errorf("call sequence = %v, want %v", verbs, want)
	}
}

func TestBuildJenkins(t *testing.T) {
	fake := &fakeExec{}
	f := newTestFactory(fake)
	dataDir := t.TempDir()

	svc, err := f.Build(service.TypeJenkins, "ws1", dataDir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if err := svc.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := fake.lastCall()
	if got.Name != "docker" || got.Args[0] != "create" || got.Args[1] != "--name" || got.Args[2] != "devlab-ws1-jenkins" {
		t.Fatalf("Args = %v, want a docker create for devlab-ws1-jenkins", got.Args)
	}

	// The host port is dynamically allocated; just verify it's a
	// well-formed port number in the "-p <port>:8080" argument.
	var hostPort string
	for i, arg := range got.Args {
		if arg == "-p" && i+1 < len(got.Args) {
			hostPort = strings.TrimSuffix(got.Args[i+1], ":8080")
		}
	}
	port, err := strconv.Atoi(hostPort)
	if err != nil || port <= 0 || port > 65535 {
		t.Errorf("host port = %q, want a valid port number", hostPort)
	}

	wantVolume := filepath.Join(dataDir, "jenkins") + ":/var/jenkins_home"
	found := false
	for i, arg := range got.Args {
		if arg == "-v" && i+1 < len(got.Args) && got.Args[i+1] == wantVolume {
			found = true
		}
	}
	if !found {
		t.Errorf("Args = %v, want a -v %s mapping", got.Args, wantVolume)
	}
}

func TestBuildUnimplementedKnownTypes(t *testing.T) {
	f := newTestFactory(&fakeExec{})

	for _, typ := range []string{service.TypeLinux, service.TypeTerraform, service.TypeAnsible} {
		if _, err := f.Build(typ, "ws1", "/data/ws1"); err == nil {
			t.Errorf("Build(%q) error = nil, want an error (no implementation yet)", typ)
		}
	}
}

// k3dMaxClusterNameLength is k3d's own hard limit, confirmed live
// (k3d v5.9.0): "Cluster name must be <= 32 characters".
const k3dMaxClusterNameLength = 32

func TestKubernetesClusterNameFitsK3dLimit(t *testing.T) {
	id, err := utils.NewID()
	if err != nil {
		t.Fatalf("utils.NewID() error = %v", err)
	}

	name := fmt.Sprintf("devlab-%s-%s", id, service.TypeKubernetes)
	if len(name) > k3dMaxClusterNameLength {
		t.Errorf("cluster name %q is %d characters, want <= %d (k3d's limit)", name, len(name), k3dMaxClusterNameLength)
	}
}

func TestBuildUnknownType(t *testing.T) {
	f := newTestFactory(&fakeExec{})

	if _, err := f.Build("not-a-real-type", "ws1", "/data/ws1"); err == nil {
		t.Fatal("Build() error = nil, want an error for an unknown service type")
	}
}
