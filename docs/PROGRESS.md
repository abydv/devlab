# DevLab Progress Log

## Sprint 0 — Architecture

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- Full repository directory structure per `docs/ARCHITECTURE.md`.
- Go module initialized (`github.com/abydv/devlab`, Go 1.25).
- Git repository initialized (`main` branch).
- Core documentation created: `ARCHITECTURE.md`, `ROADMAP.md`,
  `PROGRESS.md`, `TASKS.md`, `DECISIONS.md`, `API.md`, `CHANGELOG.md`,
  `PROJECT_STATE.json`.
- `.gitignore`, `Makefile`, `LICENSE` (MIT), `README.md` created.
- Minimal `cmd/devlab` entrypoint added so the module builds.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

No business logic (Engine, Workspace Manager, Services, Runtimes) was
implemented in this sprint by design.

Next: Sprint 1 — Workspace Engine (awaiting approval to start).

## Sprint 1 — Workspace Engine

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/workspace`: `Workspace` domain model (ID, Name, Description,
  Template, Services, Status, CreatedAt, UpdatedAt) and `Status` enum
  (`created`, `running`, `stopped`, `error`).
- `internal/workspace.Manager`: disk-backed CRUD (`Create`, `Get`,
  `List`, `Delete`). Each Workspace is persisted as
  `<id>/workspace.json` with `logs/`, `data/`, `cache/` subdirectories,
  matching the Workspace Rules in `docs/ARCHITECTURE.md`. Enforces
  required, unique (case-insensitive) names.
- `internal/engine.Engine`: orchestration seam above the Workspace
  Manager (`CreateWorkspace`, `GetWorkspace`, `ListWorkspaces`,
  `DeleteWorkspace`), matching the CLI → REST API → Engine → Workspace
  Manager architecture.
- `internal/config`: resolves DevLab's filesystem layout
  (`DEVLAB_HOME` env var, defaulting to the working directory) so no
  package hardcodes a path.
- `internal/utils.NewID`: random unique ID generation for Workspaces.
- Full unit test coverage for `config`, `utils`, `workspace`, and
  `engine`.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope for this sprint (reserved for later sprints per
the roadmap): Template resolution (Sprint 2), SQLite persistence
(Sprint 3), Runtimes (Sprints 4-6), Services (Sprints 7-9), and
Workspace Start/Stop/Reset lifecycle operations (Sprint 10) — these
require the Service/Runtime layers that don't exist yet, so the Engine
does not expose them.

Next: Sprint 2 — Template Engine (awaiting approval to start).

## Sprint 2 — Template Engine

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/template`: `Template` domain model (Name, Description,
  Services) and `Registry`, which loads and validates `*.json`
  definitions from a directory into memory (`Load`, `Get`, `List`).
  Rejects missing names, empty service lists, and duplicate names. A
  missing templates directory loads as an empty catalog rather than
  failing.
- `templates/`: six seed Template definitions — one per Service Rules
  example in `CLAUDE.md` (kubernetes, docker, jenkins, linux, terraform,
  ansible) — each a minimal single-service template.
- `internal/config`: added `TemplatesDir`, resolved the same way as
  `WorkspacesDir` (via `DEVLAB_HOME`), so the Registry never hardcodes
  a path.
- `internal/engine.Engine`: now takes a `*template.Registry` alongside
  the Workspace Manager. `CreateWorkspace(name, description,
  templateName)` looks up the Template and resolves the Workspace's
  Services from it — callers no longer pass Services directly. Added
  `ListTemplates` / `GetTemplate` passthroughs so template discovery is
  available ahead of the CLI/API. `workspace.Manager.Create` itself is
  unchanged and still accepts an explicit services list, keeping
  `internal/workspace` decoupled from `internal/template` (see
  ADR-0008).
- Unit tests for `template.Registry` (including one that loads the
  real `templates/` seed data) and updated `engine` tests covering
  template resolution and the unknown-template error path.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: validating Template `services` entries against
a known service-type catalog (there is no such catalog yet — it belongs
to `internal/service`, starting Sprint 7); Storage/SQLite (Sprint 3);
Runtimes and Services themselves (Sprints 4-9).

Next: Sprint 3 — Storage (awaiting approval to start).
