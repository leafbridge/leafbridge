package lbdeployevent

import (
	"log/slog"
	"time"

	"github.com/gentlemanautomaton/structformat"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// Deployment event types.
const (
	DeploymentStartedType = lbevent.Type("deployment:started")
	DeploymentStoppedType = lbevent.Type("deployment:stopped")
)

// DeploymentStarted is an event that occurs when a deployment has started.
type DeploymentStarted struct {
	Invocation lbdeploy.Invocation
	Deployment lbdeploy.Deployment
}

// Type returns the type of the event.
func (e DeploymentStarted) Type() lbevent.Type {
	return DeploymentStartedType
}

// Level returns the level of the event.
func (e DeploymentStarted) Level() slog.Level {
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e DeploymentStarted) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment.ID))
	builder.WriteStandard("Starting.")

	builder.WriteNote(string(e.Invocation.ID))

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e DeploymentStarted) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e DeploymentStarted) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("invocation", string(e.Invocation.ID)),
		slog.String("deployment", string(e.Deployment.ID)),
	}
}

// DeploymentStopped is an event that occurs when a deployment has stopped.
type DeploymentStopped struct {
	Invocation lbdeploy.InvocationID
	Deployment lbdeploy.DeploymentID
	Started    time.Time
	Stopped    time.Time
	Err        error
}

// Type returns the type of the event.
func (e DeploymentStopped) Type() lbevent.Type {
	return DeploymentStoppedType
}

// Level returns the level of the event.
func (e DeploymentStopped) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e DeploymentStopped) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WriteStandard("Stopped.")

	builder.WriteNote(string(e.Invocation))
	builder.WriteNote(e.Duration().Round(time.Millisecond * 10).String())

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e DeploymentStopped) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e DeploymentStopped) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("invocation", string(e.Invocation)),
		slog.String("deployment", string(e.Deployment)),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the deployment.
func (e DeploymentStopped) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}
