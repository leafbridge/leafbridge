package localfs

import (
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// File is an open file on the local file system.
type File struct {
	file *os.File
	path string
}

// OpenFile attempts to open the file identified by the given file reference.
func OpenFile(ref lbdeploy.FileRef) (File, error) {
	// Retrieve the known folder path, which is our starting point.
	knownFolderPath, err := ref.Root.Path()
	if err != nil {
		return File{}, err
	}

	// Start to build up the path of the file.
	path := knownFolderPath

	// Open the known folder as our root directory.
	root, err := os.OpenRoot(knownFolderPath)
	if err != nil {
		return File{}, err
	}

	// Traverse subdirectories, if present.
	for _, next := range ref.Lineage {
		// Continue buliding up the path of the file.
		localized, err := filepath.Localize(next.Path)
		if err != nil {
			return File{}, err
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
			return File{}, err
		}
	}

	// Now that we have the parent directory, but sure to close it when we're
	// finished.
	defer root.Close()

	// Finish constrution of the file's path.
	{
		localized, err := filepath.Localize(ref.FilePath)
		if err != nil {
			return File{}, err
		}
		path = filepath.Join(path, localized)
	}

	// Open the file.
	file, err := root.Open(ref.FilePath)
	if err != nil {
		return File{}, err
	}

	// Return the file and its path.
	return File{
		file: file,
		path: path,
	}, nil
}

// Path returns the path to the file on the local system.
func (f File) Path() string {
	return f.path
}

// System returns the underlying [os.File] for the file.
func (f File) System() *os.File {
	return f.file
}

// Close releases any resources or system handles held by the file.
func (f File) Close() error {
	return f.file.Close()
}
