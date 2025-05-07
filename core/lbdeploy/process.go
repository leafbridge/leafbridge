package lbdeploy

// ProcessResourceMap holds a set of process resources mapped by their
// identifiers.
type ProcessResourceMap map[ProcessResourceID]ProcessResource

// ProcessResourceID is a unique identifier for a process resource.
type ProcessResourceID string

// ProcessResource describes a process resource.
type ProcessResource struct {
	// Description of the process that is identified.
	Description string `json:"description,omitempty"`

	// Match describes criteria for identification of a running process.
	Match ProcessMatch `json:"match,omitzero"`
}

// ProcessAttributeID identifies an attribute of a process.
type ProcessAttributeID string

// Process Attributes.
const (
	ProcessName ProcessAttributeID = "name"
	//ProcessPath ProcessAttributeID = "path"
)

// MatchType defines the type of match to use for a field.
type MatchType string

// Match Types.
const (
	MatchEquals   MatchType = "equals"
	MatchContains MatchType = "contains"
	//MatchStartWith         MatchType = "starts-with"
	//MatchEndsWith          MatchType = "ends-with"
	//MatchRegularExpression MatchType = "expression"
)

// ProcessMatch holds information used to identify processes running on a
// local machine.
type ProcessMatch struct {
	Label     string             `json:"label,omitempty"`
	Attribute ProcessAttributeID `json:"attribute,omitempty"`
	Type      MatchType          `json:"type,omitempty"`
	Value     string             `json:"value,omitempty"`
	Any       []ProcessMatch     `json:"any,omitzero"`
	All       []ProcessMatch     `json:"all,omitzero"`
}
