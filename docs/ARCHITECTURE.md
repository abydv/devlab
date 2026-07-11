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
  kubernetes/        Kubernetes Service (k3d-backed)
  docker/            Docker Service
  jenkins/           Jenkins Service
  factory/           Builds a Service from a type name + Workspace context
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

`internal/workspace.Manager` owns this layout (Create/Get/List/Delete/
SetStatus/DataDir). `WorkspacesDir` is resolved by `internal/config`
(via `DEVLAB_HOME`, see ADR-0006) — never hardcoded. `internal/engine.Engine`
sits above the Manager and is the only thing the future REST API calls
into, per the CLI → REST API → Engine → Workspace Manager flow.

A Workspace's `ID` (from `internal/utils.NewID`) is 12 hex characters,
not a more conventional 32 — short enough that
`devlab-<id>-kubernetes`, the naming scheme Service resources use (see
Workspace Lifecycle below), fits within k3d's 32-character cluster
name limit (ADR-0023).

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
enforces a required, unique name, at least one Service, and — since
Sprint 10 — that every named Service is a recognized type
(`service.IsKnownType`, `ErrUnknownService`; ADR-0020). "Recognized"
is broader than "implemented": all six Service Rules examples validate
successfully even though only three have a concrete `service.Service`
today.

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

`internal/service` defines the interface every Service implementation
satisfies:

```go
type Service interface {
    Create(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Reset(ctx context.Context) error
    Delete(ctx context.Context) error
    Status(ctx context.Context) (Status, error)
    Logs(ctx context.Context) (string, error)
}
```

`Status` is its own type (created/running/stopped/error) — deliberately
not `workspace.Status` — keeping `internal/service` decoupled from
`internal/workspace`, the same direction as the workspace/template
decoupling (ADR-0008). `ErrNotFound` is a shared sentinel for
`Status`/`Logs` when the underlying resource doesn't exist yet.

`internal/service/kubernetes` is the first implementation, backed by a
k3d cluster. It holds both a `*k3d.Runtime` (cluster lifecycle) and a
`*docker.Runtime` (`Status`/`Logs`, read from the cluster's server node
container `k3d-<cluster>-server-0`, since k3d nodes are themselves
Docker containers) — a Service is not limited to one Runtime instance
(ADR-0016). `Reset` composes `ClusterExists`/`DeleteCluster`/
`CreateCluster` since k3d has no native reset operation.

`internal/service/docker` is the second implementation: a single
container, needing only `*docker.Runtime` since every Service Rules
verb already has a direct Docker Runtime method. It takes a
`docker.ContainerSpec` directly as its construction parameter rather
than a parallel struct (ADR-0018). Its `Status` mapping is the first to
use `service.StatusCreated`, since a Docker container (unlike a k3d
cluster) genuinely has a distinct "created but not started" state.

`internal/service/jenkins` is the third implementation: it embeds
`*servicedocker.Service` configured with Jenkins-specific defaults
(image, port, `/var/jenkins_home` volume) rather than reimplementing
container lifecycle logic — `Start`/`Stop`/`Reset`/`Delete`/`Status`/
`Logs` are inherited via Go method promotion, and only `Create` is
overridden (to prepare the host data directory) plus one addition,
`InitialAdminPassword`, reading directly from the bind-mounted host
path rather than via a new Runtime capability (ADR-0019).

All three planned Service implementations are complete: Kubernetes,
Docker, Jenkins. Linux, Terraform, and Ansible are named as Service
Rules examples in CLAUDE.md but have no dedicated roadmap sprint.

`internal/service/factory.Factory.Build(serviceType, workspaceID,
dataDir)` constructs the right concrete Service for a type name,
naming its underlying resource `devlab-<workspaceID>-<serviceType>`.
The generic `"docker"` type builds a privileged Docker-in-Docker
container (`docker:dind`) rather than an arbitrary user image, per an
explicit product decision (ADR-0021) — its storage is a Docker named
volume, not a host bind-mount under the Workspace's `data/`, because a
bind-mount was confirmed live to leave root-owned files an
unprivileged process cannot later remove (ADR-0022, ADR-0024). The
`"jenkins"` type allocates a free host port per Workspace
(`utils.FreePort`) so multiple Jenkins-backed Workspaces don't collide.
`Build` returns a clear error for `linux`/`terraform`/`ansible` —
recognized types with no implementation yet — rather than silently
doing nothing.

