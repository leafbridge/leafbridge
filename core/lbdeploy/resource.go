package lbdeploy

import "fmt"

// Resources defines the set of resources used by a deployment, both local
// and remote.
type Resources struct {
	Processes  ProcessResourceMap  `json:"processes,omitzero"`
	Mutexes    MutexMap            `json:"mutexes,omitzero"`
	Locks      LockMap             `json:"locks,omitzero"`
	Registry   RegistryResources   `json:"registry,omitzero"`
	FileSystem FileSystemResources `json:"file-system,omitzero"`
	Packages   PackageMap          `json:"packages,omitzero"`
}

// Validate returns a non-nil error if the deployment ID is invalid.
func (resources Resources) Validate() error {
	for id, p := range resources.Packages {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("package %s: %w", id, err)
		}
	}
	return nil
}
