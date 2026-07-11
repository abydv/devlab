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

## Sprint 5 — k3d Runtime

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/runtime/k3d`: the k3d Runtime, composed over an injected
  `runtime.Runtime` (typically Shell Runtime). Its `Execute` rejects
  any command whose `Name` isn't `"k3d"` — unlike Shell Runtime's
  deliberately unrestricted `Execute` (ADR-0013), k3d Runtime's whole
  purpose is a narrower, single-binary boundary.
- Convenience methods mirroring the Service Rules lifecycle where a
  real k3d CLI operation exists for it: `CreateCluster`, `StartCluster`,
  `StopCluster`, `DeleteCluster`, `ListClusters`, `ClusterExists`.
  `Reset` and `Logs` are intentionally not included — k3d has no
  native "reset cluster" operation (that will be a composition at the
  Service layer, Sprint 7) and no "cluster logs" command (container
  logs belong to Docker Runtime, Sprint 6).
- Unit tests use a fake `runtime.Runtime` test double — fast,
  deterministic, and independent of whether `k3d`/Docker are actually
  installed.
- `k3d` and `docker` happen to be available in this sandbox. Used them
  for one-off manual verification only (not part of the automated
  suite): created a real cluster, captured its actual
  `k3d cluster list --output json` response, confirmed `ListClusters`'
  minimal `{name}` unmarshal target matches the real (much larger)
  schema and ignores the rest correctly, then deleted the cluster. The
  captured response is embedded in `k3d_test.go` as a fixture so the
  automated tests stay grounded in real output without requiring the
  binaries to be present.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: Docker Runtime (Sprint 6); wiring k3d Runtime
into a Service (`internal/service` doesn't exist until Sprint 7); any
`kubectl`-level interaction with a cluster's contents (Kubernetes
Service's concern, Sprint 7) — k3d Runtime only manages cluster
lifecycle via the `k3d` binary itself.

Next: Sprint 6 — Docker Runtime (awaiting approval to start).

## Sprint 6 — Docker Runtime

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/runtime/docker`: the third `runtime.Runtime`
  implementation, mirroring k3d Runtime's design — composed over an
  injected `runtime.Runtime`, `Execute` rejects any command not
  targeting the `docker` binary.
- Container lifecycle methods mapping directly onto the Service Rules
  vocabulary, since — unlike k3d — every one of Create/Start/Stop/
  Delete/Status/Logs has a real `docker` CLI equivalent:
  `CreateContainer(spec)`, `StartContainer`, `StopContainer`,
  `RemoveContainer` (idempotent — real `docker rm -f` on an absent
  container exits 0), `ContainerStatus`, `ContainerExists`,
  `ContainerLogs`. `ContainerSpec` covers name, image, env, port
  mappings, volume mappings, and an optional command override — the
  minimum a Docker- or Jenkins-backed workspace service will need to
  publish a UI port and mount the workspace's `data/` directory.
- `ErrNotFound` / `ErrAlreadyExists` sentinels, classified from
  `docker`'s own stderr text (`"No such container"` / `"no such
  object"` / `"already in use by container"`) — verified against a
  real `docker` (Engine 29.6.1) instance available in this sandbox,
  not guessed; the exact strings are embedded as test fixtures.
- Before writing the Go code, ran a full manual lifecycle against real
  `docker` (create → start → inspect → logs → stop → rm -f, plus a
  duplicate-name conflict) to confirm the command shapes and error text
  this design depends on, then cleaned up every container it created.
- Unit tests use a fake `runtime.Runtime` double — fast and independent
  of Docker actually being installed.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: wiring Docker Runtime into a Service
