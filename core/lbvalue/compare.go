package lbvalue

import (
	"strings"

	"github.com/leafbridge/leafbridge/core/datatype"
)

// TryCompare returns an integer comparing values a and b.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
//
// If the values cannot be compared, it returns an error.
func TryCompare(a, b Value) (int, error) {
	n := Compare(a, b)
	if n < -1 {
		return n, ComparisonError{A: a.Kind(), B: b.Kind()}
	}
	return n, nil
}

// Compare returns an integer comparing values a and b.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
//
// If the values cannot be compared, it returns -2.
func Compare(a, b Value) int {
	switch data1 := a.data.(type) {
	case Kind:
		switch data1 {
		case KindBool:
			if b.Kind() == KindBool {
				b1, b2 := a.Bool(), b.Bool()
				switch {
				case b1 == b2:
					return 0
				case b2:
					return -1
				default:
					return 1
				}
			}
		case KindInt64:
			if b.Kind() == KindInt64 {
				i1, i2 := a.Int64(), b.Int64()
				switch {
				case i1 == i2:
					return 0
				case i1 < i2:
					return -1
				default:
					return 1
				}
			}
		}
	case datatype.Version:
		if data2, ok := b.data.(datatype.Version); ok {
			return datatype.CompareVersions(data1, data2)
		}
	case string:
		if data2, ok := b.data.(string); ok {
			return strings.Compare(data1, data2)
		}
	}

	return -2
}
