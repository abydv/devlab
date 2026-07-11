// Package config resolves DevLab's filesystem locations so no other
// package hardcodes a path.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	envHomeDir       = "DEVLAB_HOME"
	workspacesSubdir = "workspaces"
)

// Config holds the resolved filesystem layout for a DevLab instance.
type Config struct {
	// HomeDir is the root directory DevLab stores its state under.
	HomeDir string
	// WorkspacesDir is where every Workspace's on-disk data lives.
	WorkspacesDir string
}

// Load resolves the DevLab configuration.
//
// The home directory is taken from DEVLAB_HOME if set, otherwise it
// defaults to the current working directory so a freshly cloned
// repository is usable without any setup.
func Load() (*Config, error) {
	home := os.Getenv(envHomeDir)
	if home == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("config: resolve working directory: %w", err)
		}
		home = wd
	}

	home, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("config: resolve home directory: %w", err)
	}

	return &Config{
		HomeDir:       home,
		WorkspacesDir: filepath.Join(home, workspacesSubdir),
	}, nil
}
