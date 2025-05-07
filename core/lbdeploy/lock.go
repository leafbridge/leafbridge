package lbdeploy

// LockMap holds a set of lockable resources mapped by their identifiers.
type LockMap map[LockID]Lock

// LockID is a unique identifier for a lockable resource.
type LockID string

// Lock is a lockable resource that can be used to prevent invocations
// from competing or interfering with each other.
type Lock struct {
	Description   string            `json:"description,omitempty"`
	Mutex         MutexID           `json:"mutex,omitempty"`
	ConflictRules LockConflictRules `json:"conflict,omitzero"`

	// TODO: Consider adding these fields:
	// Type ("mutex")
	// Max (max number of concurrent lock holders)?

	// TODO: Consider adding file-based locks that refer to a FileID:
	// File FileID
}

// LockConflictRules provide guidance for what to do when a conflict is
// encountered on a lockable resource.
type LockConflictRules struct {
	Message string `json:"message,omitempty"`
}
