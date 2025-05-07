package filehash

// List is an ordered list of file hash entries.
type List []Entry

// Primary returns the first entry from the list, which is considered the
// primary and canonical hash value.
//
// If the list is empty, it returns a zeroed hash entry.
func (list List) Primary() Entry {
	if len(list) == 0 {
		return Entry{}
	}
	return list[0]
}
