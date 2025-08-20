package tempfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/platform/windows/filetime"
)

// Options hold a set of options for extraction directories.
type Options struct {
	// DeleteOnClose requests that temporary directories and their contents
	// are deleted when the directory is closed.
	DeleteOnClose bool
}

// ExtractionDir is an extraction directory for a package in LeafBridge.
//
// It is a temporary directory created via os.MkdirTemp. Its name will have
// "leafbridge-" as a prefix.
type ExtractionDir struct {
	name   string
	path   string
	parent *os.Root
	dir    *os.Root
	opts   Options
}

// OpenExtractionDirForPackage opens a temporary directory to receive
// extracted files from a package.
//
// It is the caller's responsibility to close the returned directory when
// finished with it.
//
// The options can be used to request that the returned directory is deleted
// when closed.
//
// TODO: Make the options variadic.
func OpenExtractionDirForPackage(pkg lbdeploy.PackageContent, opts Options) (ExtractionDir, error) {
	// Unfortunately, this returns a path instead of an open directory handle.
	dirPath, err := os.MkdirTemp("", "leafbridge-"+pkg.String())
	if err != nil {
		return ExtractionDir{}, err
	}

	// Sanity check the directory path to make sure it conforms to our
	// expectations. If it doesn't, then return an error.
	//
	// Note that We might call RemoveAll() on the path later, and we really
	// don't want to make that call on an unintended path, especially when
	// operating with SYSTEM privileges.
	{
		dirPath := strings.ToLower(dirPath) // Case-insensitive search
		if !strings.Contains(dirPath, "leafbridge") || !strings.Contains(dirPath, "temp") {
			return ExtractionDir{}, fmt.Errorf("the os.MkdirTemp call failed to create a directory with the expected format: %s", dirPath)
		}
	}

	// Keep track of whether this call succeeds. This is referenced by
	// deferred functions to close open file handles in the case of failure.
	success := false

	// Get the path to the parent directory.
	parentPath := filepath.Dir(dirPath)

	// Get the name of the temp directory that was created.
	dirName := filepath.Base(dirPath)

	// Open the parent of the newly created temp directory as its own root.
	//
	// Having a separate root for the parent will be useful later, as it
	// will let us call parent.RemoveAll() when we close and delete the
	// temp directory.
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return ExtractionDir{}, err
	}
	defer func() {
		if !success {
			parent.Close()
		}
	}()

	// Open the root of the newly created temp directory.
	dir, err := parent.OpenRoot(dirName)
	if err != nil {
		return ExtractionDir{}, err
	}
	defer func() {
		if !success {
			dir.Close()
		}
	}()

	// Verify that the directory we opened via the parent is in fact the
	// temp directory that we created.
	fi1, err := parent.Stat(dirName)
	if err != nil {
		return ExtractionDir{}, fmt.Errorf("failed to stat the temporary directory \"%s\" via its parent: %w", dirPath, err)
	}

	fi2, err := os.Stat(dirPath)
	if err != nil {
		return ExtractionDir{}, fmt.Errorf("failed to stat the temporary directory \"%s\" via its absolute path: %w", dirPath, err)
	}

	if !os.SameFile(fi1, fi2) {
		return ExtractionDir{}, fmt.Errorf("failed to open the temporary directory \"%s\": the opened directory is not the same as the one that was created", dirPath)
	}

	// Indicate success so that we don't close the open file handles.
	success = true

	// Return the extraction directory.
	return ExtractionDir{
		name:   dirName,
		path:   dirPath,
		parent: parent,
		dir:    dir,
		opts:   opts,
	}, nil
}

// Path returns the path to the extraction directory at the time of its
// creation.
func (d ExtractionDir) Path() string {
	return d.path
}

// MkdirAll ensures that the given relative directory path and all of its
// parents have been created within the extraction directory.
//
// If name does not identify a local file path, or if directory creation
// fails, it rturns an error.
func (d ExtractionDir) MkdirAll(path string) error {
	// Removing trailing path separators, which are present at the end of
	// directory paths in zip files.
	path = strings.TrimSuffix(path, "/")

	// Localize the directory path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(path)
	if err != nil {
		return fmt.Errorf("localization of the directory path failed: %w", err)
	}

	// Create the directory and any of it ancestors that don't already exist.
	if err := d.dir.MkdirAll(localized, 0755); err != nil {
		return fmt.Errorf("failed to create directory path within \"%s\": %w", d.path, err)
	}

	return nil
}

// FilePath returns the absolute file path for the requested file.
//
// It returns an error if the given path is not relative.
func (d ExtractionDir) FilePath(path string) (string, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(path)
	if err != nil {
		return "", fmt.Errorf("localization of the file path failed: %w", err)
	}

	return filepath.Join(d.path, localized), nil
}

// Stat returns a [os.FileInfo] describing the named file in the root.
func (d ExtractionDir) Stat(path string) (os.FileInfo, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(path)
	if err != nil {
		return nil, fmt.Errorf("localization of the file path failed: %w", err)
	}

	return d.dir.Stat(localized)
}

// WriteFile reads data from r and writes it to the provided relative file
// path. It continues until the reader returns io.EOF or an error is
// encountered.
//
// If a non-zero modified time is provided, it is set as the file's
// modification time.
//
// The standard unix file separator, forward slash (/), must be used as the
// separator in the provided path.
func (d ExtractionDir) WriteFile(path string, r io.Reader, modified time.Time) (written int64, err error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(path)
	if err != nil {
		return 0, fmt.Errorf("localization of the file path failed: %w", err)
	}

	// If this file is in a subdirectory, open its parent.
	dirPath, fileName := filepath.Split(localized)
	var parent *os.Root
	if dirPath != "" {
		parent, err = d.dir.OpenRoot(dirPath)
		if err != nil {
			return 0, fmt.Errorf("failed to open parent directory: %w", err)
		}
		defer parent.Close()
	} else {
		parent = d.dir
	}

	// Create the file.
	file, err := parent.Create(fileName)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write the file content.
	written, err = io.Copy(file, r)
	if err != nil {
		return written, err
	}

	// Preserve the modification date, if available.
	if !modified.IsZero() {
		if err := filetime.SetFileModificationTime(file, modified); err != nil {
			return written, fmt.Errorf("failed to set modification time: %w", err)
		}
	}

	return written, err
}

// Close releases any file system resources consumed by the directory.
//
// If the directory was created with the DeleteOnClose option, calling this
// function will cause the directory and all of its contents to be deleted.
func (d ExtractionDir) Close() error {
	// Simple closure.
	if !d.opts.DeleteOnClose {
		return errors.Join(d.dir.Close(), d.parent.Close())
	}

	// Close and delete the temp directory.
	err1 := d.dir.Close()
	err2 := d.parent.RemoveAll(d.name)

	// Close the parent directory.
	err3 := d.parent.Close()

	return errors.Join(err1, err2, err3)
}