(`internal/service` doesn't exist until Sprint 7); a true multiplexed
stdout/stderr log stream (`ContainerLogs` concatenates the two captured
buffers stdout-then-stderr, not chronologically interleaved — a known
simplification, not a live tail).

All three Runtimes (Shell, k3d, Docker) are now implemented. Next:
Sprint 7 — Kubernetes Service (awaiting approval to start).

## Sprint 7 — Kubernetes Service

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/service`: the `Service` interface (Create, Start, Stop,
  Reset, Delete, Status, Logs — exactly the Service Rules verbs), a
  `Status` type (created/running/stopped/error), and an `ErrNotFound`
  sentinel. Justified as an interface now because three concrete
  Services are explicitly planned (Kubernetes, Docker, Jenkins).
- `internal/service/kubernetes`: the first `service.Service`
  implementation, backed by a k3d cluster. Create/Start/Stop/Delete map
  directly onto k3d Runtime's cluster methods; `Reset` composes
  `ClusterExists` + `DeleteCluster` + `CreateCluster` since k3d has no
  native reset; `Status`/`Logs` read the cluster's server node
  container (`k3d-<cluster>-server-0`) via the Docker Runtime, since a
  k3d cluster's nodes are themselves Docker containers. A `Kubeconfig`
  method (beyond the `Service` interface) exposes the cluster's
  kubeconfig for future consumers (e.g. a browser terminal, Sprint 13).
- Extended `internal/runtime/k3d` (built in Sprint 5) with what this,
  its first real consumer, needs: `ErrAlreadyExists` (classified from
  real k3d stderr text, captured live: `"...because a cluster with
  that name already exists"`) and `GetKubeconfig` (`k3d kubeconfig
  get`). This is the same incremental pattern as Sprint 7 discovering
  Sprint 5's package needed one more method — not scope creep on this
  sprint, and not a redesign of Sprint 5's decisions.
- Verified the full real lifecycle end-to-end before finalizing: wrote
  a throwaway program (not committed) wiring Shell Runtime → k3d
  Runtime + Docker Runtime → Kubernetes Service, and ran Create → Status
  → Logs → Kubeconfig → Stop → Status → Start → Status → Reset → Status
  → Delete → Status (confirmed `ErrNotFound`) against the real
  `k3d`/`docker` in this sandbox. Cleaned up all resources afterward;
  confirmed no leftover clusters or containers.
- Unit tests compose real `k3d.Runtime`/`docker.Runtime` instances over
  a single shared fake `runtime.Runtime` (routing on the underlying
  binary), so they exercise the actual composition DevLab will run in
  production — not a mock of the composition — while staying fast and
  independent of `k3d`/Docker being installed.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: wiring Kubernetes Service into the Engine or
Workspace (that dependency direction — Engine knowing about Services —
belongs to Sprint 10, Workspace Lifecycle); Docker Service and Jenkins
Service (Sprints 8-9); validating a Template's `services: ["kubernetes"]`
entry against this implementation (ADR-0009 still applies — no
service-type catalog exists yet, since one Service alone isn't a
catalog).

Next: Sprint 8 — Docker Service (awaiting approval to start).

## Sprint 8 — Docker Service

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/service/docker`: the second `service.Service`
  implementation, a single container run from a caller-supplied
  `dockerruntime.ContainerSpec` (reused directly from Sprint 6 rather
  than duplicating its fields). Simpler than Kubernetes Service since
  Docker Runtime's methods already map 1:1 onto the Service Rules verbs
  — no second Runtime needed. `Reset` composes
  `ContainerExists`/`RemoveContainer`/`CreateContainer` since
  `docker` has no single "recreate" command.
- `Status` maps `docker inspect`'s state to `service.Status`: unlike a
  k3d cluster (always "running" immediately after create), a Docker
  container genuinely has a distinct `"created"` state before Start —
  confirmed live, not assumed — so `service.StatusCreated` is used
  here for the first time. `"paused"`/`"restarting"`/etc. map to
  `service.StatusError` as a deliberate catch-all, same reasoning as
  Kubernetes Service's default case.
- Verified the full lifecycle end-to-end with a throwaway program (not
  committed): Create → Status (created) → Start → Status (running) →
  Logs → Stop → Status (stopped) → Reset → Status (created) → Delete →
  Status (confirmed `ErrNotFound`), against the real `docker` in this
  sandbox. Cleaned up the container afterward.
- Unit tests compose a real `docker.Runtime` over a fake
  `runtime.Runtime`, including a table test over every observed Docker
  state.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: wiring Docker Service into the Engine/Workspace
(Sprint 10); Jenkins Service (Sprint 9).

Next: Sprint 9 — Jenkins Service (awaiting approval to start).

## Sprint 9 — Jenkins Service

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `internal/service/jenkins`: the third `service.Service`
  implementation, built by composition rather than reimplementation —
  it embeds `*servicedocker.Service` (Sprint 8), configured with
  Jenkins-specific defaults (`jenkins/jenkins:lts`, port 8080 mapped to
  a caller-chosen host port, `/var/jenkins_home` bind-mounted to a
  caller-chosen host directory). `Start`/`Stop`/`Reset`/`Delete`/
  `Status`/`Logs` come for free via Go's method promotion; only
  `Create` is overridden (to `os.MkdirAll` the data directory first)
  and one bonus method, `InitialAdminPassword`, is added.
- Before finalizing the design, tested live whether the official
  `jenkins/jenkins:lts` image (which runs as a non-root UID) would hit
  a permission error against a plain host-created bind-mount directory
  — a well-known Docker gotcha. It did not: Jenkins started cleanly and
  the well-known `secrets/initialAdminPassword` file was both generated
  correctly and readable directly from the host path. Because of that,
  `InitialAdminPassword` is implemented as a plain host file read
  (`os.ReadFile` on the bind-mounted path), not a `docker exec` into
  the container — no new Runtime capability was needed, and reading a
  file DevLab itself mounted is not "executing an operating system
  command" any more than `workspace.Manager` reading `workspace.json`
  directly is.
- Verified the full lifecycle end-to-end with a throwaway program (not
  committed): Create → Start → Status (running) → poll for
  `InitialAdminPassword` → Logs → Stop → Status (stopped) → Delete →
  Status (confirmed `ErrNotFound`), against the real `docker` in this
  sandbox, including a real Jenkins boot. Cleaned up the container and
  data directory afterward.
- Unit tests cover the Jenkins-specific behavior (container spec
  construction, data directory creation, missing-data-dir validation,
  password file reading) plus one delegation check confirming struct
  embedding correctly wires through to Docker Service's methods — full
  Docker Service behavior is already covered by Sprint 8's own tests
  and isn't re-tested here.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

All three planned Services (Kubernetes, Docker, Jenkins) are now
implemented. Explicitly out of scope: wiring any Service into the
Engine/Workspace, and validating a Template's `services` entries
against a service-type catalog — both belong to Sprint 10, Workspace
Lifecycle, which is next.

Next: Sprint 10 — Workspace Lifecycle (awaiting approval to start).

## Sprint 10 — Workspace Lifecycle

**Status:** Complete
**Date:** 2026-07-12

The largest sprint so far: wires Workspace, Template, Engine, Runtime,
and Service together into a working, verified end-to-end system.

Before implementation, asked the user how the generic `"docker"`
Service should behave by default, since Templates (Sprint 2) carry no
per-service config and nothing upstream specified an image. Answer:
Docker-in-Docker (`docker:dind`) — the standard devcontainer/CI
pattern for a generic Docker workspace.

Delivered:

- **Service type catalog** (`internal/service/types.go`): `KnownTypes`
  / `IsKnownType`, covering all six Service Rules examples
  (kubernetes, docker, jenkins, linux, terraform, ansible) — not just
  the three implemented ones, so `templates/linux.json` etc. (seeded in
  Sprint 2) keep loading. `internal/template.Registry.Load` now
  validates each Template's `services` entries against this catalog
  (`ErrUnknownService`), finally fulfilling ADR-0009.
- **`internal/service/factory`**: builds the concrete `service.Service`
  for a service type, workspace ID, and data directory. Kubernetes →
  `kubernetes.New`; Jenkins → `jenkins.New` with a dynamically
  allocated free host port (`utils.FreePort`, new) and
  `<dataDir>/jenkins`; Docker → Docker-in-Docker (privileged,
  named-volume storage — see below); Linux/Terraform/Ansible → a clear
  "recognized but not implemented yet" error, not silently ignored.
- **`internal/runtime/docker`**: added `ContainerSpec.Privileged` and
  `RemoveVolume`, both required by Docker-in-Docker and both verified
  against a real `docker` (Engine 29.6.1) instance before implementing
  — including the discovery, live, that a host bind-mount for dind's
  `/var/lib/docker` leaves root-owned files an unprivileged process
  cannot later delete, while `docker volume rm -f` on a Docker named
  volume has no such problem (and is idempotent). The Docker-type
  Service the factory builds is wrapped (`dindService`, unexported, in
  `factory.go`) to remove that named volume as part of `Delete`.
- **Workspace ID shortened** from 32 to 12 hex characters
  (`internal/utils.NewID`, 6 random bytes): discovered live that k3d
  rejects cluster names over 32 characters, and `devlab-<32hex>-kubernetes`
  is 50. 48 bits of randomness remains far more than sufficient for a
  personal, single-user tool.
- **`internal/workspace.Manager`**: added `SetStatus` (persists a
  Status change to both `workspace.json` and the SQLite index) and
  `DataDir` (exposes a Workspace's data directory path without
  leaking the package's internal layout).
- **`internal/engine`**: split into `engine.go` (CRUD, unchanged in
  shape) and `lifecycle.go` (new). `CreateWorkspace` still only
  creates the Workspace record — Service resources are provisioned
  lazily. `StartWorkspace` builds each attached Service, creates it if
  `Status` reports `ErrNotFound`, then starts it (idempotent by
  construction: every underlying `Start`/cluster-create call was
  confirmed live to be a no-op success when already running/existing).
  `StopWorkspace`/`ResetWorkspace`/`DeleteWorkspace` (now
  context-aware) loop the same way. `WorkspaceStatus` aggregates each
  Service's status into one Workspace-level `Status` (running only if
  at least one Service is running and none disagree; error on a
  disagreement or an underlying Service error) and persists it.
  `WorkspaceLogs` concatenates each Service's logs, labeled by type.
  `DeleteWorkspace` deletes every Service's resource before removing
  the Workspace record.
- Verified the complete real lifecycle end-to-end with a throwaway
  program (not committed), for both the `kubernetes` and `docker`
  templates: Create → Start → Status → Logs → Stop → Status → Start
  (restart) → Status → Reset → Status → Delete → Status (confirmed
  `ErrNotFound`) — against the real `k3d`/`docker` in this sandbox.
  This run is what caught both the k3d name-length limit and the dind
  bind-mount permission problem; both are fixed and re-verified clean
  (no leftover clusters, containers, or volumes).
- Unit tests throughout use fakes; the new/changed packages
  (`service/types`, `service/factory`, `engine/lifecycle`,
  `workspace.SetStatus`/`DataDir`, `docker.RemoveVolume`/`Privileged`)
  are all covered.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: CLI/REST API wiring to actually call the
Engine (Sprint 11); Linux/Terraform/Ansible Service implementations
(no dedicated roadmap sprint exists for them — see ARCHITECTURE.md).

Next: Sprint 11 — REST API (awaiting approval to start).

## Sprint 11 — REST API

**Status:** Complete
**Date:** 2026-07-12

Delivered:

- `api/`: a thin Fiber v2 HTTP surface over `internal/engine`. Holds no
  business logic — every handler validates its input, delegates to the
  Engine, and translates the result into an HTTP response. Routes:
  `GET /healthz`; `GET /api/templates`, `GET /api/templates/:name`;
  `POST /api/workspaces`, `GET /api/workspaces`, `GET
  /api/workspaces/:id`, `DELETE /api/workspaces/:id`, `POST
  /api/workspaces/:id/{start,stop,reset}`, `GET
  /api/workspaces/:id/status`, `GET /api/workspaces/:id/logs` (plain
  text). `writeError` maps domain errors (`workspace.ErrNotFound`,
  `template.ErrNameExists`, ...) to HTTP status codes — transport
  translation, not business logic.
- `internal/config`: added `ListenAddr` (`DEVLAB_LISTEN_ADDR`, default
  `:8080`), resolved the same way as the filesystem paths — no
  hardcoded port either.
- `cmd/devlab/main.go`: rewritten from a version-only stub into a real
  entrypoint. With no flags, it bootstraps config → storage →
  workspace manager → template registry → runtimes → service factory →
  engine → API server, and serves it. `--version` still prints the
  version and exits. There is no dedicated CLI sprint on the roadmap,
  so `devlab` itself functions as the REST API server process — the
  most direct reading of CLAUDE.md's "CLI → REST API → Engine" flow
  for a self-hosted single binary.
- Added `github.com/gofiber/fiber/v2` (v2.52.14) as a dependency.
- Verified the actual compiled binary, not just unit tests: ran
  `devlab --version`, then started the real server
  (`DEVLAB_LISTEN_ADDR`, an isolated `DEVLAB_HOME` with the repo's
  `templates/` copied in) and drove the complete workspace lifecycle
  with real `curl` requests against real `k3d`/`docker`: create → start
  (a real k3d cluster came up) → status → logs → stop → delete → a
  subsequent GET correctly 404s. Cross-checked the API's view against
  `k3d cluster list` directly at the running-cluster point — exact
  match on cluster name and state. Shut the server down and confirmed
  zero leftover clusters or containers.
- Unit tests use `fiber`'s `app.Test()` against a real `Engine` wired
  over the same fake-`runtime.Runtime` pattern used throughout the
  project — fast, deterministic, and exercising the real routing/
  handler code, not a mock of it.
- Build validation passed: `go fmt`, `go vet`, `go test`, `go build`.

Explicitly out of scope: the `web/` Next.js frontend (Sprint 12);
authentication (Sprint 15) — the API currently has none, appropriate
for a personal, single-user, localhost-bound tool at this stage but
worth flagging.

Next: Sprint 12 — Dashboard (awaiting approval to start).
