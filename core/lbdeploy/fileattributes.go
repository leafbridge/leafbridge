package lbdeploy

import (
	"errors"
	"fmt"
	"slices"

	"github.com/leafbridge/leafbridge/core/filehash"
)

// FileAttributes store file size and cryptographic hash data for a file.
type FileAttributes struct {
	Size   int64        `json:"size"`
	Hashes filehash.Map `json:"hashes"`
}

// Features returns a list of features that are present within the attributes.
func (attr FileAttributes) Features() (features []string) {
	if attr.Size > 0 {
		features = append(features, "file size")
	}
	for _, entry := range attr.Hashes.ToList() {
		features = append(features, string(entry.Type))
	}
	return features
}

// Validate returns a non-nil error if the file attributes are missing or
// invalid.
func (attr FileAttributes) Validate() error {
	if attr.Size < 0 {
		return errors.New("a negative file size was provided")
	}

	for _, entry := range attr.Hashes.ToList() {
		if entry.Type.Priority() == 0 {
			return fmt.Errorf("the file hash type \"%s\" is not recognized", entry.Type)
		}
		if len(entry.Value) == 0 {
			return fmt.Errorf("the file hash value for \"%s\" is missing", entry.Type)
		}
	}

	return nil
}

// EqualFileAttributes returns true if a and b have identical sizes and
// identical sets of file hashes.
func EqualFileAttributes(a, b FileAttributes) bool {
	// Compare file size.
	if a.Size != b.Size {
		return false
	}

	// Compare hashes.
	a1, a2 := a.Hashes.ToList(), b.Hashes.ToList()
	if slices.CompareFunc(a1, a2, filehash.CompareEntries) != 0 {
		return false
	}

	return true
}
