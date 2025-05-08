package lbdeploy

import (
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

// FileResource describes a directory resource.
type DirectoryResource struct {
	Location DirectoryResourceID // A well-known directory, or another directory ID.
	Path     string              // Relative to location
}

// DirRef is a resolved reference to a directory on the local file system.
type DirRef struct {
	Root    KnownFolder
	Lineage []DirectoryResource
}

// Path returns the path of the directory on the local file system.
func (ref DirRef) Path() (string, error) {
	path := ref.Root.Path
	for _, dir := range ref.Lineage {
		localized, err := filepath.Localize(dir.Path)
		if err != nil {
			return "", err
		}
		path = filepath.Join(path, localized)
	}

	return path, nil
}

// DirectoryResourceSet holds a set of directory resource IDs.
type DirectoryResourceSet = idset.SetOf[DirectoryResourceID]

// FileResourceMap holds a set of file resources mapped by their identifiers.
type FileResourceMap map[FileResourceID]FileResource

// FileResourceID is a unique identifier for a file resource.
type FileResourceID string

// FileResource describes a file resource.
type FileResource struct {
	Location DirectoryResourceID // A well-known directory, or another directory ID.
	Path     string              // Relative to location
}

// FileRef is a resolved reference to a file on the local file system.
type FileRef struct {
	Root     KnownFolder
	Lineage  []DirectoryResource
	FileID   FileResourceID
	FilePath string
}

// Dir returns a reference to the file's directory.
func (ref FileRef) Dir() DirRef {
	return DirRef{
		Root:    ref.Root,
		Lineage: ref.Lineage,
	}
}

// Path returns the path of the file on the local file system.
func (ref FileRef) Path() (string, error) {
	path, err := ref.Dir().Path()
	if err != nil {
		return "", err
	}

	localized, err := filepath.Localize(ref.FilePath)
	if err != nil {
		return "", err
	}

	return filepath.Join(path, localized), nil
}

// KnownFolder is a folder with a known location.
type KnownFolder struct {
	ID        DirectoryResourceID
	Path      string
	Protected bool

	// TODO: Create our own representation of a GUID that is suitable for
	// cross-platform use, then include it here.
	//guid      *windows.KNOWNFOLDERID
}
