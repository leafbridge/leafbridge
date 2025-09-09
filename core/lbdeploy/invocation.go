package lbdeploy

import "time"

// InvocationID uniquely identifies the execution of a deployment.
type InvocationID string

// Invocation holds information about a specific invocation of a deployment.
type Invocation struct {
	ID          InvocationID `json:"id"`
	Host        string       `json:"host,omitempty"`
	InitialFlow FlowID       `json:"initial-flow,omitempty"`
	Started     time.Time    `json:"started,omitzero"`

	// TODO: Consider including a content hash of the deployment.
}
