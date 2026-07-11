package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsToWorkingDirectory(t *testing.T) {
	t.Setenv(envHomeDir, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HomeDir == "" {
		t.Fatal("HomeDir is empty")
	}
	want := filepath.Join(cfg.HomeDir, workspacesSubdir)
	if cfg.WorkspacesDir != want {
		t.Errorf("WorkspacesDir = %q, want %q", cfg.WorkspacesDir, want)
	}
}

func TestLoadHonorsDevlabHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envHomeDir, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HomeDir != dir {
		t.Errorf("HomeDir = %q, want %q", cfg.HomeDir, dir)
	}
	want := filepath.Join(dir, workspacesSubdir)
	if cfg.WorkspacesDir != want {
		t.Errorf("WorkspacesDir = %q, want %q", cfg.WorkspacesDir, want)
	}
}
