package localregistry

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// Resolver is capable of locating registry resources on the local system.
type Resolver struct {
	reg lbdeploy.RegistryResources
}

// NewResolver returns a new resolver for the given registry resources.
func NewResolver(resources lbdeploy.RegistryResources) Resolver {
	return Resolver{reg: resources}
}

// ResolveRoot looks for a well-known registry root with the given registry
// key resource ID. If a registry root with the given ID is not recognized,
// it returns [fs.ErrNotExist].
func (resolver *Resolver) ResolveRoot(id lbdeploy.RegistryKeyResourceID) (lbdeploy.RegistryRoot, error) {
	// Look up the root by its registry key resource ID.
	root, ok := registryRoots[id]
	if !ok {
		return lbdeploy.RegistryRoot{}, fs.ErrNotExist
	}

	// Verify that the predefined key is valid.
	if _, err := PredefinedKeyHandle(root.key); err != nil {
		return lbdeploy.RegistryRoot{}, fmt.Errorf("the \"%s\" registry root could not be resolved: %w", id, err)
	}

	return lbdeploy.RegistryRoot{
		ID:            id,
		PredefinedKey: root.key,
		Path:          root.path,
	}, nil
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
func (resolver Resolver) ResolveKey(key lbdeploy.RegistryKeyResourceID) (ref lbdeploy.RegistryKeyRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the registry key by its ID.
	data, exists := resolver.reg.Keys[key]
	if !exists {
		if candidate, err := resolver.ResolveRoot(key); err == nil {
			return lbdeploy.RegistryKeyRef{Root: candidate}, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return lbdeploy.RegistryKeyRef{}, err
		}
		return lbdeploy.RegistryKeyRef{}, fmt.Errorf("the \"%s\" registry key is not defined in the deployment's resources", key)
	}

	// Make sure the registry key has a location.
	if data.Location == "" {
		return lbdeploy.RegistryKeyRef{}, fmt.Errorf("the \"%s\" registry key does not have a location", key)
	}

	// Successful resolution must end in a known registry root.
	var root lbdeploy.RegistryRoot

	// Keep track of the keys we traverse, which will ultimately form
	// a lineage under the root.
	var lineage []lbdeploy.RegistryKeyResource

	// Maintain a map of registry keys we've encountered, so that we can
	// detect cycles.
	seen := make(lbdeploy.RegistryKeyResourceSet)

	// Start with the registry key's location and traverse its ancestry,
	// recording each parent along the way. Stop when we encounter a registry
	// root.
	lineage = append(lineage, data)
	next := data.Location
	for {
		// Check for cycles.
		if seen.Contains(next) {
			return lbdeploy.RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" parent key has a cyclic reference to itself in the deployment's registry resources", key, next)
		}
		seen.Add(next)

		// Look for a registry key with the ID.
		if parent, found := resolver.reg.Keys[next]; found {
			lineage = append(lineage, parent)
			if parent.Location == "" {
				return lbdeploy.RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" parent key does not have a location", key, next)
			}
			next = parent.Location
			continue
		}

		// Look for a registry root with the ID.
		if candidate, err := resolver.ResolveRoot(next); err == nil {
			root = candidate
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return lbdeploy.RegistryKeyRef{}, err
		}

		// The location is not defined.
		return lbdeploy.RegistryKeyRef{}, fmt.Errorf("failed to resolve the \"%s\" registry key: the \"%s\" prent key is not defined in the deployment's resources", key, next)
	}

	// Reverse the order of the registry keys that were recorded, so they can
	// easily be traversed from the root.
	slices.Reverse(lineage)

	return lbdeploy.RegistryKeyRef{
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
func (resolver Resolver) ResolveValue(value lbdeploy.RegistryValueResourceID) (ref lbdeploy.RegistryValueRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the registry value by its ID.
	data, exists := resolver.reg.Values[value]
	if !exists {
		return lbdeploy.RegistryValueRef{}, fmt.Errorf("the \"%s\" registry value is not defined in the deployment's resources", value)
	}

	// Make sure the registry value has a key.
	if data.Key == "" {
		return lbdeploy.RegistryValueRef{}, fmt.Errorf("the \"%s\" registry value does not have a key", value)
	}

	// Resolve the value's registry key.
	key, err := resolver.ResolveKey(data.Key)
	if err != nil {
		return lbdeploy.RegistryValueRef{}, fmt.Errorf("failed to resolve the \"%s\" registry value: %w", value, err)
	}

	return lbdeploy.RegistryValueRef{
		Root:    key.Root,
		Lineage: key.Lineage,
		ID:      value,
		Name:    data.Name,
		Type:    data.Type,
	}, nil
}
