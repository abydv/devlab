package service

// Service type identifiers, matching the examples in the Service
// Rules section of CLAUDE.md. Not every type has a concrete
// implementation yet — see internal/service/factory.
const (
	TypeKubernetes = "kubernetes"
	TypeDocker     = "docker"
	TypeJenkins    = "jenkins"
	TypeLinux      = "linux"
	TypeTerraform  = "terraform"
	TypeAnsible    = "ansible"
)

// KnownTypes lists every recognized service type identifier.
var KnownTypes = []string{
	TypeKubernetes,
	TypeDocker,
	TypeJenkins,
	TypeLinux,
	TypeTerraform,
	TypeAnsible,
}

// IsKnownType reports whether serviceType is a recognized service type.
func IsKnownType(serviceType string) bool {
	for _, t := range KnownTypes {
		if t == serviceType {
			return true
		}
	}
	return false
}
