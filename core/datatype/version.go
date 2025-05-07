package datatype

import (
	"iter"
	"strconv"
	"strings"
)

// Version encodes a version number or identifier in a string. It should be in
// dotted form, like "1.2.3" or "2.5.A".
//
// It permits a leading "v" charater, as in "v1.2.3". The leading character
// will be ignored when performing comparisons.
//
// Versions are made up of segments, which are the portions of the version
// between each dot.
//
// TODO: Consider treating spaces the same as dots.
type Version string

// Segments returns an iterator for the version segments contained in v.
func (v Version) Segments() iter.Seq[VersionSegment] {
	// If there's a leading "v" at the start of the string, ignore it.
	if len(v) > 1 && (v[0] == 'v' || v[0] == 'V') {
		v = v[1:]
	}

	return func(yield func(VersionSegment) bool) {
		for {
			cut := strings.IndexByte(string(v), '.')
			if cut < 0 {
				break
			}
			segment := VersionSegment(v[:cut])
			if !yield(segment) {
				return
			}
			if cut+1 >= len(v) {
				return
			}
			v = v[cut+1:]
		}
		if v != "" {
			yield(VersionSegment(v))
		}
	}
}

// Canonical returns the version in a canonical format that omits excess
// zeroes and any leading version designator such as "v".
func (v Version) Canonical() string {
	var out strings.Builder
	for segment := range v.Segments() {
		if out.Len() > 0 {
			out.WriteString(".")
		}
		out.WriteString(string(segment))
	}
	return out.String()
}

// VersionSegment is a segement within a version. Segments are separated by
// dots.
type VersionSegment string

// CompareVersions returns an integer comparing two versions.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
func CompareVersions(a, b Version) int {
	next1, stop1 := iter.Pull(a.Segments())
	defer stop1()
	next2, stop2 := iter.Pull(b.Segments())
	defer stop2()
	for {
		segment1, ok1 := next1()
		segment2, ok2 := next2()
		switch {
		case !ok1 && !ok2:
			return 0
		case !ok1:
			return -1
		case !ok2:
			return 1
		}

		if s := CompareVersionSegments(segment1, segment2); s != 0 {
			return s
		}
	}
}

// CompareVersionSegments returns an integer comparing two version segments.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
//
// If both segments can be interpreted as 64-bit unsigned integers, they will
// be compared as unsigned integers.
//
// If integer comparison is not possible, the length of the segments will be
// compared. If the segments are different lengths, the shorter segment will
// be considered "less" than the longer one.
//
// If integer comarison is not possible and the segments are the same legnth,
// they will be compared lexicographically.
func CompareVersionSegments(a, b VersionSegment) int {
	i1, err1 := strconv.ParseUint(string(a), 10, 64)
	i2, err2 := strconv.ParseUint(string(b), 10, 64)

	if err1 == nil && err2 == nil {
		switch {
		case i1 < i2:
			return -1
		case i1 > i2:
			return 1
		default:
			return 0
		}
	}

	switch len1, len2 := len(a), len(b); {
	case len1 < len2:
		return -1
	case len1 > len2:
		return 1
	default:
		return strings.Compare(string(a), string(b))
	}
}
