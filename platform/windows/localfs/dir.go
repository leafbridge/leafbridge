package localfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// Dir is an open directory on the local file system.
type Dir struct {
	root *os.Root
	path string
}

// OpenDir attempts to open the directory identified by the given directory
// reference.
func OpenDir(ref lbdeploy.DirRef) (Dir, error) {
	// Examine the known folder's path, which is our starting point.
	if ref.Root.Path == "" {
		return Dir{}, errors.New("the directory reference has a root with an empty path")
	}

	// Keep track of whether this call succeeds. This is referenced by
	// deferred functions to close open file handles in the case of failure.
	success := false

	// Start to build up the path of the directory.
	path := ref.Root.Path

	// Open the known folder as our first root directory.
	root, err := os.OpenRoot(ref.Root.Path)
	if err != nil {
		return Dir{}, fmt.Errorf("failed to open known folder root path \"%s\": %w", ref.Root.Path, err)
	}
	defer func() {
		if !success {
			root.Close()
		}
	}()

	// Traverse subdirectories, if present.
	for _, next := range ref.Lineage {
		// Hold a copy of the parent's path in case we need to traverse
		// a symlink.
		parentPath := path

		// Continue buliding up the path of the directory.
		nextPathLocalized, err := filepath.Localize(next.Path)
		if err != nil {
			return Dir{}, fmt.Errorf("failed to localize subdirectory path for \"%s\": %w", parentPath, err)
		}
		path = filepath.Join(path, nextPathLocalized)

		// Hold a reference to the parent so that we can close it in a moment.
		parent := root

		// Lstat the next directory so that we can determine whether it is a
		// symlink or not.
		fi, err := parent.Lstat(nextPathLocalized)
		if err != nil {
			return Dir{}, fmt.Errorf("failed to examine subdirectory path \"%s\": %w", path, err)
		}

		if fi.Mode()&os.ModeSymlink == 0 {
			// Handle regular directories.

			// Make sure we weren't expecting a symlink.
			if next.Symlink {
				return Dir{}, fmt.Errorf("expected a symlink at path: %s", path)
			}

			// Traverse down to the next descendent.
			nextRoot, err := parent.OpenRoot(nextPathLocalized)
			if err != nil {
				return Dir{}, fmt.Errorf("failed to open directory \"%s\": %w", path, err)
			}
			root = nextRoot

			// Always close the parent directory's file handle.
			parent.Close()
		} else {
			// Handle symlinks.

			// Make sure we were expecting a symlink.
			if !next.Symlink {
				return Dir{}, fmt.Errorf("encountered an unexpected symlink: %s", path)
			}

			// Read the destination.
			destination, err := parent.Readlink(nextPathLocalized)
			if err != nil {
				return Dir{}, fmt.Errorf("failed to read symlink \"%s\": %w", path, err)
			}

			// Attempt to create a relative path to the destination from
			// the parent.
			relativeDestination, err := filepath.Rel(parentPath, destination)
			if err != nil {
				return Dir{}, fmt.Errorf("failed to build a relative path from symlink \"%s\" to \"%s\": %w", path, destination, err)
			}

			// Traverse down to the next descendent.
			nextRoot, err := parent.OpenRoot(relativeDestination)
			if err != nil {
				return Dir{}, fmt.Errorf("failed to open directory \"%s\": %w", path, err)
			}
			root = nextRoot

			// Always close the parent directory's file handle.
			parent.Close()
		}
	}

	// Indicate success so that we don't close the open file handles.
	success = true

	// Return the final directory and its path.
	return Dir{
		root: root,
		path: path,
	}, nil
}

// Path returns the path to the directory on the local system.
func (d Dir) Path() string {
	return d.path
}

// System returns the underlying [os.Root] for the directory.
func (d Dir) System() *os.Root {
	return d.root
}

// Close releases any resources or system handles held by the directory.
func (d Dir) Close() error {
	return d.root.Close()
}
