// Package factory constructs the concrete service.Service
// implementation for a Workspace's service type, scoped to that
// Workspace's ID and on-disk data directory.
package factory

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service"
	servicedocker "github.com/abydv/devlab/internal/service/docker"
	"github.com/abydv/devlab/internal/service/jenkins"
	"github.com/abydv/devlab/internal/service/kubernetes"
	"github.com/abydv/devlab/internal/utils"
)

const dindImage = "docker:dind"

// Factory builds service.Service instances from a service type
// identifier (see service.KnownTypes), backed by shared Runtimes.
type Factory struct {
	k3d    *k3d.Runtime
	docker *docker.Runtime
}

// New returns a Factory that builds Services using the given Runtimes.
func New(k3dRuntime *k3d.Runtime, dockerRuntime *docker.Runtime) *Factory {
	return &Factory{k3d: k3dRuntime, docker: dockerRuntime}
}

// Build constructs the service.Service for serviceType, scoped to the
// Workspace identified by workspaceID with on-disk data at dataDir.
// serviceType must be one of service.KnownTypes; not every known type
// has an implementation yet.
func (f *Factory) Build(serviceType, workspaceID, dataDir string) (service.Service, error) {
	name := fmt.Sprintf("devlab-%s-%s", workspaceID, serviceType)

	switch serviceType {
	case service.TypeKubernetes:
		return kubernetes.New(f.k3d, f.docker, name), nil

	case service.TypeDocker:
		// Docker-in-Docker's storage tree contains files owned by the
		// container's root user. A Docker named volume — removed by the
		// daemon itself via RemoveVolume, not by our unprivileged
		// process walking the host filesystem — sidesteps that; a host
		// bind-mount under dataDir was tried and confirmed (live) to
		// leave permission-denied files behind on workspace deletion.
		volumeName := name + "-data"
		base := servicedocker.New(f.docker, docker.ContainerSpec{
			Name:       name,
			Image:      dindImage,
			Privileged: true,
			Volumes: []docker.VolumeMapping{
				{HostPath: volumeName, ContainerPath: "/var/lib/docker"},
			},
		})
		return &dindService{Service: base, runtime: f.docker, volumeName: volumeName}, nil

	case service.TypeJenkins:
		port, err := utils.FreePort()
		if err != nil {
			return nil, fmt.Errorf("service: allocate port for jenkins service: %w", err)
		}
		return jenkins.New(f.docker, jenkins.Config{
			Name:     name,
			HostPort: strconv.Itoa(port),
			DataDir:  filepath.Join(dataDir, service.TypeJenkins),
		}), nil

	case service.TypeLinux, service.TypeTerraform, service.TypeAnsible:
		return nil, fmt.Errorf("service: %q is a recognized service type but has no implementation yet", serviceType)

	default:
		return nil, fmt.Errorf("service: unknown service type %q", serviceType)
	}
}

// dindService wraps a Docker Service to also remove its named Docker
// volume on Delete. Docker itself owns and removes named volumes, so
// this avoids the host-permission problem a bind-mounted dind data
// directory has (see Build).
type dindService struct {
	*servicedocker.Service
	runtime    *docker.Runtime
	volumeName string
}

func (s *dindService) Delete(ctx context.Context) error {
	if err := s.Service.Delete(ctx); err != nil {
		return err
	}
	return s.runtime.RemoveVolume(ctx, s.volumeName)
}
