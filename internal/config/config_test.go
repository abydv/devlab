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
	wantWorkspaces := filepath.Join(cfg.HomeDir, workspacesSubdir)
	if cfg.WorkspacesDir != wantWorkspaces {
		t.Errorf("WorkspacesDir = %q, want %q", cfg.WorkspacesDir, wantWorkspaces)
	}
	wantTemplates := filepath.Join(cfg.HomeDir, templatesSubdir)
	if cfg.TemplatesDir != wantTemplates {
		t.Errorf("TemplatesDir = %q, want %q", cfg.TemplatesDir, wantTemplates)
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
	wantWorkspaces := filepath.Join(dir, workspacesSubdir)
	if cfg.WorkspacesDir != wantWorkspaces {
		t.Errorf("WorkspacesDir = %q, want %q", cfg.WorkspacesDir, wantWorkspaces)
	}
	wantTemplates := filepath.Join(dir, templatesSubdir)
	if cfg.TemplatesDir != wantTemplates {
		t.Errorf("TemplatesDir = %q, want %q", cfg.TemplatesDir, wantTemplates)
	}
}
