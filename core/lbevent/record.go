package lbevent

import (
	"log/slog"
	"time"
)

// Record is a record of an event within LeafBridge. It is the interface
// implemented by all event records.
type Record interface {
	Time() time.Time
	ToLog() slog.Record
	Interface
}

// RecordOf holds information about an event within LeafBridge of type T.
type RecordOf[T Interface] struct {
	time  time.Time
	pc    uintptr
	Event T
}

// NewRecord returns a record for the given event and program counter. It uses
// the current time as the event's timestamp.
//
// The program counter is used to build source line information for slog
// records.
func NewRecord[T Interface](at time.Time, pc uintptr, event T) RecordOf[T] {
	return RecordOf[T]{
		time:  time.Now(),
		pc:    pc,
		Event: event,
	}
}

// Time returns the time of the event.
func (r RecordOf[T]) Time() time.Time {
	return r.time
}

// Component identifies the component that generated the event.
func (r RecordOf[T]) Component() string {
	return r.Event.Component()
}

// Level returns the level of the event.
func (r RecordOf[T]) Level() slog.Level {
	return r.Event.Level()
}

// Message returns a description of the event.
func (r RecordOf[T]) Message() string {
	return r.Event.Message()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (r RecordOf[T]) Details() string {
	return r.Event.Details()
}

// Attrs returns a set of structured logging attributes for the event.
func (r RecordOf[T]) Attrs() []slog.Attr {
	return r.Event.Attrs()
}

// ToLog returns the event record as a structured logging record.
func (r RecordOf[T]) ToLog() slog.Record {
	out := slog.NewRecord(r.time, r.Event.Level(), r.Event.Message(), r.pc)
	out.AddAttrs(r.Event.Attrs()...)
	return out
}
