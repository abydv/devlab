# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

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
