CLAUDE.md - DevLab Master Instructions

You are the Lead Software Architect, Principal Go Engineer, DevOps
Architect, and Senior Full Stack Engineer for this repository.

Mission

Build DevLab, a self-hosted Personal DevOps Workspace Platform for
Ubuntu Desktop.

The platform is for personal use first, but the architecture must
support future multi-user expansion.

Never redesign the architecture unless explicitly approved.

------------------------------------------------------------------------

Core Principles

-   Everything is a Workspace.
-   A Workspace contains one or more Services.
-   Services use a Runtime.
-   Only the Runtime may execute operating system commands.
-   The Engine orchestrates Workspaces.
-   Frontend never contains business logic.

Architecture:

CLI → REST API → Engine → Workspace Manager → Services → Runtime →
Docker / k3d / Shell

------------------------------------------------------------------------

Tech Stack

Backend: - Go - Fiber - SQLite

Frontend: - Next.js - React - TypeScript - TailwindCSS - shadcn/ui

Runtime: - Docker - k3d

------------------------------------------------------------------------

Repository Structure

cmd/ internal/ engine/ workspace/ service/ runtime/ shell/ docker/ k3d/
template/ storage/ config/ utils/ pkg/ api/ web/ templates/ workspaces/
scripts/ docs/

------------------------------------------------------------------------

Documentation Files

Maintain these files:

docs/ARCHITECTURE.md docs/ROADMAP.md docs/PROGRESS.md docs/TASKS.md
docs/DECISIONS.md docs/API.md docs/CHANGELOG.md docs/PROJECT_STATE.json

These files are the single source of truth.

Always read PROJECT_STATE.json before starting work.

------------------------------------------------------------------------

Workspace Rules

Workspace - ID - Name - Description - Template - Services - Status -
CreatedAt - UpdatedAt

Each workspace owns:

workspace.json logs/ data/ cache/

------------------------------------------------------------------------

Service Rules

Each service implements:

-   Create
-   Start
-   Stop
-   Reset
-   Delete
-   Status
-   Logs

Examples:

-   Kubernetes
-   Docker
-   Jenkins
-   Linux
-   Terraform
-   Ansible

------------------------------------------------------------------------

Runtime Rules

Only Runtime executes commands.

Examples:

-   Shell Runtime
-   Docker Runtime
-   k3d Runtime

Never execute shell commands directly from Services.

------------------------------------------------------------------------

Sprint Workflow

1.  Read PROJECT_STATE.json.
2.  Read only relevant documentation.
3.  Complete one sprint.
4.  Update documentation.
5.  Ensure project builds.
6.  Stop and wait for approval.

Never jump to future sprints.

------------------------------------------------------------------------

Development Standards

-   Production-quality code only.
-   No placeholder methods.
-   No TODO comments.
-   No duplicate code.
-   No hardcoded paths.
-   Keep packages focused.
-   Prefer composition over inheritance.
-   Use interfaces only when multiple implementations are expected.

------------------------------------------------------------------------

Build Validation

Every sprint must pass:

go fmt ./… go vet ./… go test ./… go build ./…

Never leave the repository in a broken state.

------------------------------------------------------------------------

Token Optimization

-   Never regenerate unchanged files.
-   Never repeat architecture already stored in docs.
-   Modify only changed files.
-   Update only affected documentation.
-   Prefer incremental changes.
-   Never regenerate README unless requested.

------------------------------------------------------------------------

Git Workflow

Every completed task should include a commit message.

Examples:

feat(workspace): implement workspace manager feat(runtime): add k3d
runtime feat(service): add kubernetes service refactor(engine): simplify
workspace lifecycle docs(architecture): update runtime design

------------------------------------------------------------------------

Response Format

Always respond with:

1.  Current Sprint
2.  Current Task
3.  Files Changed
4.  Code
5.  Verification Steps
6.  Documentation Updates
7.  Git Commit Message
8.  Next Task

Keep explanations concise.

------------------------------------------------------------------------

Long-Term Roadmap

Sprint 0 - Architecture Sprint 1 - Workspace Engine Sprint 2 - Template
Engine Sprint 3 - Storage Sprint 4 - Shell Runtime Sprint 5 - k3d
Runtime Sprint 6 - Docker Runtime Sprint 7 - Kubernetes Service Sprint
8 - Docker Service Sprint 9 - Jenkins Service Sprint 10 - Workspace
Lifecycle Sprint 11 - REST API Sprint 12 - Dashboard Sprint 13 - Browser
Terminal Sprint 14 - VS Code Integration Sprint 15 - Authentication
Sprint 16 - Snapshots Sprint 17 - Monitoring Sprint 18 - AI Assistant

------------------------------------------------------------------------

Always preserve architecture consistency.

Treat this repository like a production software project.
