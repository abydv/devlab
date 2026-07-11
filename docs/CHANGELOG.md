# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Sprint 11 — REST API

- Add `api/`: a thin Fiber v2 HTTP surface over `internal/engine`,
  covering health, templates, and the full workspace lifecycle.
- Add `ListenAddr` to `internal/config` (`DEVLAB_LISTEN_ADDR`).
- Rewrite `cmd/devlab/main.go` to bootstrap every layer and serve the
  API by default; add `--version`.
- Add `github.com/gofiber/fiber/v2` dependency.
- Add unit tests using `fiber`'s `app.Test()` over a real Engine wired
  with fakes.

### Sprint 10 — Workspace Lifecycle

- Add the service type catalog (`internal/service/types.go`):
  `KnownTypes`/`IsKnownType`, covering all six Service Rules examples.
- Wire `internal/template.Registry.Load` to validate `services`
  entries against the catalog (`ErrUnknownService`).
- Add `internal/service/factory`: builds a `service.Service` from a
  type name, workspace ID, and data directory.
- Add `ContainerSpec.Privileged` and `docker.Runtime.RemoveVolume`
  (`internal/runtime/docker`), required for Docker-in-Docker.
- Shorten `utils.NewID` from 32 to 12 hex characters, to fit k3d's
  32-character cluster name limit.
- Add `workspace.Manager.SetStatus` and `DataDir`.
- Add `internal/engine/lifecycle.go`: `StartWorkspace`,
  `StopWorkspace`, `ResetWorkspace`, `WorkspaceStatus`,
  `WorkspaceLogs`; `DeleteWorkspace` is now context-aware and cleans up
  every attached Service before removing the Workspace record.
- Add unit tests for every new/changed piece.

### Sprint 9 — Jenkins Service

- Add the Jenkins Service (`internal/service/jenkins`), embedding
  `*servicedocker.Service` configured with Jenkins-specific defaults
  (`jenkins/jenkins:lts`, port 8080, `/var/jenkins_home` volume).
- Override `Create` to create the host data directory first; add
  `InitialAdminPassword`, read directly from the bind-mounted host
  path.
- Add unit tests for Jenkins-specific behavior plus a delegation check.

All three planned Services (Kubernetes, Docker, Jenkins) are complete.

### Sprint 8 — Docker Service

- Add the Docker Service (`internal/service/docker`): a single
  container backed directly by `docker.Runtime`, constructed from a
  reused `dockerruntime.ContainerSpec`.
- Map `docker inspect` state to `service.Status`, including the
  distinct `created` state (Docker containers, unlike k3d clusters,
  are not immediately running after creation).
- `Reset` composes `ContainerExists`/`RemoveContainer`/
  `CreateContainer`.
- Add unit tests composing a real `docker.Runtime` over a fake
  `runtime.Runtime`.

### Sprint 7 — Kubernetes Service

- Add the `Service` interface, `Status` type, and `ErrNotFound`
  (`internal/service`).
- Add the Kubernetes Service (`internal/service/kubernetes`), backed
  by a k3d cluster; `Status`/`Logs` read the server node container via
  Docker Runtime; `Reset` composes k3d Runtime calls.
- Extend `internal/runtime/k3d` with `ErrAlreadyExists` and
  `GetKubeconfig`.
- Add unit tests composing real `k3d.Runtime`/`docker.Runtime` over a
  shared fake `runtime.Runtime`.

### Sprint 6 — Docker Runtime

- Add the Docker Runtime (`internal/runtime/docker`), composed over an
  injected `runtime.Runtime`; `Execute` rejects non-`docker` commands.
- Add `CreateContainer`/`StartContainer`/`StopContainer`/
  `RemoveContainer`/`ContainerStatus`/`ContainerExists`/
  `ContainerLogs`, plus `ErrNotFound`/`ErrAlreadyExists`.
- Add unit tests using a fake `runtime.Runtime`, with error-text
  fixtures captured from a real `docker` instance.

### Sprint 5 — k3d Runtime

- Add the k3d Runtime (`internal/runtime/k3d`), composed over an
  injected `runtime.Runtime`; `Execute` rejects non-`k3d` commands.
- Add `CreateCluster`/`StartCluster`/`StopCluster`/`DeleteCluster`/
  `ListClusters`/`ClusterExists` convenience methods.
- Add unit tests using a fake `runtime.Runtime`, including a fixture
  captured from a real `k3d cluster list --output json` response.

### Sprint 4 — Shell Runtime

- Add the shared `Runtime` contract (`internal/runtime`):
  `Execute(ctx, Command) (*Result, error)`.
- Add the Shell Runtime (`internal/runtime/shell`), backed by
  `os/exec.CommandContext`; arguments are never shell-interpolated.
- Fix a context-cancellation bug found during testing:
  `exec.CommandContext` reports a killed process as a normal
  `*exec.ExitError`, which `Execute` now correctly distinguishes from
  a genuine nonzero exit by checking `ctx.Err()` first.
- Add unit tests covering stdout/stderr, exit codes, working
  directory, env vars, missing executables, and cancellation.

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
