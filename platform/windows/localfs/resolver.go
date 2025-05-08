package localfs

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"golang.org/x/sys/windows"
)

// Resolver is capable of locating file system resources on the local system.
type Resolver struct {
	fs lbdeploy.FileSystemResources
}

// NewResolver returns a new resolver for the given file system resources.
func NewResolver(resources lbdeploy.FileSystemResources) Resolver {
	return Resolver{fs: resources}
}

// ResolveKnownFolder looks for a known folder with the given directory
// resource ID. If a known folder with the given ID is not recognized,
// it returns [fs.ErrNotExist].
func (resolver *Resolver) ResolveKnownFolder(id lbdeploy.DirectoryResourceID) (lbdeploy.KnownFolder, error) {
	// Look up the folder by its directory resource ID.
	folder, ok := knownFolders[id]
	if !ok {
		return lbdeploy.KnownFolder{}, fs.ErrNotExist
	}

	// Ask the operating system for the known folder's path.
	path, err := windows.KnownFolderPath(folder.guid, 0)
	if err != nil {
		return lbdeploy.KnownFolder{}, fmt.Errorf("the \"%s\" known folder could not be resolved: %w", id, err)
	}

	return lbdeploy.KnownFolder{
		ID:        id,
		Path:      path,
		Protected: folder.protected,
	}, nil
}

// ResolveDirectory resolves the requested directory resource, returning a
// directory reference that can be mapped to a path on the local system.
//
// Successfully resolving a directory resource means that its path on the
// local system can be determined, but it does not imply that the directory
// exists.
//
// If the directory cannot be resolved, an error is returned.
func (resolver *Resolver) ResolveDirectory(id lbdeploy.DirectoryResourceID) (ref lbdeploy.DirRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the directory by its ID.
	data, exists := resolver.fs.Directories[id]
	if !exists {
		if candidate, err := resolver.ResolveKnownFolder(id); err == nil {
			return lbdeploy.DirRef{Root: candidate}, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return lbdeploy.DirRef{}, err
		}
		return lbdeploy.DirRef{}, fmt.Errorf("the \"%s\" directory is not defined in the deployment's resources", id)
	}

	// Make sure the directory has a location.
	if data.Location == "" {
		return lbdeploy.DirRef{}, fmt.Errorf("the \"%s\" directory does not have a location", id)
	}

	// Successful resolution must end in a known folder.
	var root lbdeploy.KnownFolder

	// Keep track of the directories we traverse, which will ultimately form
	// a lineage under the root.
	var lineage []lbdeploy.DirectoryResource

	// Maintain a map of directories we've encountered, so that we can detect
	// cycles.
	seen := make(lbdeploy.DirectoryResourceSet)

	// Start with the directory's location and traverse its ancestry,
	// recording each parent along the way. Stop when we encounter a known
	// folder.
	lineage = append(lineage, data)
	next := data.Location
	for {
		// Check for cycles.
		if seen.Contains(next) {
			return lbdeploy.DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory has a cyclic reference to itself in the deployment's resources", id, next)
		}
		seen.Add(next)

		// Look for a directory with the next directory ID.
		if parent, found := resolver.fs.Directories[next]; found {
			lineage = append(lineage, parent)
			if parent.Location == "" {
				return lbdeploy.DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory does not have a location", id, next)
			}
			next = parent.Location
			continue
		}

		// Look for a known folder with the ID.
		if candidate, err := resolver.ResolveKnownFolder(next); err == nil {
			root = candidate
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return lbdeploy.DirRef{}, err
		}

		// The location is not defined.
		return lbdeploy.DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory is not defined in the deployment's resources", id, next)
	}

	// Reverse the order of the directories that were recorded, so they can
	// easily be traversed from the root.
	slices.Reverse(lineage)

	return lbdeploy.DirRef{
		Root:    root,
		Lineage: lineage,
	}, nil
}

// ResolveFile resolves the requested file resource, returning a file
// reference that can be mapped to a path on the local system.
//
// Successfully resolving a file resource means that its path on the local
// system can be determined, but it does not imply that the file exists.
//
// If the file cannot be resolved, an error is returned.
func (resolver *Resolver) ResolveFile(id lbdeploy.FileResourceID) (ref lbdeploy.FileRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the file by its ID.
	data, exists := resolver.fs.Files[id]
	if !exists {
		return lbdeploy.FileRef{}, fmt.Errorf("the \"%s\" file is not defined in the deployment's resources", id)
	}

	// Make sure the file has a location.
	if data.Location == "" {
		return lbdeploy.FileRef{}, fmt.Errorf("the \"%s\" file does not have a location", id)
	}

	// Resolve the file's parent directory.
	dir, err := resolver.ResolveDirectory(data.Location)
	if err != nil {
		return lbdeploy.FileRef{}, fmt.Errorf("failed to resolve the \"%s\" file: %w", id, err)
	}

	return lbdeploy.FileRef{
		Root:     dir.Root,
		Lineage:  dir.Lineage,
		FileID:   id,
		FilePath: data.Path,
	}, nil
}
