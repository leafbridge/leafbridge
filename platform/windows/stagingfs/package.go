package stagingfs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// PackageDir is a staging directory for a package in LeafBridge.
type PackageDir struct {
	content lbdeploy.PackageContent
	path    string
	dir     *os.Root
}

// Stat returns a [os.FileInfo] describing the package file.
func (d PackageDir) Stat(pkg lbdeploy.Package) (os.FileInfo, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(pkg.FileName())
	if err != nil {
		return nil, fmt.Errorf("localization of the package file name failed: %w", err)
	}

	return d.dir.Stat(localized)
}

// FilePath returns the absolute file path for the requested package.
//
// It returns an error if the package file name is invalid.
func (d PackageDir) FilePath(pkg lbdeploy.Package) (string, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(pkg.FileName())
	if err != nil {
		return "", fmt.Errorf("localization of the package file name failed: %w", err)
	}

	return filepath.Join(d.path, localized), nil
}

// OpenFile opens the staging file for the given package.
//
// It is the caller's responsibility to close the file when finished with it.
func (d PackageDir) OpenFile(pkg lbdeploy.Package) (PackageFile, error) {
	// Localize the file path, which ensures that it conforms to the
	// local file system path separators and is in fact a relative path.
	localized, err := filepath.Localize(pkg.FileName())
	if err != nil {
		return PackageFile{}, fmt.Errorf("localization of the package file name failed: %w", err)
	}

	f, err := d.dir.OpenFile(localized, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return PackageFile{}, err
	}
	return PackageFile{
		Name:   localized,
		Type:   pkg.Type,
		Format: pkg.Format,
		Path:   filepath.Join(d.path, localized),
		File:   f,
	}, nil
}

// Close releases any file handles or resources held by the package
// staging directory.
func (d PackageDir) Close() error {
	return d.dir.Close()
}

// PackageFile is an open file for a package.
type PackageFile struct {
	Name   string
	Type   lbdeploy.PackageType
	Format lbdeploy.PackageFormat
	Path   string
	*os.File
}
