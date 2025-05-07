package localregistry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/leafbridge/leafbridge/core/datatype"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbvalue"
	"golang.org/x/sys/windows/registry"
)

// Key is an open registry key on the local file system.
type Key struct {
	key        registry.Key
	path       string
	predefined bool
}

// OpenKey attempts to open the regisry key identified by the given registry
// key reference.
func OpenKey(ref lbdeploy.RegistryKeyRef) (Key, error) {
	// Make sure the root is valid.
	if ref.Root.IsZero() {
		return Key{}, errors.New("unable to open registry key: an empty root was provided in the key reference")
	}

	// Start to build up the path of the key.
	path, err := ref.Root.AbsolutePath()
	if err != nil {
		return Key{}, err
	}

	// Open the root's path relative to a predefined key. If the root does
	// not specify a path, this will return the predefined key.
	key, err := registry.OpenKey(ref.Root.Key(), ref.Root.Path(), registry.QUERY_VALUE)
	if err != nil {
		return Key{}, err
	}

	// Keep track of whether the key we return is predefined or not.
	predefined := key == ref.Root.Key()

	// Traverse subkeys, if present.
	for _, next := range ref.Lineage {
		// Hold a reference to the parent so that we can close it in a moment.
		parent := key

		// Traverse down to the next descendent.
		switch {
		case next.Name != "":
			key, err = registry.OpenKey(parent, next.Name, registry.QUERY_VALUE)
			path = path + `\` + next.Name // Permit forward slashes
		case next.Path != "":
			var localized string
			localized, err = filepath.Localize(next.Path)
			if err == nil {
				key, err = registry.OpenKey(parent, localized, registry.QUERY_VALUE)
				path = filepath.Join(path, localized)
			}
		default:
			err = errors.New("a registry key resource does not specify a name or path")
		}

		// Always close the parent key's registry handle, unless it's a
		// predefined key.
		if !predefined {
			parent.Close()
		}

		// Stop if we were unable to traverse down.
		if err != nil {
			return Key{}, err
		}

		// We've successfully traversed down from a predefined key.
		predefined = false
	}

	// Return the final registry key and its path.
	return Key{
		key:        key,
		path:       path,
		predefined: false,
	}, nil
}

// Path returns the path to the registry key on the local system.
func (key Key) Path() string {
	return key.path
}

// System returns the underlying [registry.Key].
func (key Key) System() registry.Key {
	return key.key
}

// Close releases any resources or system handles held by the registry key.
func (key Key) Close() error {
	// It's very unlikely that we'd end up with a predefined key, but don't
	// close predefined keys if we do.
	if key.predefined {
		return nil
	}
	return key.key.Close()
}

// HasValue returns true if the registry key has a value with the given name.
func (key Key) HasValue(name string) (bool, error) {
	_, _, err := key.key.GetValue(name, nil)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetValue retrieves a value from the registry key with the requested type.
func (key Key) GetValue(name string, kind lbvalue.Kind) (lbvalue.Value, error) {
	switch kind {
	case lbvalue.KindBool:
		valueAsString, _, err := key.key.GetStringValue(name)
		if err != nil {
			return lbvalue.Value{}, err
		}
		value, err := strconv.ParseBool(valueAsString)
		if err != nil {
			return lbvalue.Value{}, err
		}
		return lbvalue.Bool(value), nil
	case lbvalue.KindInt64:
		value, _, err := key.key.GetIntegerValue(name)
		if err != nil {
			return lbvalue.Value{}, err
		}
		return lbvalue.Int64(int64(value)), nil
	case lbvalue.KindString:
		value, _, err := key.key.GetStringValue(name)
		if err != nil {
			return lbvalue.Value{}, err
		}
		return lbvalue.String(value), nil
	case lbvalue.KindVersion:
		value, _, err := key.key.GetStringValue(name)
		if err != nil {
			return lbvalue.Value{}, err
		}
		return lbvalue.Version(datatype.Version(value)), nil
	default:
		return lbvalue.Value{}, fmt.Errorf("unable to retrieve \"%s\" registry value: \"%s\" is not a regognized variable type", name, kind)
	}
}
