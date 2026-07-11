// Package docker implements the Docker Runtime: internal/runtime.Runtime
// constrained to the docker binary, with convenience methods for
// managing containers.
package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/abydv/devlab/internal/runtime"
)

const binaryName = "docker"

var (
	// ErrNotFound is returned when a container does not exist.
	ErrNotFound = errors.New("docker: container not found")
	// ErrAlreadyExists is returned when a container name is already in use.
	ErrAlreadyExists = errors.New("docker: container name already in use")
)

// PortMapping publishes a container port on the host.
type PortMapping struct {
	HostPort      string
	ContainerPort string
}

// VolumeMapping bind-mounts a host path into the container.
type VolumeMapping struct {
	HostPath      string
	ContainerPath string
}

// ContainerSpec describes a container to create.
type ContainerSpec struct {
	Name    string
	Image   string
	Env     []string
	Ports   []PortMapping
	Volumes []VolumeMapping
	// Command optionally overrides the image's default command.
	Command []string
	// Privileged grants the container extended privileges, required by
	// images that manage their own kernel-level resources (e.g.
	// Docker-in-Docker).
	Privileged bool
}

// Runtime executes Docker container operations. It is the only DevLab
// component permitted to invoke the docker CLI.
type Runtime struct {
	exec runtime.Runtime
}

var _ runtime.Runtime = (*Runtime)(nil)

// New returns a Docker Runtime that executes commands through exec.
func New(exec runtime.Runtime) *Runtime {
	return &Runtime{exec: exec}
}

// Execute implements runtime.Runtime. Only the docker binary may be run
// through a Docker Runtime.
func (r *Runtime) Execute(ctx context.Context, cmd runtime.Command) (*runtime.Result, error) {
	if cmd.Name != binaryName {
		return nil, fmt.Errorf("docker: only the %q binary may be executed, got %q", binaryName, cmd.Name)
	}
	return r.exec.Execute(ctx, cmd)
}

// CreateContainer creates a container from spec without starting it.
func (r *Runtime) CreateContainer(ctx context.Context, spec ContainerSpec) error {
	if err := validateSpec(spec); err != nil {
		return err
	}

	args := []string{"create", "--name", spec.Name}
	if spec.Privileged {
		args = append(args, "--privileged")
	}
	for _, e := range spec.Env {
		args = append(args, "-e", e)
	}
	for _, p := range spec.Ports {
		args = append(args, "-p", p.HostPort+":"+p.ContainerPort)
	}
	for _, v := range spec.Volumes {
		args = append(args, "-v", v.HostPath+":"+v.ContainerPath)
	}
	args = append(args, spec.Image)
	args = append(args, spec.Command...)

	if _, err := r.run(ctx, args...); err != nil {
		if isAlreadyExistsError(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

// StartContainer starts a previously created or stopped container.
func (r *Runtime) StartContainer(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if _, err := r.run(ctx, "start", name); err != nil {
		if isNotFoundError(err) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// StopContainer stops a running container without removing it.
func (r *Runtime) StopContainer(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if _, err := r.run(ctx, "stop", name); err != nil {
		if isNotFoundError(err) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// RemoveContainer removes a container, stopping it first if necessary.
// Removing an already-absent container is not an error.
func (r *Runtime) RemoveContainer(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "rm", "-f", name)
	return err
}

// RemoveVolume removes a named volume. Removing an already-absent
// volume is not an error.
func (r *Runtime) RemoveVolume(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "volume", "rm", "-f", name)
	return err
}

// ContainerStatus returns the container's current state (e.g.
// "running", "exited", "created"), as reported by "docker inspect".
func (r *Runtime) ContainerStatus(ctx context.Context, name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	result, err := r.run(ctx, "inspect", "--format", "{{.State.Status}}", name)
	if err != nil {
		if isNotFoundError(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ContainerExists reports whether a container named name exists.
func (r *Runtime) ContainerExists(ctx context.Context, name string) (bool, error) {
	_, err := r.ContainerStatus(ctx, name)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, ErrNotFound):
		return false, nil
	default:
		return false, err
	}
}

// ContainerLogs returns the container's combined stdout and stderr, as
// reported by "docker logs".
func (r *Runtime) ContainerLogs(ctx context.Context, name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	result, err := r.run(ctx, "logs", name)
	if err != nil {
		if isNotFoundError(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	return result.Stdout + result.Stderr, nil
}

func (r *Runtime) run(ctx context.Context, args ...string) (*runtime.Result, error) {
	result, err := r.Execute(ctx, runtime.Command{Name: binaryName, Args: args})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("docker: %s: exit %d: %s", strings.Join(args, " "), result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return result, nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("docker: container name is required")
	}
	return nil
}

func validateSpec(spec ContainerSpec) error {
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("docker: container name is required")
	}
	if strings.TrimSpace(spec.Image) == "" {
		return fmt.Errorf("docker: image is required")
	}
	return nil
}

func isNotFoundError(err error) bool {
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "no such container") || strings.Contains(lower, "no such object")
}

func isAlreadyExistsError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already in use by container")
}
