# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Sprint 3 — Storage

- Add `internal/storage.Open`: domain-agnostic SQLite connection opener
  (`modernc.org/sqlite`, pure Go, no cgo).
- Add `DatabasePath` to `internal/config`.
- Add a SQLite index to `internal/workspace.Manager`: case-insensitive
  name-uniqueness check and ordered `List`, with `workspace.json`
  remaining the source of truth for full Workspace data.
- `workspace.NewManager` now takes `(rootDir string, db *sql.DB)` and
  returns `(*Manager, error)`.
- Add `modernc.org/sqlite` dependency.
- Add unit tests for `internal/storage`; update `workspace`/`engine`
  tests to use a real temporary SQLite database.

### Sprint 2 — Template Engine

- Add `Template` domain model and `Registry` (`internal/template`):
  loads and validates `*.json` definitions from a directory.
- Seed `templates/` with kubernetes, docker, jenkins, linux, terraform,
  and ansible definitions.
- Add `TemplatesDir` to `internal/config`.
- `engine.Engine.CreateWorkspace` now resolves a Workspace's Services
  from its Template instead of taking them as a caller-supplied list.
- Add `Engine.ListTemplates` / `Engine.GetTemplate`.
- Add unit tests for `template.Registry`; update `engine` tests.

### Sprint 1 — Workspace Engine

- Add `Workspace` domain model and `Status` enum (`internal/workspace`).
- Add disk-backed `workspace.Manager` (Create, Get, List, Delete) with
  `workspace.json` manifests and `logs/`, `data/`, `cache/` directories.
- Add `internal/engine.Engine` orchestration layer above the Workspace
  Manager.
- Add `internal/config` for filesystem path resolution (`DEVLAB_HOME`).
- Add `internal/utils.NewID` for unique ID generation.
- Add unit tests for all new packages.

### Sprint 0 — Architecture

- Initialize repository directory structure.
- Initialize Go module `github.com/abydv/devlab`.
- Initialize Git repository.
- Add project documentation: ARCHITECTURE, ROADMAP, PROGRESS, TASKS,
  DECISIONS, API, CHANGELOG, PROJECT_STATE.
- Add `.gitignore`, `Makefile`, `LICENSE` (MIT), `README.md`.
- Add minimal `cmd/devlab` entrypoint to establish a buildable module.
