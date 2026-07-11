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

---

## ADR-0019: Jenkins Service is a configured Docker Service via struct embedding, not a fourth from-scratch implementation

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/service/jenkins.Service` embeds
`*servicedocker.Service` (Sprint 8) rather than reimplementing
container lifecycle logic or composing over `*docker.Runtime`
directly like Docker Service does. `New` builds a
`docker.ContainerSpec` with Jenkins-specific defaults (image, port
8080, `/var/jenkins_home` volume) and passes it straight to
`servicedocker.New`. `Start`/`Stop`/`Reset`/`Delete`/`Status`/`Logs`
are inherited via Go method promotion, unmodified. Only `Create` is
overridden — to `os.MkdirAll` the host data directory before
delegating to the embedded Service's `Create` — and one addition,
`InitialAdminPassword`, goes beyond the `Service` interface.

**Context:** A Jenkins Service *is*, mechanically, a Docker container
with Jenkins-specific configuration — there is no Jenkins-specific
lifecycle behavior to justify a parallel implementation, so embedding
avoids duplicating logic Sprint 8 already built and tested (the "no
duplicate code" standard). `InitialAdminPassword` is implemented as a
plain `os.ReadFile` on the bind-mounted host path, not a `docker exec`
into the container: this was verified live before finalizing — the
official `jenkins/jenkins:lts` image (which runs as a non-root UID,
a well-known Docker bind-mount permission trap) started cleanly
against a plain host-created directory in this sandbox, and its
`secrets/initialAdminPassword` file was both generated and
host-readable. Because DevLab itself controls that mount, reading it
directly is not "executing an operating system command" any more than
`workspace.Manager` reading `workspace.json` is — no new Runtime
capability (e.g. a generic `docker exec` method) was needed for this.

---

## ADR-0020: Service type catalog lives in `internal/service`; validated by Template, populated by Factory

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/service.KnownTypes`/`IsKnownType` list all six
Service Rules examples (kubernetes, docker, jenkins, linux, terraform,
ansible) — not just the three with a concrete implementation.
`internal/template.Registry.Load` validates a Template's `services`
entries against this catalog (`ErrUnknownService`).
`internal/service/factory.Factory.Build` separately returns a clear
"recognized service type but has no implementation yet" error for
linux/terraform/ansible, rather than pretending they work or failing
Template validation for them.

**Context:** Fulfills the plan recorded in ADR-0009 now that all
three implemented Services exist. Validating Templates against only
the *implemented* subset would have broken `templates/linux.json`,
`terraform.json`, and `ansible.json` (seeded in Sprint 2, all
legitimate per CLAUDE.md's Service Rules) — the catalog's job is
"is this a recognized service category," not "can we build it today."
Putting the catalog in `internal/service` (not `internal/service/factory`)
avoids a real import cycle: `internal/service/factory` must import the
concrete `kubernetes`/`docker`/`jenkins` packages, which themselves
import `internal/service` for the interface — if the catalog lived in
`factory`, `internal/template` would need to import it and pull in
every Runtime dependency transitively, just to check a string.
`internal/service` itself stays dependency-free (stdlib only), so
`internal/template` depending on it for validation costs nothing.

---

## ADR-0021: Generic "docker" Service defaults to Docker-in-Docker

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/service/factory.Build` constructs the
`"docker"` service type as a `docker:dind` container (privileged, with
its own isolated Docker daemon), not a caller-specified arbitrary
image.

**Context:** Templates (Sprint 2) carry only a bare service type name
— no image, ports, or volumes — so wiring `CreateWorkspace` → Service
provisioning required *some* default for what a generic "Docker
workspace" runs, and nothing upstream specified one. Asked the user
directly rather than guessing a product decision silently. Confirmed
choice: Docker-in-Docker, the standard devcontainer/CI pattern for a
workspace that needs to build/run its own containers — not a
user-facing application, which "run an arbitrary named image" would
have implied without a concrete use case driving it.

---

## ADR-0022: `ContainerSpec.Privileged`, verified required for Docker-in-Docker

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/runtime/docker.ContainerSpec` gained a
`Privileged bool` field, wired to `docker create --privileged` when
set.

**Context:** Verified live before implementing: `docker:dind` without
`--privileged` fails at daemon startup (`mount: permission denied
(are you root?)`, exit 1); with `--privileged` it starts and its inner
daemon is fully functional (confirmed via `docker exec ... docker
version`). This is the same "extend a prior sprint's package when its
new consumer needs it" pattern as ADR-0017.

---

## ADR-0023: Workspace IDs shortened to fit k3d's 32-character cluster name limit

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `internal/utils.NewID` generates 6 random bytes (12 hex
characters), not 16 bytes (32 hex characters).

**Context:** Discovered live, not anticipated: k3d hard-rejects
cluster names over 32 characters ("Cluster name must be <= 32
characters"). The naming scheme `devlab-<id>-kubernetes` (established
this sprint for Service resource names) is 18 characters of fixed
overhead; a 32-hex-character ID made that 50 characters, breaking
every Kubernetes-templated workspace outright — this was caught by
the mandatory real end-to-end verification, not by unit tests (the
unit tests, using fakes, could not have caught it). 12 hex characters
(48 bits of randomness) keeps the total at 30 characters with margin,
and remains far more collision-resistant than a personal, single-user
tool's realistic workspace count will ever need.

---

## ADR-0024: Docker-in-Docker storage uses a named Docker volume, not a host bind-mount

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** The factory-built `"docker"` Service mounts
`/var/lib/docker` from a Docker *named volume*
(`devlab-<id>-docker-data`, passed via the same `-v` flag syntax
Docker itself uses to distinguish volume names from host paths — no
Runtime-level special casing needed), not a host directory under the
Workspace's `data/`. Deleting it (`dindService.Delete`, an unexported
wrapper in `factory.go`) calls the new `docker.Runtime.RemoveVolume`
(`docker volume rm -f`, confirmed idempotent on an absent volume)
after removing the container.

