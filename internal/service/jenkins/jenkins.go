// Package jenkins implements the Jenkins Service: a service.Service
// wrapping a Docker Service (internal/service/docker) preconfigured to
// run Jenkins with its home directory persisted to a host path.
package jenkins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/service"
	servicedocker "github.com/abydv/devlab/internal/service/docker"
)

const (
	defaultImage        = "jenkins/jenkins:lts"
	containerPort       = "8080"
	jenkinsHomePath     = "/var/jenkins_home"
	initialPasswordFile = "secrets/initialAdminPassword"
)

// Config configures a Jenkins Service instance.
type Config struct {
	// Name uniquely identifies the underlying container.
	Name string
	// HostPort is the host port Jenkins' web UI is published on.
	HostPort string
	// DataDir is a host directory persisted as Jenkins' home directory
	// (JENKINS_HOME). Created on Create if it does not already exist.
	DataDir string
}

// Service is a Jenkins workspace Service: a Jenkins container with its
// home directory persisted to a host path. It embeds a Docker Service
// for its Start/Stop/Reset/Delete/Status/Logs behavior, adding only
// what's specific to Jenkins: data directory setup on Create, and
// reading the initial admin password.
type Service struct {
	*servicedocker.Service
	dataDir string
}

var _ service.Service = (*Service)(nil)

// New returns a Jenkins Service configured by cfg.
func New(runtime *docker.Runtime, cfg Config) *Service {
	spec := docker.ContainerSpec{
		Name:  cfg.Name,
		Image: defaultImage,
		Ports: []docker.PortMapping{
			{HostPort: cfg.HostPort, ContainerPort: containerPort},
		},
		Volumes: []docker.VolumeMapping{
			{HostPath: cfg.DataDir, ContainerPath: jenkinsHomePath},
		},
	}
	return &Service{
		Service: servicedocker.New(runtime, spec),
		dataDir: cfg.DataDir,
	}
}

// Create creates the Jenkins data directory, then the container.
func (s *Service) Create(ctx context.Context) error {
	if strings.TrimSpace(s.dataDir) == "" {
		return fmt.Errorf("jenkins: data directory is required")
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return fmt.Errorf("jenkins: create data directory: %w", err)
	}
	return s.Service.Create(ctx)
}

// InitialAdminPassword returns Jenkins' initial admin password, read
// directly from its well-known location under the bind-mounted data
// directory. It is only available before Jenkins' setup wizard has
// been completed — Jenkins deletes the file afterward.
func (s *Service) InitialAdminPassword() (string, error) {
	path := filepath.Join(s.dataDir, initialPasswordFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("jenkins: read initial admin password: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
