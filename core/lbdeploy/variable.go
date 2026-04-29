package lbdeploy

import (
	"fmt"

	"github.com/gentlemanautomaton/structformat"
	"github.com/leafbridge/leafbridge/core/lbvalue"
)

// VariableMap holds a set of veriables mapped by their identifiers.
type VariableMap map[VariableID]Variable

// VariableCache holds a cache of computed variables.
type VariableCache map[VariableID]lbvalue.Value

// VariableID is a unique identifier for a variable.
type VariableID string

// VariableSource identifies the source of a variable.
type VariableSource string

// Supported variable sources.
const (
	VariableSourceSubvariable           VariableSource = "variable"
	VariableSourceRegistryKeyValueNames VariableSource = "resource.registry.key:value-names"
	VariableSourceRegistryValue         VariableSource = "resource.registry.value"
	VariableSourceFileVersion           VariableSource = "resource.file-system.file:file-version"
	VariableSourceProductVersion        VariableSource = "resource.file-system.file:product-version"
)

// VariableType specifies the type of a variable when it is not clear from
// its source or when its type needs to be coerced.
type VariableType string

// Kind returns the kind of value indicated by the variable type.
func (t VariableType) Kind() lbvalue.Kind {
	if t == "" {
		return lbvalue.KindUnknown
	}
	var k lbvalue.Kind
	if err := k.UnmarshalText([]byte(t)); err != nil {
		return lbvalue.KindUnknown
	}
	return k
}

// Variable describes a variable that can be evaluated.
type Variable struct {
	Label    string           `json:"label,omitempty"`
	Source   VariableSource   `json:"source,omitempty"`
	Subject  string           `json:"subject,omitempty"`
	Type     VariableType     `json:"type,omitempty"`
	Elements []Variable       `json:"elements,omitzero"`
	Criteria lbvalue.Criteria `json:"criteria,omitzero"`
}

// VariableErrorOrigin identifies the origin of a variable error.
type VariableErrorOrigin int

// Potential origins of a variable error.
const (
	VariableErrorOriginSelf VariableErrorOrigin = iota
	VariableErrorOriginElement
)

// VariableError is returned when a variable cannot be calculated due to an
// error.
type VariableError struct {
	ID      VariableID
	Label   string
	Origin  VariableErrorOrigin
	Source  VariableSource
	Element int
	Err     error
}

// Unwrap returns the underlying error for the variable.
func (e VariableError) Unwrap() error {
	return e.Err
}

// Error returns the error as a string.
func (e VariableError) Error() string {
	var builder structformat.Builder
	switch {
	case e.ID != "" && e.Label != "":
		builder.WritePrimary(fmt.Sprintf("%s (%s)", e.ID, e.Label))
	case e.ID != "":
		builder.WritePrimary(string(e.ID))
	case e.Label != "":
		builder.WritePrimary(string(e.Label))
	}

	switch e.Origin {
	case VariableErrorOriginElement:
		builder.WritePrimary(fmt.Sprintf("Element [%d]", e.Element))
	default:
		if e.Source != "" {
			builder.WritePrimary(string(e.Source))
		}
	}

	builder.WriteStandard(e.Err.Error())

	return builder.String()
}

func variableSelfError(v Variable, err error) error {
	return VariableError{
		Label:  v.Label,
		Source: v.Source,
		Origin: VariableErrorOriginSelf,
		Err:    err,
	}
}
