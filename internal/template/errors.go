package template

import "errors"

var (
	// ErrNotFound is returned when a Template does not exist.
	ErrNotFound = errors.New("template: not found")
	// ErrNameRequired is returned when a Template definition has no name.
	ErrNameRequired = errors.New("template: name is required")
	// ErrNameExists is returned when two Template definitions share a name.
	ErrNameExists = errors.New("template: name already exists")
	// ErrServicesRequired is returned when a Template defines no Services.
	ErrServicesRequired = errors.New("template: at least one service is required")
)
