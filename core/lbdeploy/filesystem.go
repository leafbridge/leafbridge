package lbdeploy

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/leafbridge/leafbridge/core/idset"
	"golang.org/x/sys/windows"
)

// FileSystemResources describes resources accessed through the file system,
// either local or remote.
type FileSystemResources struct {
	Directories DirectoryResourceMap `json:"directories,omitempty"`
	Files       FileResourceMap      `json:"files,omitempty"`
}

// ResolveDirectory resolves the requested directory resource, returning a
// directory reference that can be mapped to a path on the local system.
//
// Successfully resolving a directory resource means that its path on the
// local system can be determined, but it does not imply that the directory
// exists.
//
// If the directory cannot be resolved, an error is returned.
func (fs FileSystemResources) ResolveDirectory(dir DirectoryResourceID) (ref DirRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the directory by its ID.
	data, exists := fs.Directories[dir]
	if !exists {
		if candidate, found := GetKnownFolder(dir); found {
			return DirRef{Root: candidate}, nil
		}
		return DirRef{}, fmt.Errorf("the \"%s\" directory is not defined in the deployment's resources", dir)
	}

	// Make sure the directory has a location.
	if data.Location == "" {
		return DirRef{}, fmt.Errorf("the \"%s\" directory does not have a location", dir)
	}

	// Successful resolution must end in a known folder.
	var root KnownFolder

	// Keep track of the directories we traverse, which will ultimately form
	// a lineage under the root.
	var lineage []DirectoryResource

	// Maintain a map of directories we've encountered, so that we can detect
	// cycles.
	seen := make(DirectoryResourceSet)

	// Start with the directory's location and traverse its ancestry,
	// recording each parent along the way. Stop when we encounter a known
	// folder.
	lineage = append(lineage, data)
	next := data.Location
	for {
		// Check for cycles.
		if seen.Contains(next) {
			return DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory has a cyclic reference to itself in the deployment's resources", dir, next)
		}
		seen.Add(next)

		// Look for a directory with the next directory ID.
		if parent, found := fs.Directories[next]; found {
			lineage = append(lineage, parent)
			if parent.Location == "" {
				return DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory does not have a location", dir, next)
			}
			next = parent.Location
			continue
		}

		// Look for a known folder with the ID.
		if candidate, found := GetKnownFolder(next); found {
			root = candidate
			break
		}

		// The location is not defined.
		return DirRef{}, fmt.Errorf("failed to resolve the \"%s\" directory: the \"%s\" parent directory is not defined in the deployment's resources", dir, next)
	}

	// Reverse the order of the directories that were recorded, so they can
	// easily be traversed from the root.
	slices.Reverse(lineage)

	return DirRef{
		Root:    root,
		Lineage: lineage,
	}, nil
}

// ResolveFile resolves the requested file resource, returning a file
// reference that can be mapped to a path on the local system.
//
// Successfully resolving a file resource means that its path on the local
// system can be determined, but it does not imply that the file exists.
//
// If the file cannot be resolved, an error is returned.
func (fs FileSystemResources) ResolveFile(file FileResourceID) (ref FileRef, err error) {
	// TODO: Consider making custom error types for resolution.

	// Look up the file by its ID.
	data, exists := fs.Files[file]
	if !exists {
		return FileRef{}, fmt.Errorf("the \"%s\" file is not defined in the deployment's resources", file)
	}

	// Make sure the file has a location.
	if data.Location == "" {
		return FileRef{}, fmt.Errorf("the \"%s\" file does not have a location", file)
	}

	// Resolve the file's parent directory.
	dir, err := fs.ResolveDirectory(data.Location)
	if err != nil {
		return FileRef{}, fmt.Errorf("failed to resolve the \"%s\" file: %w", file, err)
	}

	return FileRef{
		Root:     dir.Root,
		Lineage:  dir.Lineage,
		FileID:   file,
		FilePath: data.Path,
	}, nil
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
	root, err := ref.Root.Path()
	if err != nil {
		return "", err
	}

	path := root
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

// KnownFolderMap is a map of predefined directory resource IDs to known
// folder locations.
type KnownFolderMap map[DirectoryResourceID]KnownFolder

// KnownFolder is a folder with a known location.
type KnownFolder struct {
	id        DirectoryResourceID
	guid      *windows.KNOWNFOLDERID
	protected bool
}

// ID returns the LeafBridge directory ID of the known folder.
func (kf KnownFolder) ID() DirectoryResourceID {
	return kf.id
}

// GUID returns the Known Folder ID in Windows.
func (kf KnownFolder) GUID() *windows.KNOWNFOLDERID {
	return kf.guid
}

// IsZero returns true if the known folder is undefined.
func (kf KnownFolder) IsZero() bool {
	return kf.guid == nil
}

// Protected returns true if the known folder is protected against
// modification.
func (kf KnownFolder) Protected() bool {
	return kf.protected
}

// Path retrieves the path to the known folder on the local system.
func (kf KnownFolder) Path() (path string, err error) {
	path, err = windows.KnownFolderPath(kf.guid, 0)
	return
}

// GetKnownFolder looks for a known folder with the given directory resource
// ID. If one is found, it is returned and ok will be true.
func GetKnownFolder(id DirectoryResourceID) (folder KnownFolder, ok bool) {
	folder, ok = knownFolders[id]
	return
}

var knownFolders = KnownFolderMap{
	"common-start-menu": KnownFolder{guid: windows.FOLDERID_CommonStartMenu, id: "common-start-menu"},
	"public-desktop":    KnownFolder{guid: windows.FOLDERID_PublicDesktop, id: "public-desktop"},
	"program-data":      KnownFolder{guid: windows.FOLDERID_ProgramData, id: "program-data"},
	"program-files":     KnownFolder{guid: windows.FOLDERID_ProgramFiles, id: "program-files"},
	"program-files-x86": KnownFolder{guid: windows.FOLDERID_ProgramFilesX86, id: "program-files-x86"},
	"program-files-x64": KnownFolder{guid: windows.FOLDERID_ProgramFilesX64, id: "program-files-x64"},
	"system":            KnownFolder{guid: windows.FOLDERID_System, id: "system", protected: true},
}
