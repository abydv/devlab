package template

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTemplate(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}

func TestRegistryLoadAndGet(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "kubernetes.json", `{"name":"kubernetes","description":"A Kubernetes workspace.","services":["kubernetes"]}`)
	writeTemplate(t, dir, "docker.json", `{"name":"docker","description":"A Docker workspace.","services":["docker"]}`)
	writeTemplate(t, dir, "ignored.txt", `not a template`)

	r := NewRegistry(dir)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	got, err := r.Get("kubernetes")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Description != "A Kubernetes workspace." {
		t.Errorf("Description = %q, want %q", got.Description, "A Kubernetes workspace.")
	}
	if len(got.Services) != 1 || got.Services[0] != "kubernetes" {
		t.Errorf("Services = %v, want [kubernetes]", got.Services)
	}

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("List() returned %d templates, want 2", len(list))
	}
	if list[0].Name != "docker" || list[1].Name != "kubernetes" {
		t.Errorf("List() order = [%s, %s], want [docker, kubernetes]", list[0].Name, list[1].Name)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry(t.TempDir())
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if _, err := r.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestRegistryLoadMissingDirectoryIsEmpty(t *testing.T) {
	r := NewRegistry(filepath.Join(t.TempDir(), "does-not-exist"))
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := r.List(); len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
}

func TestRegistryLoadRejectsMissingName(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "bad.json", `{"description":"no name","services":["docker"]}`)

	r := NewRegistry(dir)
	if err := r.Load(); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("Load() error = %v, want ErrNameRequired", err)
	}
}

func TestRegistryLoadRejectsNoServices(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "bad.json", `{"name":"empty","services":[]}`)

	r := NewRegistry(dir)
	if err := r.Load(); !errors.Is(err, ErrServicesRequired) {
		t.Fatalf("Load() error = %v, want ErrServicesRequired", err)
	}
}

func TestRegistryLoadRejectsDuplicateName(t *testing.T) {
	dir := t.TempDir()
	writeTemplate(t, dir, "a.json", `{"name":"dup","services":["docker"]}`)
	writeTemplate(t, dir, "b.json", `{"name":"dup","services":["jenkins"]}`)

	r := NewRegistry(dir)
	if err := r.Load(); !errors.Is(err, ErrNameExists) {
		t.Fatalf("Load() error = %v, want ErrNameExists", err)
	}
}

func TestRegistryLoadSeedTemplates(t *testing.T) {
	dir := filepath.Join("..", "..", "templates")

	r := NewRegistry(dir)
	if err := r.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	list := r.List()
	if len(list) == 0 {
		t.Fatal("List() is empty, want the seed templates shipped in templates/")
	}
	for _, tmpl := range list {
		if tmpl.Name == "" {
			t.Errorf("seed template has empty name: %+v", tmpl)
		}
		if len(tmpl.Services) == 0 {
			t.Errorf("seed template %q has no services", tmpl.Name)
		}
	}
}
