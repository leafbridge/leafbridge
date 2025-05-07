package lbdeploy

// FlowMap holds a set of deployment flows mapped by their identifiers.
type FlowMap map[FlowID]Flow

// FlowID is a unique identifier for a flow within a deployment.
type FlowID string

// Flow is a flow of actions within a deployment.
//
// TODO: Consider renaming "Preconditions" to "Requirements".
type Flow struct {
	Constraints   ConditionList `json:"constraints,omitzero"`
	Preconditions ConditionList `json:"preconditions,omitzero"`
	Locks         []LockID      `json:"locks,omitzero"`
	Behavior      Behavior      `json:"behavior,omitzero"`
	Actions       []Action      `json:"actions,omitzero"`
}

// FlowStats hold statistics about a flow that has been invoked.
type FlowStats struct {
	ActionsCompleted int
	ActionsFailed    int
}
