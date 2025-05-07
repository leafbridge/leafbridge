package lbevent

import "log/slog"

// Component identifies the component within LeafBridge that generated the
// event.
//type Component string

// Interface is a common interface implemented by all LeafBridge events.
type Interface interface {
	// Component identifies the component that generated the event.
	Component() string

	// Level returns the level of the event.
	Level() slog.Level

	// Message returns a description of the event.
	Message() string

	// Details returns additional details about the event. It might include
	// multiple lines of text. An empty string is returned when no details
	// are available.
	Details() string

	// Attrs returns a set of structured logging attributes for the event.
	Attrs() []slog.Attr
}
