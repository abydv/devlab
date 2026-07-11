# DevLab Architecture Decision Records

Decisions are appended chronologically and are not rewritten after the
fact. If a decision is superseded, add a new entry that references it.

---

## ADR-0001: Go module path

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** The Go module path is `github.com/abydv/devlab`.

**Context:** Repository is being initialized from scratch with no
existing remote. The path was confirmed directly with the project owner.

---

## ADR-0002: Repository layout uses `internal/` for private packages

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `engine`, `workspace`, `service`, `runtime` (with `shell`,
`docker`, `k3d` subpackages), `template`, `storage`, `config`, and
`utils` all live under `internal/`. `pkg/` is reserved for code
explicitly intended for reuse outside this module.

**Context:** CLAUDE.md lists these package names at the top level of the
repository structure without an explicit parent. Go idiom uses
`internal/` to enforce that these packages are not importable outside
the module, which matches the "only the Runtime executes commands" and
layered-architecture constraints. `pkg/` is created empty and reserved
for future reusable code (e.g. client SDKs).

---

## ADR-0003: Runtime is the only layer permitted to execute OS commands

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** Services depend on a `Runtime` interface and never invoke
`os/exec` (or equivalent) directly. Only implementations under
`internal/runtime/` may execute shell, Docker, or k3d commands.

