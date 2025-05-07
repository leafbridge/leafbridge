package filehash

import (
	"slices"
)

// Map is a map of file hash values.
type Map map[Type]Value

// Primary returns the primary hash entry from the map.
// The primary entry is the hash type that is most preferred.
func (m Map) Primary() Entry {
	return m.ToList().Primary() // This is safe for nil maps and lists.
}

// Types returns an ordered set of hash types present in the map.
func (m Map) Types() []Type {
	types := make([]Type, 0, len(m))
	for typ := range m {
		types = append(types, typ)
	}
	slices.SortFunc(types, CompareTypes)
	return types
}

// ToList returns an ordered list of file hashes. File hashes with recognized
// types will be placed first in the list, in order of preference.
//
// Unrecognized file hash types will be sorted lexicographically.
//
// If the map is nil or empty, it returns a nil list.
func (m Map) ToList() List {
	// If the map is empty, return nil.
	if len(m) == 0 {
		return nil
	}

	// Convert the map to a list.
	list := make(List, 0, len(m))
	for typ, value := range m {
		list = append(list, Entry{Type: typ, Value: value})
	}

	// Sort the list.
	slices.SortFunc(list, CompareEntries)

	return list
}
