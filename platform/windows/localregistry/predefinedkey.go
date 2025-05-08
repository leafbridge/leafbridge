package localregistry

import (
	"fmt"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"golang.org/x/sys/windows/registry"
)

// PredefinedKeyHandle returns the Windows registry key handle for a
// predefined key. If the key is not recognized or supported, it returns
// an error.
//
// The handle that is returned is always open and does not need to be closed.
func PredefinedKeyHandle(key lbdeploy.PredefinedRegistryKey) (registry.Key, error) {
	switch key {
	case lbdeploy.PredefinedKeyLocalMachine:
		return registry.LOCAL_MACHINE, nil
	}

	return 0, fmt.Errorf("the predefined registry key is unrecognized or unsupported: %s", key)
}
