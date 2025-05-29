package lbdeployevent

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gentlemanautomaton/structformat"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
	"github.com/leafbridge/leafbridge/core/lbevent"
)

// Deployment action event types.
const (
	ActionStartedType = lbevent.Type("deployment.action:started")
	ActionStoppedType = lbevent.Type("deployment.action:stopped")
)

// ActionStarted is an event that occurs when a deployment action has started.
type ActionStarted struct {
	Deployment  lbdeploy.DeploymentID
	Flow        lbdeploy.FlowID
	ActionIndex int
	ActionType  lbdeploy.ActionType
}

// Type returns the type of the event.
func (e ActionStarted) Type() lbevent.Type {
	return ActionStartedType
}

// Level returns the level of the event.
func (e ActionStarted) Level() slog.Level {
	return slog.LevelDebug
}

// Message returns a description of the event.
func (e ActionStarted) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))
	builder.WriteStandard("Starting action")

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e ActionStarted) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e ActionStarted) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
	}
}

// ActionStopped is an event that occurs when a deployment action has stopped.
type ActionStopped struct {
	Deployment  lbdeploy.DeploymentID
	Flow        lbdeploy.FlowID
	ActionIndex int
	ActionType  lbdeploy.ActionType
	Started     time.Time
	Stopped     time.Time
	Err         error
}

// Type returns the type of the event.
func (e ActionStopped) Type() lbevent.Type {
	return ActionStoppedType
}

// Level returns the level of the event.
func (e ActionStopped) Level() slog.Level {
	if e.Err != nil {
		return slog.LevelError
	}
	if e.Duration() < time.Second*5 {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// Message returns a description of the event.
func (e ActionStopped) Message() string {
	var builder structformat.Builder

	builder.WritePrimary(string(e.Deployment))
	builder.WritePrimary(string(e.Flow))
	builder.WritePrimary(strconv.Itoa(e.ActionIndex + 1))
	builder.WritePrimary(string(e.ActionType))
	if e.Err != nil {
		builder.WriteStandard(fmt.Sprintf("Stopped action due to an error: %s", e.Err))
	} else {
		builder.WriteStandard(fmt.Sprintf("Completed action"))
	}
	builder.WriteNote(e.Duration().Round(time.Millisecond * 10).String())

	return builder.String()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e ActionStopped) Details() string {
	return ""
}

// Attrs returns a set of structured log attributes for the event.
func (e ActionStopped) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		slog.String("deployment", string(e.Deployment)),
		slog.String("flow", string(e.Flow)),
		slog.Group("action", "index", e.ActionIndex, "type", e.ActionType),
		slog.Time("started", e.Started),
		slog.Time("stopped", e.Stopped),
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("error", e.Err.Error()))
	}
	return attrs
}

// Duration returns the duration of the action.
func (e ActionStopped) Duration() time.Duration {
	return e.Stopped.Sub(e.Started)
}
