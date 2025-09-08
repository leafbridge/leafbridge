package lbvalue

import "github.com/leafbridge/leafbridge/core/datatype"

// Criteria holds criteria that can be used to match or filter a value.
//
// TODO: Consider adding a Violation property that yields an error if the
// criteria doesn't match.
type Criteria struct {
	Label      string     `json:"label,omitempty"`
	Comparison Comparison `json:"comparison,omitempty"`
	Value      Value      `json:"value,omitzero"`
	Negated    bool       `json:"negated,omitzero"`
	Any        []Criteria `json:"any,omitzero"`
	All        []Criteria `json:"all,omitzero"`
}

// IsZero returns true if the criteria is unspecified.
func (criteria Criteria) IsZero() bool {
	if len(criteria.Any) > 0 {
		return false
	}
	if len(criteria.All) > 0 {
		return false
	}
	if criteria.Value != Unknown {
		return false
	}
	return true
}

// Evaluate returns true if the criteria matches the given value.
func (criteria Criteria) Evaluate(value Value) (bool, error) {
	// Evaluate "any" conditions.
	if len(criteria.Any) > 0 {
		for i := range criteria.Any {
			result, err := criteria.Any[i].Evaluate(value)
			if err != nil {
				return result, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Evaluate "all" conditions.
	if len(criteria.All) > 0 {
		for i := range criteria.All {
			result, err := criteria.All[i].Evaluate(value)
			if err != nil {
				return result, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// Evaluate individual conditions.
	difference, err := TryCompare(value, criteria.Value)
	if err != nil {
		return false, err
	}

	// Apply the comparison operator.
	result := criteria.Comparison.Evaluate(difference)

	// Negate the result if requested.
	if criteria.Negated {
		result = !result
	}

	return result, nil
}

// Filter returns the given value with the criteria applied to it as a filter.
//
// If a value does not meet the criteria, it is omitted or replaced with a
// zero value.
func (criteria Criteria) Filter(value Value) (Value, error) {
	switch value.Kind() {
	case KindVersionSet:
		original := value.VersionSet()
		if original == nil {
			return value, nil
		}
		filtered := make(datatype.VersionSet, len(original))
		for version := range original {
			passed, err := criteria.Evaluate(Version(version))
			if err != nil {
				return value, err
			}
			if passed {
				filtered.Add(version)
			}
		}
		return VersionSet(filtered), nil
	default:
		passed, err := criteria.Evaluate(value)
		if err != nil {
			return value, err
		}
		if !passed {
			return Zero(value.Kind()), nil
		}
		return value, nil
	}
}
