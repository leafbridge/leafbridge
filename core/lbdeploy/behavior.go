package lbdeploy

// OnErrorBehavior identifies a response to take when an error is encountered.
type OnErrorBehavior string

// Behavior options when an error is encountered.
const (
	OnErrorUnspecified OnErrorBehavior = ""
	OnErrorStop        OnErrorBehavior = "stop"
	OnErrorContinue    OnErrorBehavior = "continue"
)

// Behavior describes behavior modifications for a deployment or flow.
type Behavior struct {
	OnError OnErrorBehavior `json:"on-error,omitempty"`
}

// OverlayBehavior overlays the given set of behaviors, giving priority
// to later members.
func OverlayBehavior(behaviors ...Behavior) Behavior {
	var out Behavior
	for _, next := range behaviors {
		if next.OnError != OnErrorUnspecified {
			out.OnError = next.OnError
		}
	}
	return out
}
