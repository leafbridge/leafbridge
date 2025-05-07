package lbdeploy

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/leafbridge/leafbridge/core/idset"
	"github.com/leafbridge/leafbridge/core/lbvalue"
	"golang.org/x/sys/windows/registry"
)

// RegistryResources describes resources accessed through the Windows
// registry.
type RegistryResources struct {
	Keys   RegistryKeyResourceMap   `json:"keys,omitempty"`
	Values RegistryValueResourceMap `json:"values,omitempty"`
}

// ResolveKey resolves the requested registry key resource, returning a
// registry key reference that can be mapped to a location in the Windows
// registry.
//
// Successfully resolving a registry key resource means that its location
// in the Windows registry can be determined, but it does not imply that the
// key exists.
//
// If the registry key cannot be resolved, an error is returned.
func (reg RegistryResources) ResolveKey(key RegistryKeyResourceID) (ref RegistryKeyRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the registry key by its ID.
	data, exists := reg.Keys[key]
	if !exists {
		if candidate, found := GetRegistryRoot(key); found {
			return RegistryKeyRef{Root: candidate}, nil
		}
		return RegistryKeyRef{}, fmt.Errorf("the \"%s\" registry key is not defined in the deployment's resources", key)
	}

	// Make sure the registry key has a location.
	if data.Location == "" {
		return RegistryKeyRef{}, fmt.Errorf("the \"%s\" registry key does not have a location", key)
	}

	// Successful resolution must end in a known registry root.
	var root RegistryRoot

	// Keep track of the keys we traverse, which will ultimately form
	// a lineage under the root.
	var lineage []RegistryKeyResource

	// Maintain a map of registry keys we've encountered, so that we can
	// detect cycles.
	seen := make(RegistryKeyResourceSet)

	// Start with the registry key's location and traverse its ancestry,
	// recording each parent along the way. Stop when we encounter a registry
	// root.
	lineage = append(lineage, data)
	next := data.Location
	for {
		// Check for cycles.
		if seen.Contains(next) {
			return RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" parent key has a cyclic reference to itself in the deployment's registry resources", key, next)
		}
		seen.Add(next)

		// Look for a registry key with the ID.
		if parent, found := reg.Keys[next]; found {
			lineage = append(lineage, parent)
			if parent.Location == "" {
				return RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" parent key does not have a location", key, next)
			}
			next = parent.Location
			continue
		}

		// Look for a registry root with the ID.
		if candidate, found := GetRegistryRoot(next); found {
			root = candidate
			break
		}

		// The location is not defined.
		return RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" prent key is not defined in the deployment's resources", key, next)
	}

	// Reverse the order of the registry keys that were recorded, so they can
	// easily be traversed from the root.
	slices.Reverse(lineage)

	return RegistryKeyRef{
		Root:    root,
		Lineage: lineage,
	}, nil
}

// ResolveValue resolves the requested registry value resource, returning a
// registry value reference that can be mapped to a location in the Windows
// registry.
//
// Successfully resolving a registry value resource means that its location
// in the Windows registry can be determined, but it does not imply that the
// value exists.
//
// If the registry value cannot be resolved, an error is returned.
func (reg RegistryResources) ResolveValue(value RegistryValueResourceID) (ref RegistryValueRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the registry value by its ID.
	data, exists := reg.Values[value]
	if !exists {
		return RegistryValueRef{}, fmt.Errorf("the \"%s\" registry value is not defined in the deployment's resources", value)
	}

	// Make sure the registry value has a key.
	if data.Key == "" {
		return RegistryValueRef{}, fmt.Errorf("the \"%s\" registry value does not have a key", value)
	}

	// Resolve the value's registry key.
	key, err := reg.ResolveKey(data.Key)
	if err != nil {
		return RegistryValueRef{}, fmt.Errorf("failed to resolve the \"%s\" registry value: %w", value, err)
	}

	return RegistryValueRef{
		Root:    key.Root,
		Lineage: key.Lineage,
		ID:      value,
		Name:    data.Name,
		Type:    data.Type,
	}, nil
}

// RegistryKeyResourceMap holds a set of registry key resources mapped by
// their identifiers.
type RegistryKeyResourceMap map[RegistryKeyResourceID]RegistryKeyResource

// RegistryKeyResourceID is a unique identifier for a registry key.
type RegistryKeyResourceID string