**Context:** Directly mandated by CLAUDE.md ("Only the Runtime may
execute operating system commands", "Never execute shell commands
directly from Services"). This is a hard architectural boundary, not a
convention — future sprints must enforce it structurally (e.g. Services
hold a `Runtime` interface, never an `exec.Cmd` constructor).

---

## ADR-0004: SQLite for persistence

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/storage` will use SQLite as the persistence
backend.

**Context:** Fixed by the CLAUDE.md tech stack. Single-file embedded
database fits a self-hosted, single-user-first desktop platform and
avoids requiring a separate database service.

---

## ADR-0005: Interfaces reserved for multi-implementation boundaries

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** Go interfaces are introduced only where multiple concrete
implementations are expected — namely `Service` (Kubernetes, Docker,
Jenkins, Linux, Terraform, Ansible, ...) and `Runtime` (Shell, Docker,
k3d). Other packages use concrete types until a second implementation is
actually required.

**Context:** Directly mandated by CLAUDE.md ("Use interfaces only when
multiple implementations are expected"). Avoids premature abstraction.

---

## ADR-0006: Workspace Manager persists to disk, not SQLite, in Sprint 1

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/workspace.Manager` persists each Workspace as
`<rootDir>/<id>/workspace.json`, with `logs/`, `data/`, `cache/`
subdirectories, rather than using SQLite.

**Context:** SQLite persistence is explicitly Sprint 3 ("Storage") on
the roadmap. The Workspace Rules section of CLAUDE.md separately
specifies that every workspace owns `workspace.json`, `logs/`, `data/`,
and `cache/` on disk — this is the on-disk manifest/data layout, not the
queryable storage layer. Implementing Storage now would jump ahead of
the approved sprint sequence. `internal/storage` (SQLite) will likely be
introduced in Sprint 3 as an index/query layer over the same on-disk
Workspaces, without changing this manifest format.

---

## ADR-0007: Engine does not expose Start/Stop/Reset in Sprint 1

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/engine.Engine` in Sprint 1 only exposes
`CreateWorkspace`, `GetWorkspace`, `ListWorkspaces`, and
`DeleteWorkspace`. It does not expose Start/Stop/Reset.

**Context:** Those operations require the Service and Runtime layers
(Sprints 4-9) and are explicitly the subject of Sprint 10 ("Workspace
Lifecycle"). Adding unimplemented or no-op lifecycle methods now would
violate the "no placeholder methods" development standard and jump
ahead of the approved sprint sequence.

---

## ADR-0008: Template resolution lives in the Engine, not in `internal/workspace`

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/workspace.Manager.Create` keeps its Sprint 1
signature — it accepts an explicit `services []string` and knows
nothing about Templates. `internal/engine.Engine.CreateWorkspace`
resolves the named Template via `template.Registry.Get`, copies its
`Services`, and passes them down to the Manager.

**Context:** Per `docs/ARCHITECTURE.md`, `internal/workspace` and
`internal/template` are both used by the Engine but should not depend
on each other — that dependency direction belongs to the orchestration
layer. This keeps each package focused (dev standard: "keep packages
focused") and means the Workspace Manager remains testable without a
Template Registry.

---

## ADR-0009: Templates do not validate Service names against a catalog

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `template.Registry.Load` validates that a Template has a
name and at least one non-empty Service entry, but does not check that
each Service entry is a recognized/implemented service type.

**Context:** No canonical list of service types exists yet —
`internal/service` (which owns the Service contract and its concrete
implementations: Kubernetes, Docker, Jenkins, Linux, Terraform,
Ansible) is not built until Sprints 7-9. Encoding a service-type allow
list inside `internal/template` now would duplicate knowledge that
rightfully belongs to `internal/service` and could drift out of sync
with it. This validation will be added once `internal/service` exists
to be the source of truth.

---

## ADR-0010: `modernc.org/sqlite` (pure Go, no cgo) as the SQLite driver

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/storage` uses `modernc.org/sqlite` via
`database/sql`, not a cgo-based driver (e.g. `mattn/go-sqlite3`).

**Context:** CLAUDE.md's tech stack fixes SQLite as the persistence
engine but not a specific driver. A cgo driver would require a C
toolchain (gcc) to be present wherever DevLab is built, which works
against "Ubuntu Desktop" self-hosted distribution simplicity —
`go build` alone should be sufficient. `modernc.org/sqlite` is a pure
Go transpilation of SQLite with no such requirement, at the cost of
somewhat higher build times and binary size, which is an acceptable
trade for a single-user desktop tool.

---

## ADR-0011: SQLite is an index over `workspace.json`, not its replacement

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/workspace.Manager` stores only a summary row
per Workspace in SQLite (`id`, `name`, `status`, `template`,
`created_at`, `updated_at`) — not `description` or `services`. `Get`
always reads the full record from `workspace.json`. `List` queries the
index for an ordered ID list, then reads each full record from disk;
an ID present in the index but missing on disk is skipped rather than
failing the whole listing. `Create` writes the index row first (so the
name-uniqueness `UNIQUE COLLATE NOCASE` constraint is the single source
of truth for uniqueness) and rolls it back if the subsequent
directory/manifest write fails; `Delete` removes the on-disk directory
before the index row.

**Context:** Confirms and implements the design anticipated in
ADR-0006. Keeping `workspace.json` authoritative avoids two competing
sources of truth for a Workspace's full data (a common source of
drift bugs); SQLite's role stays scoped to what it's good at —
constraint enforcement and ordered/filtered queries — which is exactly
what `Create`'s uniqueness check and `List`'s ordering need.

---

## ADR-0012: A single `Runtime` interface (`Execute(ctx, Command)`) for Shell, Docker, and k3d

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/runtime` defines one interface,
`Runtime.Execute(ctx context.Context, cmd Command) (*Result, error)`,
that Shell (Sprint 4), Docker (Sprint 6), and k3d (Sprint 5) Runtimes
all implement. `Command` is generic (`Name`, `Args`, `Dir`, `Env`); it
is not specialized per backend (e.g. no `ContainerName` field).

**Context:** Docker and k3d are themselves invoked as CLI binaries
(`docker ...`, `k3d ...`/`kubectl ...`); at the OS-command level, "run
a container" and "run a shell script" are both just "execute a named
program with arguments." A single generic contract lets Docker/k3d
Runtimes be implemented as thin, constrained wrappers around the same
execution primitive Shell Runtime already provides — Service
implementations only need to hold a `runtime.Runtime`, not a
type-specific interface per backend. CLAUDE.md's "use interfaces only
when multiple implementations are expected" is satisfied directly:
three concrete implementations are named in the roadmap.

---

## ADR-0013: No command allow-list; injection is prevented structurally

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/runtime/shell.Runtime` does not restrict which
executables or arguments may run. It invokes
`exec.CommandContext(ctx, cmd.Name, cmd.Args...)` directly — arguments
are passed as a slice and never interpolated into a shell string, so
shell metacharacters in `Args` carry no special meaning.

**Context:** DevLab's entire purpose is running arbitrary DevOps
tooling (`kubectl`, `docker`, `terraform`, `ansible`, ...) on the
user's own machine on their behalf, so a fixed allow-list would work
against the tool's purpose and give a false sense of restriction. The
actual security boundary CLAUDE.md mandates is architectural — "only
the Runtime may execute operating system commands," i.e. the
capability is confined to one auditable layer — not a restriction on
which commands that layer may run. Passing `Args` as a slice (never
building a shell command string) is what actually prevents injection
and is non-negotiable regardless of any future allow-list decision.

---

## ADR-0014: k3d Runtime is composed over a `runtime.Runtime`, not a second `os/exec` caller

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/runtime/k3d.New(exec runtime.Runtime) *Runtime`
takes an injected `runtime.Runtime` (in practice, a Shell Runtime) and
implements `Execute` by delegating to it, rejecting any `Command` whose
`Name` isn't `"k3d"`. It does not call `os/exec` itself.

**Context:** ADR-0012 anticipated Docker/k3d Runtimes as thin wrappers
around the same execution primitive Shell Runtime provides, rather than
each reimplementing process execution. Composition also makes k3d
Runtime trivially unit-testable with a fake `runtime.Runtime` (see
`k3d_test.go`), with no dependency on the real `k3d`/Docker binaries
being installed — important since CI/build environments won't
universally have them, unlike this development sandbox.

Unlike Shell Runtime (ADR-0013, deliberately unrestricted), k3d
Runtime's `Execute` DOES reject non-`k3d` commands. This is not a
contradiction: Shell Runtime's job is general-purpose execution, so
restricting it would work against its purpose; k3d Runtime's entire
purpose is a narrow, single-binary boundary, so the restriction is the
point. Its convenience methods (`CreateCluster`, `StartCluster`,
`StopCluster`, `DeleteCluster`, `ListClusters`, `ClusterExists`) map to
the Service Rules lifecycle only where a real `k3d` CLI command exists
for it — `Reset` (no native k3d equivalent; will be a Service-layer
composition in Sprint 10) and `Logs` (belongs to Docker Runtime,
Sprint 6, since k3d clusters run as Docker containers) are deliberately
not included.

---

## ADR-0015: Docker Runtime mirrors k3d Runtime's shape; errors classified from real CLI stderr text

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/runtime/docker.Runtime` follows the same shape
as k3d Runtime (ADR-0014): composed over an injected `runtime.Runtime`,
`Execute` rejects any `Command` whose `Name` isn't `"docker"`. Unlike
k3d Runtime, every Service Rules lifecycle verb (Create, Start, Stop,
Delete, Status, Logs) maps onto a real `docker` subcommand, so all six
are implemented as convenience methods (`CreateContainer`,
`StartContainer`, `StopContainer`, `RemoveContainer`,
`ContainerStatus`, `ContainerLogs`), plus `ContainerExists`. Not-found
and already-exists conditions are reported as `ErrNotFound` /
`ErrAlreadyExists`, classified by matching known substrings in
`docker`'s own stderr (`"No such container"`, `"no such object"`,
`"already in use by container"`).

