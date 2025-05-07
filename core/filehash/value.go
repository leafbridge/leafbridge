package filehash

import (
	"bytes"
	"encoding/hex"
)

// Entry stores a file hash value along with its type.
type Entry struct {
	Type  Type
	Value Value
}

// CompareEntries returns an integer comparing two file hash entries.
// It returns -1 if a is higher priority that b, 1 if b is higher priority
// than a, and 0 if the two entries are identical.
func CompareEntries(a, b Entry) int {
	// Perform a comparison of the hash types, which allows us to exert a
	// preference on known hashes.
	if result := CompareTypes(a.Type, b.Type); result != 0 {
		return result
	}

	// Compare the values as a last resort.
	return bytes.Compare(a.Value, b.Value)
}

// Value stores the bytes of a file hash.
type Value []byte

// String returns a string representation of v in hexadecimal format.
func (v Value) String() string {
	return hex.EncodeToString(v)
}

// MarshalText encodes v as a hexadecimal string.
func (v Value) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText unmarshals the given hexadecimal value into v.
func (v *Value) UnmarshalText(text []byte) error {
	b, err := hex.DecodeString(string(text))
	if err != nil {
		return err
	}
	*v = b
	return nil
}
