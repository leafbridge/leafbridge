package lbvalue

import (
	"fmt"
)

// Comparison identifies an operator to use when comparing variables.
type Comparison int

// Comparison operators.
const (
	CompareEquals Comparison = iota
	CompareLessThan
	CompareLessThanOrEquals
	CompareGreaterThan
	CompareGreaterThanOrEquals
)

var comparisonStrings = []string{
	"=",
	"<",
	"<=",
	">",
	">=",
}

// Evaluate applies the comparison operator against the given result of a
// Compare function, which should be -1, 0, or +1. It returns true if the
// applying the comparison operator to the two values would be true.
func (c Comparison) Evaluate(result int) bool {
	switch c {
	case CompareEquals:
		return result == 0
	case CompareLessThan:
		return result < 0
	case CompareLessThanOrEquals:
		return result <= 0
	case CompareGreaterThan:
		return result > 0
	case CompareGreaterThanOrEquals:
		return result >= 0
	default:
		return false
	}
}

// String returns a string representation of c.
func (c Comparison) String() string {
	if c := int(c); c >= 0 && c < len(comparisonStrings) {
		return comparisonStrings[c]
	}
	return fmt.Sprintf("<unknown comparison operator type \"%d\">", c)
}

// UnmarshalText attempts to unmarshal the given text into c.
func (c *Comparison) UnmarshalText(b []byte) error {
	switch string(b) {
	case "=":
		*c = CompareEquals
	case "<":
		*c = CompareLessThan
	case "<=":
		*c = CompareLessThanOrEquals
	case ">":
		*c = CompareGreaterThan
	case ">=":
		*c = CompareGreaterThanOrEquals
	default:
		return fmt.Errorf("unrecognized comparison operator: %s", b)
	}
	return nil
}

// MarshalText marshals the comparison operator as text.
func (c Comparison) MarshalText() ([]byte, error) {
	if c := int(c); c >= 0 && c < len(comparisonStrings) {
		return []byte(comparisonStrings[c]), nil
	}
	return nil, fmt.Errorf("unrecognized comparison operator: %d", c)
}

// ComparisonError is returned when a comparison is attempted on incomparable
// values.
type ComparisonError struct {
	A, B Kind
}

// Error returns a string describing the error.
func (e ComparisonError) Error() string {
	return fmt.Sprintf("the \"%s\" and \"%s\" types are not comparable", e.A, e.B)
}
