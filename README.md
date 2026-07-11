# DevLab

A self-hosted Personal DevOps Workspace Platform for Ubuntu Desktop.

DevLab lets you spin up isolated, reproducible development workspaces —
each composed of one or more services (Kubernetes, Docker, Jenkins,
Linux, Terraform, Ansible, ...) — backed by pluggable runtimes (Shell,
Docker, k3d).

Built for personal use first; architected for future multi-user
expansion.

## Architecture

```
CLI → REST API → Engine → Workspace Manager → Services → Runtime → Docker / k3d / Shell
```

- **Everything is a Workspace.** A Workspace contains one or more Services.
- **Services use a Runtime.** Only the Runtime may execute operating
  system commands.
- **The Engine orchestrates Workspaces.**
- **The Frontend never contains business logic.**

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for full details.

## Tech Stack

| Layer    | Technology                                      |
|----------|--------------------------------------------------|
| Backend  | Go, Fiber, SQLite                                 |
| Frontend | Next.js, React, TypeScript, TailwindCSS, shadcn/ui |
| Runtime  | Docker, k3d                                       |

## Repository Layout

```
cmd/devlab/     CLI entrypoint
internal/       Engine, Workspace, Service, Runtime, Template, Storage, Config, Utils
pkg/            Reusable public packages
api/            REST API (Fiber)
web/            Next.js frontend
templates/      Workspace/service templates
workspaces/     Runtime workspace data
scripts/        Developer scripts
docs/           Project documentation (source of truth)
```

## Status

This project is under active development, following a fixed sprint
roadmap. See [`docs/ROADMAP.md`](docs/ROADMAP.md) and
[`docs/PROJECT_STATE.json`](docs/PROJECT_STATE.json) for current
progress.

**Sprint 0 (Architecture) is complete.** No application code has been
written yet — Sprint 1 (Workspace Engine) is pending approval.

## Development

Requires Go 1.25+.

```sh
make build   # go fmt + vet + test + build -> bin/devlab
make run     # run the CLI directly
make test    # go test ./...
make verify  # fmt, vet, test, build
```

## Documentation

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — sprint plan
- [`docs/PROGRESS.md`](docs/PROGRESS.md) — sprint completion log
- [`docs/TASKS.md`](docs/TASKS.md) — current sprint task checklist
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — architecture decision records
- [`docs/API.md`](docs/API.md) — REST API reference
- [`docs/CHANGELOG.md`](docs/CHANGELOG.md) — changelog
- [`docs/PROJECT_STATE.json`](docs/PROJECT_STATE.json) — machine-readable project state

## License

[MIT](LICENSE)
