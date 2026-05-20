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
	if ref.Root.LocalizedPath == "" {
		return Dir{}, errors.New("the directory reference has a root with an empty path")
	}

	// Keep track of whether this call succeeds. This is referenced by
	// deferred functions to close open file handles in the case of failure.
	success := false

	// Start to build up the path of the directory.
	path := ref.Root.LocalizedPath

	// Open the known folder as our first root directory.
	root, err := os.OpenRoot(ref.Root.LocalizedPath)
	if err != nil {
		return Dir{}, fmt.Errorf("failed to open known folder root path \"%s\": %w", ref.Root.LocalizedPath, err)
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
		path = filepath.Join(path, next.LocalizedPath)

		// Hold a reference to the parent so that we can close it in a moment.
		parent := root

		// Lstat the next directory so that we can determine whether it is a
		// symlink or not.
		fi, err := parent.Lstat(next.LocalizedPath)
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
			nextRoot, err := parent.OpenRoot(next.LocalizedPath)
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
			destination, err := parent.Readlink(next.LocalizedPath)
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

// LocalizedPath returns the absolute path to the directory on the local
// system.
func (d Dir) LocalizedPath() string {
	return d.path
}

// Stat returns a [os.FileInfo] describing the named file in the directory.
// See [os.Stat] for more details.
func (d Dir) Stat(name string) (os.FileInfo, error) {
	return d.root.Stat(name)
}

// Create creates or truncates the named file in the directory.
// See [os.Create] for more details.
func (d Dir) Create(name string) (*os.File, error) {
	return d.root.Create(name)
}

// Remove removes the named file or (empty) directory in the directory.
// See [os.Remove] for more details.
func (d Dir) Remove(name string) error {
	return d.root.Remove(name)
}

// RemoveAll removes the named file or directory and any children that it
// contains. See [os.RemoveAll] for more details.
func (d Dir) RemoveAll(name string) error {
	return d.root.RemoveAll(name)
}

// System returns the underlying [os.Root] for the directory.
func (d Dir) System() *os.Root {
	return d.root
}

// Close releases any resources or system handles held by the directory.
func (d Dir) Close() error {
	return d.root.Close()
}
