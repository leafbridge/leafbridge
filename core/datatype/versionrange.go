package datatype

import (
	"encoding/json"
	"fmt"
)

// VersionRange describes a range of versions.
//
// It handles special cases in the following ways:
//   - If Start is empty, it matches all versions before End.
//   - If End is empty, it matches all versions after Start.
//   - If both Start and End are empty, it matches all versions.
//
// The starting and ending versions can be inclusive or exclusive.
//
// When marshaled as JSON, it marshals exclusive Start fields as "after"
// and exclusive End fields as "before". This allows the JSON representation
// to be more inuitive.
type VersionRange struct {
	Start            Version
	End              Version
	StartIsExclusive bool
	EndIsExclusive   bool
}

// versionRangeEncoding is used to encode and decode a version range as JSON.
//
// The "Start" and "After" fields are mutually exclusive, as are the "Before"
// and "End" fields.
type versionRangeEncoding struct {
	Start  Version `json:"start,omitempty"`  // Inclusive start
	After  Version `json:"after,omitempty"`  // Exclusive start
	Before Version `json:"before,omitempty"` // Exclusive end
	End    Version `json:"end,omitempty"`    // Inclusive end
}

// Includes returns true if the version range includes v.
func (r VersionRange) Includes(v Version) bool {
	if r.Start != "" {
		switch CompareVersions(v, r.Start) {
		case 0:
			if r.StartIsExclusive {
				return false
			}
		case -1:
			return false
		}
	}
	if r.End != "" {
		switch CompareVersions(v, r.End) {
		case 0:
			if r.EndIsExclusive {
				return false
			}
		case 1:
			return false
		}
	}
	return true
}

// String returns a string representation of the version range in one of
// the following forms, with "(" and ")" denoting inclusive ranges and
// "[" and "]" denoting exclusive ranges:
//   - [Start, End]
//   - (Start, End]
//   - (Start, End)
//   - [Start, End)
//   - >Start
//   - >=Start
//   - <=End
//   - <End
//
// If the range is empty, it returns the empty string.
func (r VersionRange) String() string {
	switch {
	case r.Start != "" && r.End != "":
		switch {
		case r.StartIsExclusive && r.EndIsExclusive:
			return "[" + string(r.Start.Canonical()) + ", " + string(r.End.Canonical()) + "]"
		case !r.StartIsExclusive && !r.EndIsExclusive:
			return "(" + string(r.Start.Canonical()) + ", " + string(r.End.Canonical()) + ")"
		case !r.StartIsExclusive && r.EndIsExclusive:
			return "(" + string(r.Start.Canonical()) + ", " + string(r.End.Canonical()) + "]"
		default:
			return "[" + string(r.Start.Canonical()) + ", " + string(r.End.Canonical()) + ")"
		}
	case r.Start != "":
		if r.StartIsExclusive {
			return ">" + string(r.Start.Canonical())
		}
		return ">=" + string(r.Start.Canonical())
	case r.End != "":
		if r.EndIsExclusive {
			return "<" + string(r.End.Canonical())
		}
		return "<=" + string(r.End.Canonical())
	default:
		return ""
	}
}

// UnmarshalJSON attempts to unmarshal the given JSON data into r.
func (r *VersionRange) UnmarshalJSON(b []byte) error {
	var aux versionRangeEncoding
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.Start != "" && aux.After != "" {
		return fmt.Errorf("version range contains both \"start\" and \"after\" fields, which are mutually exclusive (\"%s\" and \"%s\")", aux.Start, aux.After)
	}

	if aux.Before != "" && aux.End != "" {
		return fmt.Errorf("version range contains both \"before\" and \"end\" fields, which are mutually exclusive (\"%s\" and \"%s\")", aux.Before, aux.End)
	}

	if aux.Start != "" {
		r.Start = aux.Start
	}
	if aux.After != "" {
		r.Start = aux.After
		r.StartIsExclusive = true
	}
	if aux.Before != "" {
		r.End = aux.Before
		r.EndIsExclusive = true
	}
	if aux.End != "" {
		r.End = aux.End
	}

	return nil
}

// MarshalJSON marshals the version range as JSON data.
func (v VersionRange) MarshalJSON() ([]byte, error) {
	var aux versionRangeEncoding
	if v.Start != "" {
		if v.StartIsExclusive {
			aux.After = v.Start
		} else {
			aux.Start = v.Start
		}
	}
	if v.End != "" {
		if v.EndIsExclusive {
			aux.Before = v.End
		} else {
			aux.End = v.End
		}
	}
	return json.Marshal(aux)
}