**Context:** These exact strings were captured from a live `docker`
(Engine 29.6.1) instance available in this sandbox — including a full
manual create → start → inspect → logs → stop → rm lifecycle and a
deliberate duplicate-name conflict — rather than assumed from
documentation, which can drift from actual CLI output across versions.
`docker rm -f` on an already-absent container was confirmed to exit 0,
so `RemoveContainer` does not need `ErrNotFound` handling — it is
naturally idempotent. This string-matching approach is inherently
version-sensitive (a future `docker` release could reword its errors);
the alternative — the Docker Engine SDK talking to the daemon socket
directly — would sidestep that fragility but was rejected to stay
consistent with ADR-0012's CLI-invocation design, shared with Shell and
k3d Runtimes.

---

## ADR-0016: Kubernetes Service reads Status/Logs from the cluster's server node container via Docker Runtime

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/service/kubernetes.Service` holds both a
`*k3d.Runtime` (for cluster lifecycle: Create/Start/Stop/Delete) and a
`*docker.Runtime` (for `Status`/`Logs`), reading the well-known
container name `k3d-<cluster>-server-0`. `Reset` has no k3d CLI
equivalent, so it composes `ClusterExists` → `DeleteCluster` →
`CreateCluster`.

**Context:** k3d has no `cluster status` or `cluster logs` command; a
k3d cluster's nodes are themselves Docker containers, and the
`k3d-<cluster>-server-0` naming convention was confirmed against a
real k3d (v5.9.0) cluster before relying on it. A Service depending on
two different Runtimes is consistent with the architecture — "Services
use a Runtime" constrains *how* they act (only through a Runtime,
never `os/exec` directly), not that a Service is limited to exactly
one Runtime instance. This was validated with a full real-lifecycle
smoke test (Create → Status → Logs → Kubeconfig → Stop → Start →
Reset → Delete) against live `k3d`/`docker`, not just the fake-backed
unit tests.

---

## ADR-0017: Extending a prior sprint's Runtime when its first consumer needs more

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** Sprint 7 added `ErrAlreadyExists` and `GetKubeconfig` to
`internal/runtime/k3d` (originally built in Sprint 5), rather than
duplicating that logic in the Kubernetes Service or leaving the gap
unaddressed.

**Context:** CLAUDE.md's "never jump to future sprints" governs adding
work that belongs to a *different, not-yet-approved* sprint (e.g.
building Docker Service now). It does not mean a completed sprint's
package is frozen forever — when Sprint 7 builds the first real
consumer of Sprint 5's k3d Runtime and discovers a genuinely missing
capability that Runtime should own, adding it there (not working around
its absence in the consumer) is the correct layering and was already
anticipated as normal by the Sprint 4→5 precedent (Shell Runtime
needed no such extension, but the pattern was established).

---

## ADR-0018: Docker Service reuses `docker.ContainerSpec` directly; `service.StatusCreated` is now load-bearing

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/service/docker.Service` takes a
`dockerruntime.ContainerSpec` (from Sprint 6) as its constructor
argument rather than redeclaring name/image/env/ports/volumes as its
own fields — a Docker Service *is* a single container, so its
construction parameters should be exactly a `ContainerSpec`, not a
parallel struct that would drift out of sync. Unlike Kubernetes
Service (one Runtime holding both lifecycle and introspection duties
split across two Runtime types), Docker Service needs only
`*docker.Runtime`, since every Service Rules verb already has a direct
1:1 Docker Runtime method.

`Status` maps `"created"` to `service.StatusCreated` — the first
Service to actually reach that branch. Confirmed live: `docker create`
without `docker start` leaves a container in a genuine, observable
`"created"` state, unlike a k3d cluster (Kubernetes Service, ADR-0016),
which is always `"running"` immediately after `k3d cluster create`.

**Context:** Verified end-to-end against a real `docker` (Engine
29.6.1) instance in this sandbox before finalizing: Create left the
container in `"created"`, Start moved it to `"running"`, Stop to
`"exited"` (mapped to `service.StatusStopped`), and Reset correctly
returned it to `"created"` (recreate does not imply start). This
grounds the status mapping in observed behavior, not assumption, same
practice as ADR-0015 and ADR-0016.
