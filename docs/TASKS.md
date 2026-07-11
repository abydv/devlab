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

- [x] Define the shared `Runtime` interface and `Command`/`Result`
      types (`internal/runtime`).
- [x] Implement the Shell Runtime (`internal/runtime/shell`) backed by
      `os/exec.CommandContext`, args passed as a slice (no shell
      interpolation).
- [x] Distinguish context cancellation from a normal nonzero exit code.
- [x] Unit tests: stdout/stderr, exit codes, working directory, env
      vars, missing executable, context cancellation.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 5 — k3d Runtime

- [x] Implement the k3d Runtime (`internal/runtime/k3d`), composed
      over an injected `runtime.Runtime`.
- [x] `Execute` rejects any command not targeting the `k3d` binary.
- [x] Add `CreateCluster`/`StartCluster`/`StopCluster`/`DeleteCluster`/
      `ListClusters`/`ClusterExists` convenience methods.
- [x] Unit tests using a fake `runtime.Runtime` test double.
- [x] Verify `ListClusters`' JSON parsing against real `k3d cluster
      list --output json` output (manual, one-off; not part of the
      automated suite).
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 6 — Docker Runtime

- [x] Implement the Docker Runtime (`internal/runtime/docker`),
      composed over an injected `runtime.Runtime`.
- [x] `Execute` rejects any command not targeting the `docker` binary.
- [x] Add `CreateContainer`/`StartContainer`/`StopContainer`/
      `RemoveContainer`/`ContainerStatus`/`ContainerExists`/
      `ContainerLogs`.
- [x] Add `ErrNotFound`/`ErrAlreadyExists`, classified from real
      `docker` CLI stderr text.
- [x] Manually verify command shapes and error text against a real
      `docker` instance (one-off; not part of the automated suite).
- [x] Unit tests using a fake `runtime.Runtime` test double.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 7 — Kubernetes Service

- [x] Define the `Service` interface, `Status` type, and `ErrNotFound`
      (`internal/service`).
- [x] Implement the Kubernetes Service (`internal/service/kubernetes`)
      backed by a k3d cluster.
- [x] Extend `internal/runtime/k3d` with `ErrAlreadyExists` and
      `GetKubeconfig`, needed by this sprint's consumer.
- [x] `Status`/`Logs` read the server node container via Docker
      Runtime; `Reset` composes k3d Runtime calls (no native reset).
- [x] Unit tests composing real `k3d.Runtime`/`docker.Runtime` over a
      shared fake `runtime.Runtime`.
- [x] Manually verify the full lifecycle end-to-end against real
      `k3d`/`docker` (one-off; not part of the automated suite);
      confirm no leftover resources afterward.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 8 — Docker Service

- [x] Implement the Docker Service (`internal/service/docker`), a
      single container backed by `docker.Runtime` and a reused
      `ContainerSpec`.
- [x] Map `docker inspect` state to `service.Status`, including the
      distinct `created` state.
- [x] `Reset` composes `ContainerExists`/`RemoveContainer`/
      `CreateContainer`.
- [x] Unit tests composing a real `docker.Runtime` over a fake
      `runtime.Runtime`.
- [x] Manually verify the full lifecycle end-to-end against real
      `docker` (one-off; not part of the automated suite); confirm no
      leftover containers.
- [x] Verify `go fmt`, `go vet`, `go test`, `go build` all pass.

## Sprint 9 — Jenkins Service

Not started. Awaiting approval to begin.
