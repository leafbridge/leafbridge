package idset

import (
	"cmp"
	"slices"
)

// SetOf is a set of IDs of type T.
type SetOf[T cmp.Ordered] map[T]struct{}

// Contains returns true if the given id is present in the set.
func (set SetOf[T]) Contains(id T) bool {
	if len(set) == 0 {
		return false
	}
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

// List returns the members of the set in an ordered list with ascending
// order.
func (set SetOf[T]) List() []T {
	if len(set) == 0 {
		return nil
	}

	items := make([]T, 0, len(set))
	for item := range set {
		items = append(items, item)
	}

	slices.Sort(items)

	return items
}