// RegistryKeyResource describes a registry key in the Windows registry.
//
// Its name and path fields are mutually exclusive.
type RegistryKeyResource struct {
	// Location is a well-known registry root ID, or another key's
	// resource ID.
	Location RegistryKeyResourceID `json:"location,omitempty"`

	// Name is the name of the key within its location.
	Name string `json:"name,omitempty"`

	// Path is the relative path of the key within its location.
	// Both forward slashes and backslashes will be interpreted as path
	// separators.
	Path string `json:"path,omitempty"`
}

// RegistryKeyRef is a resolved reference to a registry key on the local
// system.
type RegistryKeyRef struct {
	Root    RegistryRoot
	Lineage []RegistryKeyResource
}

// Path returns the path of the registry key on the local system.
func (ref RegistryKeyRef) Path() (string, error) {
	path, err := ref.Root.AbsolutePath()
	if err != nil {
		return "", err
	}

	for _, key := range ref.Lineage {
		switch {
		case key.Name != "":
			path = path + `\` + key.Name
		case key.Path != "":
			localized, err := filepath.Localize(key.Path)
			if err != nil {
				return "", err
			}
			path = filepath.Join(path, localized)
		default:
			return "", fmt.Errorf("a registry key resource does not specify a name or path")
		}
	}

	return path, nil
}

// RegistryKeyResourceSet holds a set of registry key resource IDs.
type RegistryKeyResourceSet = idset.SetOf[RegistryKeyResourceID]

// RegistryValueResourceMap holds a set of registry value resources mapped by
// their identifiers.
type RegistryValueResourceMap map[RegistryValueResourceID]RegistryValueResource

// RegistryValueResourceID is a unique identifier for a registry value.
type RegistryValueResourceID string

// RegistryValueResource describes a value within the Windows registry.
type RegistryValueResource struct {
	// Key is the registry key resource ID of the key to which the value
	// belongs, or the well-known resource ID of a registry root.
	Key RegistryKeyResourceID `json:"key"`

	// Name is the name of the value within its registry key.
	Name string `json:"name"`

	// Type is the type of data the value holds.
	Type lbvalue.Kind `json:"type"`
}

// RegistryValueRef is a resolved reference to a registry key on the local
// system.
type RegistryValueRef struct {
	Root    RegistryRoot
	Lineage []RegistryKeyResource
	ID      RegistryValueResourceID
	Name    string
	Type    lbvalue.Kind
}

// Key returns a reference to the values's registry key.
func (ref RegistryValueRef) Key() RegistryKeyRef {
	return RegistryKeyRef{
		Root:    ref.Root,
		Lineage: ref.Lineage,
	}
}

// RegistryRootMap holds a set of registry roots mapped by their well-known
// identifiers.
type RegistryRootMap map[RegistryKeyResourceID]RegistryRoot

// RegistryRoot is a root location within the Windows registry.
type RegistryRoot struct {
	id   RegistryKeyResourceID
	key  registry.Key
	path string
}

// ID returns the resource ID of the registry root.
func (root RegistryRoot) ID() RegistryKeyResourceID {
	return root.id
}

// Key returns the predefined key used by the registry root.
func (root RegistryRoot) Key() registry.Key {
	return root.key
}

// Path retrieves the relative path to the root from its predefined key.
func (root RegistryRoot) Path() (path string) {
	return root.path
}

// AbsolutePath return the absolute path to the registry root on the
// local system, including the predefined key.
func (root RegistryRoot) AbsolutePath() (path string, err error) {
	switch root.key {
	case registry.LOCAL_MACHINE:
		path = "HKEY_LOCAL_MACHINE"
	default:
		return "", fmt.Errorf("the \"%s\" registry root relies on an unsupported root key", root.id)
	}
	if root.path != "" {
		path = filepath.Join(path, root.path)
	}
	return
}

// IsZero returns true if the registry root is undefined.
func (root RegistryRoot) IsZero() bool {
	return root.id == ""
}

// GetRegistryRoot looks for a well-known registry root with the given
// resource ID. If one is found, it is returned and ok will be true.
func GetRegistryRoot(id RegistryKeyResourceID) (root RegistryRoot, ok bool) {
	root, ok = registryRoots[id]
	return
}

var registryRoots = RegistryRootMap{
	"software": RegistryRoot{id: "software", key: registry.LOCAL_MACHINE, path: "SOFTWARE"},
}
