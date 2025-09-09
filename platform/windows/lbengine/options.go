package lbengine

import (
	"io"

	"github.com/leafbridge/leafbridge/core/lbevent"
)

// Options hold configuration options for a LeafBridge deployment engine.
type Options struct {
	// Events, if non-nil, is an event recorder to receive deployment events.
	Events lbevent.Recorder

	// Force, if true, requests that commands be run even if the app evalution
	// engine doesn't think they're necessary.
	Force bool

	// CommandOutputOptions providers optional writers that will receive the
	// output of any commands that are run.
	CommandOutput CommandOutput
}

// CommandOutput defines optional writers that can receive the output of
// commands that are run.
type CommandOutput struct {
	Stdout io.Writer
	Stderr io.Writer
}
