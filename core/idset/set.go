package idset

// SetOf is a set of IDs of type T.
type SetOf[T comparable] map[T]struct{}

// Contains returns true if the given id is present in the set.
func (set SetOf[T]) Contains(id T) bool {
	_, present := set[id]
	return present
}

// Add adds the given id to the set. If it is already present, it takes
// no action.
func (set SetOf[T]) Add(id T) {
	set[id] = struct{}{}
}

// Remove removes the given id from the set. If it is not present, it takes
// no action.
func (set SetOf[T]) Remove(id T) {
	delete(set, id)
}
