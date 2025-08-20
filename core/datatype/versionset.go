package datatype

import "slices"

// VersionSet is a set of versions.
//
// The set always stores versions in canonical form. It prevents duplicates
// from being added that would be considered equivalent by [CompareVersions].
type VersionSet map[Version]struct{}

// Contains returns true if the given version is present in the set.
func (set VersionSet) Contains(version Version) bool {
	if len(set) == 0 {
		return false
	}

	version = version.Canonical()

	// Fast path for exact matches.
	_, present := set[version]
	if present {
		return true
	}

	// Slow path for inexact matches.
	for existing := range set {
		if CompareVersions(version, existing) == 0 {
			return true
		}
	}

	return false
}

// Max returns the highest version present in the set.
//
// If the set is empty, it returns an empty version.
func (set VersionSet) Max() Version {
	var max Version
	for version := range set {
		if max == "" || CompareVersions(version, max) > 0 {
			max = version
		}
	}
	return max
}

// Add adds the given version to the set. If it is already present, it takes
// no action.
func (set VersionSet) Add(version Version) {
	version = version.Canonical()

	if set.Contains(version) {
		return
	}

	set[version] = struct{}{}
}

// Remove removes the given version from the set. If it is not present, it
// takes no action.
func (set VersionSet) Remove(version Version) {
	version = version.Canonical()

	// Fast path for exact matches.
	_, present := set[version]
	if present {
		delete(set, version)
		return
	}

	// Slow path for inexact matches.
	for existing := range set {
		if CompareVersions(version, existing) == 0 {
			delete(set, existing)
			return
		}
	}
}

// List returns the members of the set in an ordered list from lowest
// to highest.
func (set VersionSet) List() (versions []Version) {
	if len(set) == 0 {
		return
	}

	versions = make([]Version, 0, len(set))
	for version := range set {
		versions = append(versions, version)
	}

	slices.SortFunc(versions, CompareVersions)

	return
}
