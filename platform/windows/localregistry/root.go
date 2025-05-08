package localregistry

import (
	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// registryRootMap holds a set of registry roots mapped by their well-known
// identifiers.
type registryRootMap map[lbdeploy.RegistryKeyResourceID]registryRoot

// registryRoot holds the predefined key and path for a registry root in
// Windows.
type registryRoot struct {
	key  lbdeploy.PredefinedRegistryKey
	path string
}

// Registry roots that are recognized by their well-known resource IDs.
var registryRoots = registryRootMap{
	"software": registryRoot{key: lbdeploy.PredefinedKeyLocalMachine, path: "SOFTWARE"},
}
