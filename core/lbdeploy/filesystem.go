package lbdeploy

import (
	"fmt"
	"path/filepath"

	"github.com/leafbridge/leafbridge/core/idset"
)

// FileSystemResources describes resources accessed through the file system,
// either local or remote.
type FileSystemResources struct {
	Directories DirectoryResourceMap `json:"directories,omitempty"`
	Files       FileResourceMap      `json:"files,omitempty"`
}

// DirectoryResourceMap holds a set of directory resources mapped by their
// identifiers.
type DirectoryResourceMap map[DirectoryResourceID]DirectoryResource

// DirectoryResourceID is a unique identifier for a directory resource.
type DirectoryResourceID string

// DirectoryType declares the type of a directory resource.
type DirectoryType string

// DirectoryResource describes a directory resource.
type DirectoryResource struct {
	Location DirectoryResourceID `json:"location"`         // A well-known directory, or another directory ID.
	Path     string              `json:"path"`             // Relative to location.
	Symlink  bool                `json:"symlink,omitzero"` // Is the directory expected to be a symlink?
}

// ResolvedDirectoryResource describes a directory resource that has been
// resolved.
type ResolvedDirectoryResource struct {
	ID            DirectoryResourceID `json:"id"`               // The ID of this directory.
	Location      DirectoryResourceID `json:"location"`         // A well-known directory, or another directory ID.
	Path          string              `json:"path"`             // Relative to its parent location.
	LocalizedPath string              `json:"localized-path"`   // Relative to its parent location.
	Symlink       bool                `json:"symlink,omitzero"` // Is the directory expected to be a symlink?
}

// DirRef is a resolved reference to a directory on the local file system.
type DirRef struct {
	Root    KnownFolder
	Lineage []ResolvedDirectoryResource
}

// Parent returns a reference to the previous directory in the lineage of ref.
//
// If ref itself is a root without any lineage an error is returned.
func (ref DirRef) Parent() (DirRef, error) {
	if len(ref.Lineage) == 0 {
		return DirRef{}, fmt.Errorf("failed to determine a parent for \"%s\": the directory reference already points to a root", ref.Root.ID)
	}
	return DirRef{
		Root:    ref.Root,
		Lineage: ref.Lineage[0 : len(ref.Lineage)-1],
	}, nil
}

// Resource returns the last directory resource in the lineage of ref.
//
// If ref itself is a root without any lineage an error is returned.
func (ref DirRef) Resource() (ResolvedDirectoryResource, error) {
	if len(ref.Lineage) == 0 {
		return ResolvedDirectoryResource{}, fmt.Errorf("a directory resource is not available for \"%s\": the directory is a root", ref.Root.ID)
	}
	return ref.Lineage[len(ref.Lineage)-1], nil
}

// LocalizedPath returns the absolute path of the directory on the local
// file system.
func (ref DirRef) LocalizedPath() string {
	path := ref.Root.LocalizedPath
	for _, dir := range ref.Lineage {
		path = filepath.Join(path, dir.LocalizedPath)
	}
	return path
}

// DirectoryResourceSet holds a set of directory resource IDs.
type DirectoryResourceSet = idset.SetOf[DirectoryResourceID]

// FileResourceMap holds a set of file resources mapped by their identifiers.
type FileResourceMap map[FileResourceID]FileResource

// FileResourceID is a unique identifier for a file resource.
type FileResourceID string

// FileResource describes a file resource.
type FileResource struct {
	Location DirectoryResourceID `json:"location"` // A well-known directory, or another directory ID.
	Path     string              `json:"path"`     // Relative to location
}

// ResolvedFileResource describes a file resource that has been resolved.
type ResolvedFileResource struct {
	ID            FileResourceID      `json:"id"`             // The ID of this file.
	Location      DirectoryResourceID `json:"location"`       // A well-known directory, or another directory ID.
	Path          string              `json:"path"`           // Relative to location
	LocalizedPath string              `json:"localized-path"` // Relative to location.
}

// FileRef is a resolved reference to a file on the local file system.
type FileRef struct {
	Root     KnownFolder
	Lineage  []ResolvedDirectoryResource
	Resource ResolvedFileResource
}

// Dir returns a reference to the file's directory.
func (ref FileRef) Dir() DirRef {
	return DirRef{
		Root:    ref.Root,
		Lineage: ref.Lineage,
	}
}

// LocalizedPath returns the absolute path of the file on the local
// file system.
func (ref FileRef) LocalizedPath() string {
	return filepath.Join(ref.Dir().LocalizedPath(), ref.Resource.LocalizedPath)
}

// KnownFolder is a folder with a known location.
type KnownFolder struct {
	ID            DirectoryResourceID
	LocalizedPath string
	Protected     bool

	// TODO: Create our own representation of a GUID that is suitable for
	// cross-platform use, then include it here.
	//guid      *windows.KNOWNFOLDERID
}

// FileVersionInfoError is returned when file version information cannot be
// retrieved.
type FileVersionInfoError struct {
	ID   FileResourceID
	Path string
	Err  error
}

// Unwrap returns the underlying error for the file version information
// retrieval.
func (e FileVersionInfoError) Unwrap() error {
	return e.Err
}

// Error returns the error as a string.
func (e FileVersionInfoError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("failed to read file version information for \"%s\": %s", e.ID, e.Err.Error())
	}
	return fmt.Sprintf("failed to read file version information for \"%s\" (%s): %s", e.ID, e.Path, e.Err.Error())
}
