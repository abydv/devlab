// Package template defines the Template catalog: declarative,
// on-disk definitions of the Services a Workspace is created with.
// Templates are data, not code — see templates/ for the definitions
// shipped with DevLab.
package template

import "strings"

// Template is a named, reusable definition of the Services a Workspace
// should be created with.
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Services    []string `json:"services"`
}

func (t *Template) validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return ErrNameRequired
	}
	if len(t.Services) == 0 {
		return ErrServicesRequired
	}
	for _, svc := range t.Services {
		if strings.TrimSpace(svc) == "" {
			return ErrServicesRequired
		}
	}
	return nil
}
