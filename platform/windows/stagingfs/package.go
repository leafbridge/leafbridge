package stagingfs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// PackageDir is a staging directory for a package in LeafBridge.
type PackageDir struct {
	content       lbdeploy.PackageContent
	packageType   lbdeploy.PackageType
	packageFormat lbdeploy.PackageFormat
	path          string
	dir           *os.Root
}

// Stat returns a [os.FileInfo] describing the package file with the given
// name.
func (d PackageDir) Stat(name string) (os.FileInfo, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(name)
	if err != nil {
		return nil, fmt.Errorf("localization of the package file name failed: %w", err)
	}

	return d.dir.Stat(localized)
}

// FilePath returns the absolute file path for the given package file name.
//
// It returns an error if the package file name is invalid.
func (d PackageDir) FilePath(name string) (string, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(name)
	if err != nil {
		return "", fmt.Errorf("localization of the package file name failed: %w", err)
	}

	return filepath.Join(d.path, localized), nil
}

// OpenFile opens the staging file for the given package file name.
//
// It is the caller's responsibility to close the file when finished with it.
func (d PackageDir) OpenFile(name string) (PackageFile, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(name)
	if err != nil {
		return PackageFile{}, fmt.Errorf("localization of the package file name failed: %w", err)
	}

	f, err := d.dir.OpenFile(localized, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return PackageFile{}, err
	}
	return PackageFile{
		name:          localized,
		packageType:   d.packageType,
		packageFormat: d.packageFormat,
		path:          filepath.Join(d.path, localized),
		file:          f,
	}, nil
}

// Close releases any file handles or resources held by the package
// staging directory.
func (d PackageDir) Close() error {
	return d.dir.Close()
}

// PackageFile is an open file for a package.
type PackageFile struct {
	name          string
	packageType   lbdeploy.PackageType
	packageFormat lbdeploy.PackageFormat
	path          string
	file          *os.File
}

// Name returns the name of the package file.
func (f PackageFile) Name() string {
	return f.name
}

// LocalizedPath returns the absolute path to the package file on the local system.
func (f PackageFile) LocalizedPath() string {
	return f.path
}

// System returns the underlying [os.File] for the package file.
func (f PackageFile) System() *os.File {
	return f.file
}

// Close releases any resources or system handles held by the package file.
func (f PackageFile) Close() error {
	return f.file.Close()
}