**Context:** Discovered live, via the mandatory end-to-end
verification: a host bind-mount for dind's storage works fine while
the container is running, but the inner Docker daemon leaves
root-owned files in it that DevLab's own unprivileged process cannot
later remove — `workspace.Manager.Delete`'s final `os.RemoveAll` on
the Workspace directory failed with `permission denied` on the first
attempt. A Docker named volume is removed by the Docker daemon itself
(root-equivalent for this purpose), so `docker volume rm -f` succeeds
unconditionally regardless of what the dind daemon wrote inside it.
Re-verified end-to-end after the fix: full lifecycle including Delete
now succeeds with zero leftover clusters, containers, or volumes.

---

## ADR-0025: Engine lifecycle — lazy provisioning, idempotent Start, aggregated Status

**Date:** 2026-07-12
**Status:** Accepted

**Decision:**
- `CreateWorkspace` only creates the Workspace record — it does not
  provision any Service's underlying resource.
- `StartWorkspace` builds each attached Service (via
  `factory.Factory`), calls `Create` only if `Status` reports
  `service.ErrNotFound`, then always calls `Start`. `StopWorkspace` and
  `ResetWorkspace` follow the same "build then call the matching
  Service method" shape.
- `DeleteWorkspace` deletes every attached Service's resource *before*
  removing the Workspace record (directory + index row).
- `WorkspaceStatus` recomputes and persists a Workspace-level `Status`
  from its Services: `running` only if at least one Service reports
  running and none report a non-running state; `stopped` if none are
  running; `error` if Services disagree (some running, some not) or
  any Service itself reports `service.StatusError` or a genuine error.

**Context:** Lazy provisioning avoids creating Docker containers/k3d
clusters for workspaces a user creates but never starts — appropriate
for a personal desktop tool where `CreateWorkspace` should be cheap.
The unconditional `Start` call after a conditional `Create` is safe
specifically because every underlying primitive was verified live to
be idempotent when already in the target state: `k3d cluster start` on
an already-running cluster, `docker start` on an already-running
container, and `k3d cluster create`/`docker rm -f`/`docker volume rm -f`
on resources that already exist or are already absent all exit 0
rather than erroring — this was checked deliberately before writing
the orchestration logic, not assumed. Deleting Services before the
Workspace record (rather than the reverse) avoids ever leaving an
orphaned cluster/container behind if the Workspace record is removed
but Service cleanup wasn't attempted. Status aggregation intentionally
has no "partial" bucket beyond mapping disagreement to `error`, since
`workspace.Status` (Sprint 1) has no such state and CLAUDE.md doesn't
call for adding one.

---

## ADR-0026: `devlab` itself is the REST API server process

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `cmd/devlab/main.go`, run with no flags, bootstraps every
layer (config → storage → workspace manager → template registry →
runtimes → service factory → engine) and serves the REST API on
`config.ListenAddr`. `--version` is the only other supported flag.

**Context:** CLAUDE.md's architecture states "CLI → REST API → Engine"
but the roadmap has no dedicated CLI sprint (noted as far back as
Sprint 1's progress notes) — there is no separate interactive command
set planned to be the "CLI" half of that arrow. For a self-hosted
single-binary tool, the most direct, non-speculative reading is that
the binary itself *is* the REST API server process; a future CLI
client (or the Sprint 12 Dashboard) talks to it over HTTP. Building an
interactive subcommand/HTTP-client CLI now would be scope invention
with no CLAUDE.md instruction backing it.

---

## ADR-0027: API error mapping is transport translation, not business logic

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** `api/errors.go`'s `writeError` maps known domain
sentinel errors (`workspace.ErrNotFound`, `template.ErrNameExists`,
`service.ErrNotFound`, ...) to HTTP status codes (404, 409, 400) via
`errors.Is`, falling back to 500 for anything unrecognized. Handlers
never branch on business conditions themselves — they call exactly one
Engine method and either return its result or hand its error to
`writeError`.

**Context:** CLAUDE.md requires the REST API layer to "contain no
business logic — it validates and delegates." Deciding which HTTP
status code represents an error the Engine already produced is
delegation's necessary counterpart (translating a decision already
made into the transport's vocabulary), not a new decision — the API
layer never decides *whether* an operation should succeed, only how to
report that Engine already decided it didn't.

---

## ADR-0028: Verify the compiled binary live, not just `app.Test()`

**Date:** 2026-07-12
**Status:** Accepted

**Decision:** Before considering Sprint 11 complete, built the real
`devlab` binary, ran it against a real, isolated `DEVLAB_HOME` (with
`templates/` copied in, since `TemplatesDir` resolves relative to
`HomeDir`), and drove the full workspace lifecycle with real `curl`
requests against real `k3d`/`docker` — including cross-checking the
API's reported cluster state directly against `k3d cluster list`.

**Context:** `api`'s own tests use Fiber's `app.Test()`, which never
binds a real port or spawns the real process — it proves the routing
and handler logic, not that `go run`/the built binary actually starts,
binds a configurable address, and serves real traffic end-to-end. This
sprint is the first time `cmd/devlab` does real work (prior sprints
only printed a version), so the mandatory-live-verification practice
established since Sprint 5 applies here at the process level, not just
the package level. Caught nothing new this time, but the discovery
that `TemplatesDir` doesn't automatically see the repo's own
`templates/` unless `DEVLAB_HOME` points there (or they're copied in)
is worth remembering for deployment/packaging later.
