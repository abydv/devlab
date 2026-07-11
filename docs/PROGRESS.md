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

## Sprint 3 — Storage

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/storage`: `Open(path string) (*sql.DB, error)` — a
  domain-agnostic SQLite connection opener (pure-Go driver,
  `modernc.org/sqlite`, no cgo). Creates the parent directory if
  needed, pings the connection, enables foreign keys. Has no knowledge
  of Workspace, Template, or any other domain type.
- `internal/workspace`: added a SQLite index (`index.go`) backing the
  Manager. `workspace.json` remains the source of truth for a
  Workspace's full data (unchanged from Sprint 1, per ADR-0006); the
  index now provides the case-insensitive name-uniqueness check
  (replacing an O(n) directory scan) and the ordered ID list `List`
  materializes from. `NewManager` now takes `(rootDir string, db
  *sql.DB)` and returns `(*Manager, error)`, creating the `workspaces`
  index table on first use. `Create` and `Delete` keep the index and
  the on-disk manifest in sync, rolling back the index entry if the
  filesystem write fails.
- `internal/config`: added `DatabasePath` (`<HomeDir>/devlab.db`),
  resolved the same way as `WorkspacesDir`/`TemplatesDir`.
- Added `modernc.org/sqlite` as a direct dependency (`go.mod`/`go.sum`).
- Unit tests for `internal/storage`; updated `workspace` and `engine`
  tests to open a real temporary SQLite database, plus a new assertion
  that `List()` reflects `Delete()` via the index.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: migrating Template definitions into SQLite
(they remain on-disk JSON per Sprint 2 — Templates are static catalog
data, not runtime-created records); Runtimes and Services (Sprints
4-9).

Next: Sprint 4 — Shell Runtime (awaiting approval to start).

## Sprint 4 — Shell Runtime

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/runtime`: the shared `Runtime` contract
  (`Execute(ctx, Command) (*Result, error)`) that every Runtime
  implementation satisfies — justified now because three concrete
  implementations are explicitly planned (Shell, Docker, k3d), per the
  "use interfaces only when multiple implementations are expected"
  standard. `Command` (Name, Args, Dir, Env) and `Result` (ExitCode,
  Stdout, Stderr) are the shared vocabulary.
- `internal/runtime/shell`: the Shell Runtime, the first concrete
  implementation, backed directly by `os/exec.CommandContext`. Args are
  passed as a slice, never interpolated into a shell string, so shell
  metacharacters carry no special meaning — this is the security
  boundary against injection, not an allow-list (DevLab intentionally
  does not restrict which commands a Runtime may run; the architectural
  boundary is that only a Runtime may run them at all).
- Found and fixed a real bug during testing: `exec.CommandContext`
  reports a context-canceled/timed-out process as a normal
  `*exec.ExitError` ("signal: killed"), which would otherwise be
  misreported as an ordinary nonzero exit instead of an execution
  error. `Execute` now checks `ctx.Err()` first to distinguish the two.
- Unit tests covering stdout/stderr capture, exit codes, working
  directory, environment variables, missing executables, and context
  cancellation.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: Docker Runtime and k3d Runtime (Sprints 5-6);
wiring any Runtime into a Service (`internal/service` doesn't exist
until Sprint 7) — Shell Runtime is a fully functional, tested, but not
yet consumed library package, same as `internal/storage` was before
Sprint 3 wired it into `workspace.Manager`.

Next: Sprint 5 — k3d Runtime (awaiting approval to start).
