package filehash

import "strings"

// Recognized file hash types.
const (
	SHA3_256 Type = "sha3-256"
)

// Type identifies the type of cryptographic hash used for file verification.
type Type string

// Priority returns a priority for recognized hash types. The higher the
// value returned, the higher the priority.
//
// Unrecognized hash types have a priority of zero.
func (t Type) Priority() int {
	switch t {
	case SHA3_256:
		return 1
	}
	return 0
}

// CompareTypes returns an integer comparing two file hash types.
// It returns -1 if a is higher priority that b, 1 if b is higher priority
// than a, and 0 if the two entries are identical.
func CompareTypes(a, b Type) int {
	// Compare the priority for known types.
	// Higher priority types are placed first.
	p1, p2 := a.Priority(), b.Priority()
	if p1 > p2 {
		return -1
	} else if p1 < p2 {
		return 1
	}

	// Perform a lexicographic comparison of the types.
	if result := strings.Compare(string(a), string(b)); result != 0 {
		return result
	}

	return 0
}
