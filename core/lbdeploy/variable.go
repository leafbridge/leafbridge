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
	VariableSourceRegistryValue  VariableSource = "resource.registry.value"
	VariableSourceFileVersion    VariableSource = "resource.file-system.file:file-version"
	VariableSourceProductVersion VariableSource = "resource.file-system.file:product-version"
)

// Variable describes a variable that can be evaluated.
type Variable struct {
	Label   string         `json:"label,omitempty"`
	Source  VariableSource `json:"source,omitempty"`
	Subject string         `json:"subject,omitempty"`
}

// VariableError is returned when a variable cannot be calculated due to an
// error.
type VariableError struct {
	ID     VariableID
	Label  string
	Source VariableSource
	Err    error
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

	if e.Source != "" {
		builder.WritePrimary(string(e.Source))
	}

	builder.WriteStandard(e.Err.Error())

	return builder.String()
}
