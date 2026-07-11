// Package workspace defines the Workspace domain model and the Workspace
// Manager responsible for its lifecycle on disk.
package workspace

import "time"

// Status represents the lifecycle state of a Workspace.
type Status string

const (
	// StatusCreated is the state of a Workspace immediately after creation,
	// before any of its Services have been started.
	StatusCreated Status = "created"
	// StatusRunning indicates the Workspace's Services are running.
	StatusRunning Status = "running"
	// StatusStopped indicates the Workspace's Services are stopped.
	StatusStopped Status = "stopped"
	// StatusError indicates the Workspace failed to reach the requested state.
	StatusError Status = "error"
)

// Workspace is the top-level unit of isolation in DevLab. It groups one
// or more Services created from a Template.
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Template    string    `json:"template"`
	Services    []string  `json:"services"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
