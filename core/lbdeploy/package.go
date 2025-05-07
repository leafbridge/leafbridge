package lbdeploy

import (
	"errors"
	"fmt"

	"github.com/leafbridge/leafbridge/core/filehash"
)

// PackageMap holds a set of packages mapped by their identifiers.
type PackageMap map[PackageID]Package

// PackageID is a unique identifier for a deployment package.
type PackageID string

// Validate returns a non-nil error if the package ID is invalid.
func (id PackageID) Validate() error {
	if id == "" {
		return errors.New("a package ID is missing")
	}
	return nil
}

// PackageContent is a content-addressable identifier for a package.
type PackageContent struct {
	ID          PackageID
	PrimaryHash filehash.Entry
}

// String returns a string representation of the package content in the form
// pkg-[id]-[hash].
func (content PackageContent) String() string {
	out := "pkg"
	if content.ID != "" {
		out += "-" + string(content.ID)
	}
	if value := content.PrimaryHash.Value.String(); value != "" {
		if len(value) > 16 {
			value = value[:16]
		}
		out += "-" + value
	}
	return out
}

// PackageType declares the type of a package.
type PackageType string

// IsArchive returns true if the package type is an archive that must be
// extracted before use.
func (t PackageType) IsArchive() bool {
	switch t {
	case "archive":
		return true
	default:
		return false
	}
}

// PackageFormat declares the format of a package.
type PackageFormat string

// Package defines a deployment package.
//
// TODO: Add support for a destination directory where an archive's extracted
// files will be extracted to. If a destination is not provided, then fall
// back to the current approach that extracts files to a temporary directory.
type Package struct {
	Name       string          `json:"name,omitempty"`
	Type       PackageType     `json:"type,omitempty"`
	Format     PackageFormat   `json:"format,omitempty"`
	Sources    []PackageSource `json:"sources,omitempty"`
	Attributes FileAttributes  `json:"attributes,omitzero"`
	Files      PackageFileMap  `json:"files,omitzero"`
	Commands   CommandMap      `json:"commands,omitzero"`
	//Destinations []DirectoryResourceID `json:"destinations,omitempty"`
}

// FileName returns a file name for the package to be downloaded.
func (pkg Package) FileName() string {
	return pkg.Name + "." + pkg.FileExtension()
}

// FileExtension returns an appropriate file extension for the package.
//
// If the package type is not recognized, it returns "file".
func (pkg Package) FileExtension() string {
	switch pkg.Type {
	case "exe":
		return "exe"
	case "msi":
		return "msi"
	case "archive":
		switch pkg.Format {
		case "zip":
			return "zip"
		}
	}
	return "file"
}

// Validate returns a non-nil error if the package contains invalid
// configuration.
func (pkg Package) Validate() error {
	// Validate package type and format.
	switch pkg.Type {
	case "exe":
	case "msi":
	case "archive":
		switch pkg.Format {
		case "zip":
		default:
			return fmt.Errorf("the package format \"%s\" is not a recognized format for %s packages", pkg.Format, pkg.Type)
		}
	default:
		return fmt.Errorf("the package type \"%s\" is not recognized", pkg.Type)
	}

	// Validate package sources.
	for i, source := range pkg.Sources {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("package source %d: %w", i, err)
		}
	}

	// Validate package file attributes.
	if err := pkg.Attributes.Validate(); err != nil {
		return fmt.Errorf("package file attributes: %w", err)
	}

	// Validate package commands.
	for id, command := range pkg.Commands {
		if command.Executable != "" {
			if pkg.Type != "archive" {
				return fmt.Errorf("package command \"%s\": an executable file ID is only valid for archive packages", id)
			}
			if _, ok := pkg.Files[PackageFileID(command.Executable)]; !ok {
				return fmt.Errorf("package command \"%s\": the executable file ID refers to package file \"%s\", which is not defined in the package file set", id, command.Executable)
			}
		}
	}
	return nil
}

// Package source types.
const (
	PackageSourceHTTP PackageSourceType = "http"
)

// PackageSourceType declares the type of source for a package.
type PackageSourceType string

// PackageSource defines a potential source for retrieval of a package.
type PackageSource struct {
	Type PackageSourceType
	URL  string
}

// Validate returns a non-nil error if the package source is invalid.
func (source PackageSource) Validate() error {
	switch source.Type {
	case "":
		return errors.New("the source type is missing")
	case PackageSourceHTTP:
	default:
		return fmt.Errorf("the package source type \"%s\" is not recognized", source.Type)
	}

	return nil
}

// PackageFileMap holds a set of package files mapped by their identifiers.
//
// It is used by archive packages to verify the presence of important files
// within the archive.
type PackageFileMap map[PackageFileID]PackageFile

// PackageFileID is a unique identifier for a file within a package.
type PackageFileID string

// PackageFile describes a set of files that are expected to be present
// within an archive package.
type PackageFile struct {
	Path       string         `json:"path"`
	Attributes FileAttributes `json:"attributes,omitzero"`
}
