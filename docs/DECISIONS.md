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
