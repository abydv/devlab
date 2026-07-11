package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsToWorkingDirectory(t *testing.T) {
	t.Setenv(envHomeDir, "")
	t.Setenv(envListenAddr, "")

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
	wantDatabase := filepath.Join(cfg.HomeDir, databaseFile)
	if cfg.DatabasePath != wantDatabase {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, wantDatabase)
	}
	if cfg.ListenAddr != defaultListenAddr {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, defaultListenAddr)
	}
}

func TestLoadHonorsDevlabListenAddr(t *testing.T) {
	t.Setenv(envListenAddr, ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, ":9090")
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
	wantDatabase := filepath.Join(dir, databaseFile)
	if cfg.DatabasePath != wantDatabase {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, wantDatabase)
	}
}
