// Package config resolves DevLab's filesystem locations and runtime
// settings so no other package hardcodes a path or a port.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	envHomeDir        = "DEVLAB_HOME"
	envListenAddr     = "DEVLAB_LISTEN_ADDR"
	workspacesSubdir  = "workspaces"
	templatesSubdir   = "templates"
	databaseFile      = "devlab.db"
	defaultListenAddr = ":8080"
)

// Config holds the resolved filesystem layout and runtime settings for
// a DevLab instance.
type Config struct {
	// HomeDir is the root directory DevLab stores its state under.
	HomeDir string
	// WorkspacesDir is where every Workspace's on-disk data lives.
	WorkspacesDir string
	// TemplatesDir is where Template definitions are loaded from.
	TemplatesDir string
	// DatabasePath is the SQLite database DevLab persists its indexes to.
	DatabasePath string
	// ListenAddr is the address the REST API server listens on.
	ListenAddr string
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

	listenAddr := os.Getenv(envListenAddr)
	if listenAddr == "" {
		listenAddr = defaultListenAddr
	}

	return &Config{
		HomeDir:       home,
		WorkspacesDir: filepath.Join(home, workspacesSubdir),
		TemplatesDir:  filepath.Join(home, templatesSubdir),
		DatabasePath:  filepath.Join(home, databaseFile),
		ListenAddr:    listenAddr,
	}, nil
}
