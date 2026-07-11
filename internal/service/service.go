// Package service defines the contract every workspace Service
// implementation satisfies. Services use a Runtime — they never
// execute operating system commands directly.
package service

import "context"

// Status represents the lifecycle state of a Service.
type Status string

const (
	// StatusCreated is the state of a Service immediately after Create,
	// before Start has been called.
	StatusCreated Status = "created"
	// StatusRunning indicates the Service is running.
	StatusRunning Status = "running"
	// StatusStopped indicates the Service is stopped but not deleted.
	StatusStopped Status = "stopped"
	// StatusError indicates the Service is in an unexpected state.
	StatusError Status = "error"
)

// Service is implemented by every workspace service type (Kubernetes,
// Docker, Jenkins, Linux, Terraform, Ansible, ...).
type Service interface {
	// Create provisions the Service's underlying resource.
	Create(ctx context.Context) error
	// Start starts the Service.
	Start(ctx context.Context) error
	// Stop stops the Service without discarding its resource.
	Stop(ctx context.Context) error
	// Reset discards the Service's resource and recreates it from scratch.
	Reset(ctx context.Context) error
	// Delete permanently removes the Service's resource.
	Delete(ctx context.Context) error
	// Status reports the Service's current lifecycle state.
	Status(ctx context.Context) (Status, error)
	// Logs returns the Service's current logs.
	Logs(ctx context.Context) (string, error)
}
