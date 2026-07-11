package workspace

import "errors"

var (
	// ErrNotFound is returned when a Workspace does not exist.
	ErrNotFound = errors.New("workspace: not found")
	// ErrNameRequired is returned when a Workspace is created without a name.
	ErrNameRequired = errors.New("workspace: name is required")
	// ErrNameExists is returned when a Workspace name is already in use.
	ErrNameExists = errors.New("workspace: name already exists")
)
