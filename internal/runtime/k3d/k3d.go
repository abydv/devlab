// Package k3d implements the k3d Runtime: internal/runtime.Runtime
// constrained to the k3d binary, with convenience methods for managing
// k3d-backed Kubernetes clusters.
package k3d

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abydv/devlab/internal/runtime"
)

const binaryName = "k3d"

// Runtime executes k3d cluster operations. It is the only DevLab
// component permitted to invoke the k3d CLI.
type Runtime struct {
	exec runtime.Runtime
}

var _ runtime.Runtime = (*Runtime)(nil)

// New returns a k3d Runtime that executes commands through exec.
func New(exec runtime.Runtime) *Runtime {
	return &Runtime{exec: exec}
}

// Execute implements runtime.Runtime. Only the k3d binary may be run
// through a k3d Runtime.
func (r *Runtime) Execute(ctx context.Context, cmd runtime.Command) (*runtime.Result, error) {
	if cmd.Name != binaryName {
		return nil, fmt.Errorf("k3d: only the %q binary may be executed, got %q", binaryName, cmd.Name)
	}
	return r.exec.Execute(ctx, cmd)
}

// CreateCluster creates a new k3d cluster named name.
func (r *Runtime) CreateCluster(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "cluster", "create", name)
	return err
}

// DeleteCluster deletes the k3d cluster named name.
func (r *Runtime) DeleteCluster(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "cluster", "delete", name)
	return err
}

// StartCluster starts a previously stopped k3d cluster named name.
func (r *Runtime) StartCluster(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "cluster", "start", name)
	return err
}

// StopCluster stops the k3d cluster named name without deleting it.
func (r *Runtime) StopCluster(ctx context.Context, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	_, err := r.run(ctx, "cluster", "stop", name)
	return err
}

// ListClusters returns the names of every existing k3d cluster.
func (r *Runtime) ListClusters(ctx context.Context) ([]string, error) {
	result, err := r.run(ctx, "cluster", "list", "--output", "json")
	if err != nil {
		return nil, err
	}

	var clusters []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &clusters); err != nil {
		return nil, fmt.Errorf("k3d: parse cluster list: %w", err)
	}

	names := make([]string, 0, len(clusters))
	for _, c := range clusters {
		names = append(names, c.Name)
	}
	return names, nil
}

// ClusterExists reports whether a k3d cluster named name exists.
func (r *Runtime) ClusterExists(ctx context.Context, name string) (bool, error) {
	if err := validateName(name); err != nil {
		return false, err
	}

	clusters, err := r.ListClusters(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range clusters {
		if c == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *Runtime) run(ctx context.Context, args ...string) (*runtime.Result, error) {
	result, err := r.Execute(ctx, runtime.Command{Name: binaryName, Args: args})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("k3d: %s: exit %d: %s", strings.Join(args, " "), result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	return result, nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("k3d: cluster name is required")
	}
	return nil
}
