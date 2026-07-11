// Package docker implements the Docker Service: a service.Service
// backed by a single Docker container.
package docker

import (
	"context"
	"errors"

	dockerruntime "github.com/abydv/devlab/internal/runtime/docker"
	"github.com/abydv/devlab/internal/service"
)

// Service is a Docker workspace Service: a single container run from
// spec.
type Service struct {
	spec    dockerruntime.ContainerSpec
	runtime *dockerruntime.Runtime
}

var _ service.Service = (*Service)(nil)

// New returns a Docker Service that creates and manages a container
// from spec via runtime.
func New(runtime *dockerruntime.Runtime, spec dockerruntime.ContainerSpec) *Service {
	return &Service{spec: spec, runtime: runtime}
}

// Create creates the container without starting it.
func (s *Service) Create(ctx context.Context) error {
	return s.runtime.CreateContainer(ctx, s.spec)
}

// Start starts the container.
func (s *Service) Start(ctx context.Context) error {
	return s.runtime.StartContainer(ctx, s.spec.Name)
}

// Stop stops the container without removing it.
func (s *Service) Stop(ctx context.Context) error {
	return s.runtime.StopContainer(ctx, s.spec.Name)
}

// Delete permanently removes the container.
func (s *Service) Delete(ctx context.Context) error {
	return s.runtime.RemoveContainer(ctx, s.spec.Name)
}

// Reset discards the container and recreates it from spec.
func (s *Service) Reset(ctx context.Context) error {
	exists, err := s.runtime.ContainerExists(ctx, s.spec.Name)
	if err != nil {
		return err
	}
	if exists {
		if err := s.runtime.RemoveContainer(ctx, s.spec.Name); err != nil {
			return err
		}
	}
	return s.runtime.CreateContainer(ctx, s.spec)
}

// Status reports the container's current lifecycle state.
func (s *Service) Status(ctx context.Context) (service.Status, error) {
	status, err := s.runtime.ContainerStatus(ctx, s.spec.Name)
	if err != nil {
		if errors.Is(err, dockerruntime.ErrNotFound) {
			return "", service.ErrNotFound
		}
		return "", err
	}

	switch status {
	case "created":
		return service.StatusCreated, nil
	case "running":
		return service.StatusRunning, nil
	case "exited":
		return service.StatusStopped, nil
	default:
		return service.StatusError, nil
	}
}

// Logs returns the container's logs.
func (s *Service) Logs(ctx context.Context) (string, error) {
	logs, err := s.runtime.ContainerLogs(ctx, s.spec.Name)
	if err != nil {
		if errors.Is(err, dockerruntime.ErrNotFound) {
			return "", service.ErrNotFound
		}
		return "", err
	}
	return logs, nil
}
