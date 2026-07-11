# DevLab Tasks

## Sprint 0 — Architecture

- [x] Create the complete directory structure.
- [x] Initialize the Go module.
- [x] Initialize Git.
- [x] Create all documentation files under `docs/`.
- [x] Create `docs/PROJECT_STATE.json`.
- [x] Create `.gitignore`.
- [x] Create `Makefile`.
- [x] Create `LICENSE` (MIT).
- [x] Create `README.md`.
- [x] Verify the repository builds.

## Sprint 1 — Workspace Engine

- [x] Define `Workspace` domain model (`internal/workspace`).
- [x] Implement `internal/config` for path resolution (no hardcoded paths).
- [x] Implement `internal/utils.NewID` for unique Workspace IDs.
- [x] Implement `workspace.Manager`: disk-backed Create/Get/List/Delete,
      with `workspace.json`, `logs/`, `data/`, `cache/` per Workspace.
- [x] Enforce required and unique Workspace names.
- [x] Implement `internal/engine.Engine` orchestration layer above the
      Workspace Manager.
- [x] Unit tests for `config`, `utils`, `workspace`, `engine`.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 2 — Template Engine

- [x] Define `Template` domain model (`internal/template`).
- [x] Implement `template.Registry`: load/validate `*.json` definitions
      from a directory, `Get`/`List`.
- [x] Add `TemplatesDir` to `internal/config`.
- [x] Seed `templates/` with one definition per Service Rules example
      (kubernetes, docker, jenkins, linux, terraform, ansible).
- [x] Wire `internal/engine.Engine` to resolve a Workspace's Services
      from its Template on creation.
- [x] Add `Engine.ListTemplates` / `Engine.GetTemplate`.
- [x] Unit tests for `template.Registry` and updated `engine` tests.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 3 — Storage

- [x] Implement `internal/storage.Open`: domain-agnostic SQLite
      connection opener (pure-Go driver, no cgo).
- [x] Add `DatabasePath` to `internal/config`.
- [x] Add a SQLite index to `internal/workspace.Manager` (name
      uniqueness check, ordered `List`), keeping `workspace.json` as
      the source of truth for full Workspace data.
- [x] Update `NewManager` to accept a `*sql.DB` and create its schema.
- [x] Keep the index in sync on `Create`/`Delete`, rolling back on
      filesystem failure.
- [x] Add `modernc.org/sqlite` dependency; `go mod tidy`.
- [x] Unit tests for `internal/storage`; update `workspace`/`engine`
      tests to use a real SQLite database.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 4 — Shell Runtime

Not started. Awaiting approval to begin.
