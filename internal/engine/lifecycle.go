package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/abydv/devlab/internal/service"
	"github.com/abydv/devlab/internal/workspace"
)

// workspaceService pairs a built Service with the type name it was
// built from, so orchestration errors and logs can identify which
// Service they concern.
type workspaceService struct {
	Type    string
	Service service.Service
}

func (e *Engine) buildServices(ws *workspace.Workspace) ([]workspaceService, error) {
	dataDir := e.workspaces.DataDir(ws.ID)

	services := make([]workspaceService, 0, len(ws.Services))
	for _, svcType := range ws.Services {
		svc, err := e.services.Build(svcType, ws.ID, dataDir)
		if err != nil {
			return nil, fmt.Errorf("engine: build %s service: %w", svcType, err)
		}
		services = append(services, workspaceService{Type: svcType, Service: svc})
	}
	return services, nil
}

// StartWorkspace starts every Service attached to the Workspace,
// provisioning (Create) any that have not been created yet.
func (e *Engine) StartWorkspace(ctx context.Context, id string) error {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return err
	}

	services, err := e.buildServices(ws)
	if err != nil {
		return err
	}

	for _, s := range services {
		if _, statusErr := s.Service.Status(ctx); errors.Is(statusErr, service.ErrNotFound) {
			if err := s.Service.Create(ctx); err != nil {
				return fmt.Errorf("engine: create %s service: %w", s.Type, err)
			}
		} else if statusErr != nil {
			return fmt.Errorf("engine: check %s service status: %w", s.Type, statusErr)
		}

		if err := s.Service.Start(ctx); err != nil {
			return fmt.Errorf("engine: start %s service: %w", s.Type, err)
		}
	}

	_, err = e.syncStatus(ctx, ws)
	return err
}

// StopWorkspace stops every Service attached to the Workspace without
// deleting their resources.
func (e *Engine) StopWorkspace(ctx context.Context, id string) error {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return err
	}

	services, err := e.buildServices(ws)
	if err != nil {
		return err
	}

	for _, s := range services {
		if err := s.Service.Stop(ctx); err != nil {
			return fmt.Errorf("engine: stop %s service: %w", s.Type, err)
		}
	}

	_, err = e.syncStatus(ctx, ws)
	return err
}

// ResetWorkspace discards and recreates every Service attached to the
// Workspace.
func (e *Engine) ResetWorkspace(ctx context.Context, id string) error {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return err
	}

	services, err := e.buildServices(ws)
	if err != nil {
		return err
	}

	for _, s := range services {
		if err := s.Service.Reset(ctx); err != nil {
			return fmt.Errorf("engine: reset %s service: %w", s.Type, err)
		}
	}

	_, err = e.syncStatus(ctx, ws)
	return err
}

// DeleteWorkspace permanently removes a Workspace, deleting every
// attached Service's underlying resource first.
func (e *Engine) DeleteWorkspace(ctx context.Context, id string) error {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return err
	}

	services, err := e.buildServices(ws)
	if err != nil {
		return err
	}

	for _, s := range services {
		if err := s.Service.Delete(ctx); err != nil {
			return fmt.Errorf("engine: delete %s service: %w", s.Type, err)
		}
	}

	return e.workspaces.Delete(id)
}

// WorkspaceStatus recomputes a Workspace's Status from its Services,
// persists it, and returns it.
func (e *Engine) WorkspaceStatus(ctx context.Context, id string) (workspace.Status, error) {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return "", err
	}

	ws, err = e.syncStatus(ctx, ws)
	if err != nil {
		return "", err
	}
	return ws.Status, nil
}

// WorkspaceLogs returns the combined logs of every Service attached to
// the Workspace, each labeled with its service type.
func (e *Engine) WorkspaceLogs(ctx context.Context, id string) (string, error) {
	ws, err := e.workspaces.Get(id)
	if err != nil {
		return "", err
	}

	services, err := e.buildServices(ws)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, s := range services {
		logs, err := s.Service.Logs(ctx)
		if errors.Is(err, service.ErrNotFound) {
			logs = "(not created)\n"
		} else if err != nil {
			return "", fmt.Errorf("engine: get %s service logs: %w", s.Type, err)
		}

		fmt.Fprintf(&sb, "=== %s ===\n%s\n", s.Type, logs)
	}
	return sb.String(), nil
}

// syncStatus recomputes ws's Status from its Services and persists it.
func (e *Engine) syncStatus(ctx context.Context, ws *workspace.Workspace) (*workspace.Workspace, error) {
	services, err := e.buildServices(ws)
	if err != nil {
		return nil, err
	}

	status, err := aggregateStatus(ctx, services)
	if err != nil {
		return nil, err
	}

	return e.workspaces.SetStatus(ws.ID, status)
}

// aggregateStatus derives a Workspace-level Status from its Services'
// individual statuses: running only if at least one Service is
// running and none are error/not-running, stopped if none are
// running, error if Services disagree or any Service itself reports
// an error.
func aggregateStatus(ctx context.Context, services []workspaceService) (workspace.Status, error) {
	if len(services) == 0 {
		return workspace.StatusCreated, nil
	}

	sawRunning := false
	sawNotRunning := false

	for _, s := range services {
		status, err := s.Service.Status(ctx)
		switch {
		case errors.Is(err, service.ErrNotFound):
			sawNotRunning = true
		case err != nil:
			return "", fmt.Errorf("engine: check %s service status: %w", s.Type, err)
		case status == service.StatusRunning:
			sawRunning = true
		case status == service.StatusError:
			return workspace.StatusError, nil
		default: // service.StatusCreated, service.StatusStopped
			sawNotRunning = true
		}
	}

	switch {
	case sawRunning && sawNotRunning:
		return workspace.StatusError, nil
	case sawRunning:
		return workspace.StatusRunning, nil
	default:
		return workspace.StatusStopped, nil
	}
}
