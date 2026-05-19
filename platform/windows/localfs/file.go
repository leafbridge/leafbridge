package localfs

import (
	"errors"
	"fmt"
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
	// Examine the known folder's path, which is our starting point.
	if ref.Root.LocalizedPath == "" {
		return File{}, errors.New("the file reference has a root with an empty path")
	}

	// Start to build up the path of the file.
	path := ref.Root.LocalizedPath

	// Open the known folder as our first root directory.
	root, err := os.OpenRoot(ref.Root.LocalizedPath)
	if err != nil {
		return File{}, fmt.Errorf("failed to open known folder root path \"%s\": %w", ref.Root.LocalizedPath, err)
	}
	defer func() {
		// This is an intentional late evaluation of the root variable.
		root.Close()
	}()

	// Traverse subdirectories, if present.
	for _, next := range ref.Lineage {
		// Hold a copy of the parent's path in case we need to traverse
		// a symlink.
		parentPath := path

		// Continue buliding up the path of the file.
		path = filepath.Join(path, next.LocalizedPath)

		// Hold a reference to the parent so that we can close it in a moment.
		parent := root

		// Lstat the next directory so that we can determine whether it is a
		// symlink or not.
		fi, err := parent.Lstat(next.LocalizedPath)
		if err != nil {
			return File{}, fmt.Errorf("failed to examine subdirectory path \"%s\": %w", path, err)
		}

		if fi.Mode()&os.ModeSymlink == 0 {
			// Handle regular directories.

			// Make sure we weren't expecting a symlink.
			if next.Symlink {
				return File{}, fmt.Errorf("expected a symlink at path: %s", path)
			}

			// Traverse down to the next descendent.
			nextRoot, err := parent.OpenRoot(next.LocalizedPath)
			if err != nil {
				return File{}, fmt.Errorf("failed to open directory \"%s\": %w", path, err)
			}
			root = nextRoot

			// Always close the parent directory's file handle.
			parent.Close()
		} else {
			// Handle symlinks.

			// Make sure we were expecting a symlink.
			if !next.Symlink {
				return File{}, fmt.Errorf("encountered an unexpected symlink: %s", path)
			}

			// Read the destination.
			destination, err := parent.Readlink(next.LocalizedPath)
			if err != nil {
				return File{}, fmt.Errorf("failed to read symlink \"%s\": %w", path, err)
			}

			// Attempt to create a relative path to the destination from
			// the parent.
			relativeDestination, err := filepath.Rel(parentPath, destination)
			if err != nil {
				return File{}, fmt.Errorf("failed to build a relative path from symlink \"%s\" to \"%s\": %w", path, destination, err)
			}

			// Traverse down to the next descendent.
			nextRoot, err := parent.OpenRoot(relativeDestination)
			if err != nil {
				return File{}, fmt.Errorf("failed to open directory \"%s\": %w", path, err)
			}
			root = nextRoot

			// Always close the parent directory's file handle.
			parent.Close()
		}
	}

	// Finish constrution of the file's path.
	path = filepath.Join(path, ref.Resource.LocalizedPath)

	// Open the file.
	file, err := root.Open(ref.Resource.LocalizedPath)
	if err != nil {
		return File{}, err
	}

	// Return the file and its path.
	return File{
		file: file,
		path: path,
	}, nil
}

// LocalizedPath returns the path to the file on the local system.
func (f File) LocalizedPath() string {
	return f.path
}

// Stat returns the [os.FileInfo] structure describing file.
// If there is an error, it will be of type [*os.PathError].
func (f File) Stat() (os.FileInfo, error) {
	return f.file.Stat()
}

// System returns the underlying [os.File] for the file.
func (f File) System() *os.File {
	return f.file
}

// Close releases any resources or system handles held by the file.
func (f File) Close() error {
	return f.file.Close()
}
