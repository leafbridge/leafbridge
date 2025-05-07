package lbevent

import (
	"runtime"
	"time"
)

// Recorder is a LeafBridge event recorder. It collects information about
// events that happen within LeafBridge and passes them to an event handler.
//
// If the recorder's handler is nil, it silently discards all events.
type Recorder struct {
	Handler Handler
}

// Record records the given event and passes it to the recorder's handler.
func (rec Recorder) Record(event Interface) error {
	// If no handler has been provided, drop the event.
	if rec.Handler == nil {
		return nil
	}

	// Record the current time.
	at := time.Now()

	// Collect the current program counter of the caller. This allows
	// for source code information to be collected by the handler.
	var pc uintptr
	{
		var pcs [1]uintptr
		// Skip [runtime.Callers, this function]
		runtime.Callers(2, pcs[:])
		pc = pcs[0]
	}

	// Prepare an event record.
	record := NewRecord(at, pc, event)

	// Send the event record to the event handler.
	err := rec.Handler.Handle(record)

	// If an error was encountered, make sure it's wrapped properly.
	err = WrapHandlerError(rec.Handler, record, err)

	// If an error was encountered while recording the event, try to record
	// the error itself as an event.
	if err != nil {
		if event, ok := err.(Interface); ok {
			at := time.Now()
			rec.Handler.Handle(NewRecord(at, pc, event))
		}
	}

	return err
}
