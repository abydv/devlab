// Package kubernetes implements the Kubernetes Service: a
// service.Service backed by a k3d cluster.
package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/runtime/k3d"
	"github.com/abydv/devlab/internal/service"
)

// Service is a Kubernetes workspace Service backed by a k3d cluster.
// Cluster lifecycle (create/start/stop/delete) goes through the k3d
// Runtime. Status and Logs are read from the cluster's server node
// container via the Docker Runtime, since k3d runs every cluster node
// as a Docker container named "k3d-<cluster>-server-0".
type Service struct {
	clusterName string
	k3d         *k3d.Runtime
	docker      *docker.Runtime
}

var _ service.Service = (*Service)(nil)

// New returns a Kubernetes Service for the k3d cluster named
// clusterName, using k3dRuntime for cluster lifecycle and
// dockerRuntime to read the server node's status and logs.
func New(k3dRuntime *k3d.Runtime, dockerRuntime *docker.Runtime, clusterName string) *Service {
	return &Service{clusterName: clusterName, k3d: k3dRuntime, docker: dockerRuntime}
}

// Create provisions a new k3d cluster.
func (s *Service) Create(ctx context.Context) error {
	return s.k3d.CreateCluster(ctx, s.clusterName)
}

// Start starts a previously stopped cluster.
func (s *Service) Start(ctx context.Context) error {
	return s.k3d.StartCluster(ctx, s.clusterName)
}

// Stop stops the cluster without deleting it.
func (s *Service) Stop(ctx context.Context) error {
	return s.k3d.StopCluster(ctx, s.clusterName)
}

// Delete permanently removes the cluster.
func (s *Service) Delete(ctx context.Context) error {
	return s.k3d.DeleteCluster(ctx, s.clusterName)
}

// Reset discards the cluster and recreates it from scratch. k3d has no
// native "reset cluster" operation, so this composes a delete (if the
// cluster exists) followed by a create.
func (s *Service) Reset(ctx context.Context) error {
	exists, err := s.k3d.ClusterExists(ctx, s.clusterName)
	if err != nil {
		return err
	}
	if exists {
		if err := s.k3d.DeleteCluster(ctx, s.clusterName); err != nil {
			return err
		}
	}
	return s.k3d.CreateCluster(ctx, s.clusterName)
}

// Status reports the cluster's current lifecycle state, derived from
// its server node container's Docker state.
func (s *Service) Status(ctx context.Context) (service.Status, error) {
	status, err := s.docker.ContainerStatus(ctx, s.serverNodeName())
	if err != nil {
		if errors.Is(err, docker.ErrNotFound) {
			return "", service.ErrNotFound
		}
		return "", err
	}

	switch status {
	case "running":
		return service.StatusRunning, nil
	case "exited", "created":
		return service.StatusStopped, nil
	default:
		return service.StatusError, nil
	}
}

// Logs returns the server node container's logs.
func (s *Service) Logs(ctx context.Context) (string, error) {
	logs, err := s.docker.ContainerLogs(ctx, s.serverNodeName())
	if err != nil {
		if errors.Is(err, docker.ErrNotFound) {
			return "", service.ErrNotFound
		}
		return "", err
	}
	return logs, nil
}

// Kubeconfig returns the kubeconfig YAML for the cluster, for use by
// kubectl or other Kubernetes API clients.
func (s *Service) Kubeconfig(ctx context.Context) (string, error) {
	return s.k3d.GetKubeconfig(ctx, s.clusterName)
}

func (s *Service) serverNodeName() string {
	return fmt.Sprintf("k3d-%s-server-0", s.clusterName)
}
