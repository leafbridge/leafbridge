package localfs

import (
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// Dir is an open directory on the local file system.
type Dir struct {
	root *os.Root
	path string
}

// OpenDir attempts to open the directory identified by the given file reference.
func OpenDir(ref lbdeploy.DirRef) (Dir, error) {
	// Retrieve the known folder path, which is our starting point.
	knownFolderPath, err := ref.Root.Path()
	if err != nil {
		return Dir{}, err
	}

	// Start to build up the path of the directory.
	path := knownFolderPath

	// Open the known folder as our first root directory.
	root, err := os.OpenRoot(knownFolderPath)
	if err != nil {
		return Dir{}, err
	}

	// Traverse subdirectories, if present.
	for _, next := range ref.Lineage {
		// Continue buliding up the path of the directory.
		localized, err := filepath.Localize(next.Path)
		if err != nil {
			return Dir{}, err
		}
		path = filepath.Join(path, localized)

		// Hold a reference to the parent so that we can close it in a moment.
		parent := root

		// Traverse down to the next descendent.
		root, err = parent.OpenRoot(next.Path)

		// Always close the parent directory's file handle.
		parent.Close()

		// Stop if we were unable to traverse down.
		if err != nil {
			return Dir{}, err
		}
	}

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