## Runtime Contract

Only the Runtime layer executes operating system commands. Services must
never shell out directly.

`internal/runtime` defines the single interface every Runtime
implementation satisfies:

```go
type Runtime interface {
    Execute(ctx context.Context, cmd Command) (*Result, error)
}
```

`Command` (`Name`, `Args`, `Dir`, `Env`) and `Result` (`ExitCode`,
`Stdout`, `Stderr`) are intentionally generic — Docker and k3d are
themselves invoked as CLI binaries, so "run a container" and "run a
shell command" are both just "execute a named program with arguments"
at this layer (ADR-0012). A non-zero `ExitCode` is not a Go `error`;
`Execute` returns an error only when the command could not be started
or run to completion (missing executable, canceled context).

`internal/runtime/shell` is the first implementation, invoking
`exec.CommandContext(ctx, cmd.Name, cmd.Args...)` directly. Arguments
are passed as a slice, never interpolated into a shell string — this,
not a command allow-list, is the injection boundary (ADR-0013).

`internal/runtime/k3d` is the second implementation. Unlike Shell
Runtime, it is composed over an injected `runtime.Runtime` (in
practice a Shell Runtime) rather than calling `os/exec` itself, and its
`Execute` rejects any `Command` whose `Name` isn't `"k3d"` — a
deliberate narrowing, in contrast to Shell Runtime's unrestricted
`Execute` (ADR-0014). It also exposes convenience methods
(`CreateCluster`, `StartCluster`, `StopCluster`, `DeleteCluster`,
`ListClusters`, `ClusterExists`) mirroring the Service Rules lifecycle
wherever a real `k3d` CLI operation exists for it.

`internal/runtime/docker` is the third implementation, shaped like k3d
Runtime (composed over an injected `runtime.Runtime`, `Execute`
restricted to the `docker` binary), but — since every Service Rules
verb has a real `docker` equivalent — exposing the full set:
`CreateContainer`, `StartContainer`, `StopContainer`,
`RemoveContainer`, `ContainerStatus`, `ContainerExists`,
`ContainerLogs`, with `ErrNotFound`/`ErrAlreadyExists` classified from
`docker`'s own CLI error text (ADR-0015).

All three planned Runtime implementations are complete: Shell Runtime,
k3d Runtime, Docker Runtime.

## Workspace Lifecycle

`internal/engine.Engine` is where Workspace, Template, Service, and
Runtime finally meet:

- `CreateWorkspace` only creates the Workspace record — it does not
  provision any Service's underlying resource (ADR-0025). Creating a
  Workspace is cheap; nothing is pulled or started until requested.
- `StartWorkspace` builds every attached Service via the Factory,
  calls `Create` on any that report `service.ErrNotFound`, then always
  calls `Start`. This is safe unconditionally because every underlying
  primitive (`k3d cluster start`, `docker start`, `k3d cluster create`,
  ...) was confirmed live to be a no-op success when the resource is
  already in the target state, not an error (ADR-0025).
- `StopWorkspace` / `ResetWorkspace` loop the same "build then call the
  matching Service method" shape.
- `DeleteWorkspace` (now `context.Context`-aware) deletes every
  Service's resource *before* removing the Workspace record, so a
  failure never leaves an orphaned cluster/container with no
  corresponding Workspace.
- `WorkspaceStatus` recomputes a Workspace-level `workspace.Status`
  from its Services' individual `service.Status` values — running only
  if at least one Service is running and none disagree, stopped if
  none are running, error on disagreement or an underlying Service
  error — and persists it via `workspace.Manager.SetStatus`.
- `WorkspaceLogs` concatenates every Service's logs, each labeled with
  its service type.

The full lifecycle (Create → Start → Status → Logs → Stop → Start →
Reset → Delete) was verified end-to-end against real `k3d`/`docker`
for both the `kubernetes` and `docker` templates before this sprint
was considered complete — see ADR-0023 and ADR-0024 for what that
caught.

## Design Rules

- Use interfaces only when multiple implementations are expected
  (e.g. `Service`, `Runtime`).
- Prefer composition over inheritance.
- Keep packages focused on a single responsibility.
- No hardcoded paths — configuration is sourced through `internal/config`.
- Never redesign this architecture without explicit approval.
