# DevLab Architecture

## Mission

DevLab is a self-hosted Personal DevOps Workspace Platform for Ubuntu
Desktop. It is built for personal use first, but the architecture must
support future multi-user expansion without a redesign.

## Core Principles

- Everything is a **Workspace**.
- A Workspace contains one or more **Services**.
- Services use a **Runtime**.
- Only the Runtime may execute operating system commands.
- The **Engine** orchestrates Workspaces.
- The Frontend never contains business logic.

## System Flow

```
CLI → REST API → Engine → Workspace Manager → Services → Runtime → Docker / k3d / Shell
```

- **CLI**: user-facing command entrypoint (`cmd/devlab`).
- **REST API**: HTTP surface (Fiber) exposed to the CLI and the web
  frontend. Contains no business logic — it validates and delegates.
- **Engine**: orchestrates workspace lifecycle operations and coordinates
  the Workspace Manager.
- **Workspace Manager**: owns Workspace state and delegates operations to
  the Services attached to a Workspace.
- **Services**: implement a fixed lifecycle contract (Create, Start, Stop,
  Reset, Delete, Status, Logs). Services never execute OS commands
  directly — they call into a Runtime.
- **Runtime**: the only layer permitted to execute operating system
  commands (via Shell, Docker, or k3d).

## Tech Stack

**Backend**
- Go
- Fiber (HTTP framework)
- SQLite (persistence)

**Frontend**
- Next.js
- React
- TypeScript
- TailwindCSS
- shadcn/ui

**Runtime**
- Docker
- k3d

## Repository Structure

```
cmd/devlab/          CLI entrypoint / process bootstrap
internal/engine/     Workspace orchestration
internal/workspace/  Workspace domain model and manager
internal/service/    Service lifecycle contract and implementations
internal/runtime/    Runtime contract and implementations
  shell/             Shell runtime
  docker/            Docker runtime
  k3d/                k3d runtime
internal/template/   Workspace/service templates
internal/storage/    SQLite persistence layer
internal/config/     Configuration loading
internal/utils/      Shared internal utilities
pkg/                 Code intended for external/reusable consumption
api/                 REST API layer (Fiber handlers, routes, DTOs)
web/                 Next.js frontend application
templates/           On-disk template definitions (data, not code)
workspaces/          Runtime workspace data (workspace.json, logs/, data/, cache/)
scripts/             Developer and operational scripts
docs/                Project documentation (source of truth)
```

`internal/` packages are private to the module. `pkg/` is reserved for
code that is safe and intended for external reuse. Business logic lives
in `internal/`; `api/` and `web/` are thin delivery layers.

## Workspace Model

A Workspace has:

- ID
- Name
- Description
- Template
- Services
- Status
- CreatedAt
- UpdatedAt

Each workspace owns, on disk, under `<WorkspacesDir>/<id>/`:

```
workspace.json
logs/
data/
cache/
```

`internal/workspace.Manager` owns this layout (Create/Get/List/Delete).
`WorkspacesDir` is resolved by `internal/config` (via `DEVLAB_HOME`, see
ADR-0006) — never hardcoded. `internal/engine.Engine` sits above the
Manager and is the only thing the future REST API calls into, per the
CLI → REST API → Engine → Workspace Manager flow.

`workspace.json` is the source of truth for a Workspace's full data.
`internal/storage` (see Storage section below) additionally opens a
SQLite database DevLab's own code indexes a summary of each Workspace
into, used for the case-insensitive name-uniqueness check on `Create`
and the ordered ID list `List` reads from — never as a second copy of
the full record (ADR-0011).

## Template Engine

Templates are data, not code: declarative `*.json` definitions under
`templates/`, each naming the Services a Workspace created from that
Template should have.

```json
{
  "name": "kubernetes",
  "description": "A single-node Kubernetes workspace backed by k3d.",
  "services": ["kubernetes"]
}
```

`internal/template.Registry` loads and validates these definitions
(`Load`/`Get`/`List`) from `TemplatesDir` (resolved by
`internal/config`, see ADR-0006's sibling for `WorkspacesDir`). It
enforces a required, unique name and at least one Service — it does
not validate Service names against a catalog yet (see ADR-0009).

`internal/engine.Engine.CreateWorkspace` resolves a Workspace's
`Services` from its named Template at creation time; `internal/workspace`
itself stays unaware of Templates (see ADR-0008).

## Storage

`internal/storage.Open(path string) (*sql.DB, error)` opens (creating if
necessary) DevLab's SQLite database at `config.DatabasePath`
(`<HomeDir>/devlab.db`), using the pure-Go `modernc.org/sqlite` driver
so `go build` never requires a C toolchain (ADR-0010). The package has
no knowledge of any domain type — schema ownership stays with the
package that owns the data (e.g. `internal/workspace` creates and
queries its own `workspaces` table). This keeps `internal/storage` a
shared low-level utility, alongside `internal/config` and
`internal/utils`, rather than a data-access layer other domain
packages would have to route through.

## Service Contract

Every Service implementation provides:

- `Create`
- `Start`
- `Stop`
- `Reset`
- `Delete`
- `Status`
- `Logs`

Planned service implementations: Kubernetes, Docker, Jenkins, Linux,
Terraform, Ansible.

## Runtime Contract

Only the Runtime layer executes operating system commands. Services must
never shell out directly.

Planned runtime implementations: Shell Runtime, Docker Runtime, k3d
Runtime.

## Design Rules

- Use interfaces only when multiple implementations are expected
  (e.g. `Service`, `Runtime`).
- Prefer composition over inheritance.
- Keep packages focused on a single responsibility.
- No hardcoded paths — configuration is sourced through `internal/config`.
- Never redesign this architecture without explicit approval.
