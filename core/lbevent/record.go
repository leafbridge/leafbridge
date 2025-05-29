package lbevent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// RecordUnmarshaler is a function capable of unmarshaling JSON data as a
// record.
type RecordUnmarshaler func([]byte) (Record, error)

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

// UnmarshalRecord interprets the given JSON data as a record of type T
// and returns its unmarshaled data as a [Record].
//
// If the data is for an event record of a different type, or if the
// unmarshaling fails, an error is returned.
func UnmarshalRecord[T Interface](b []byte) (Record, error) {
	var record RecordOf[T]
	err := record.UnmarshalJSON(b)
	return record, err
}

// Time returns the time of the event.
func (r RecordOf[T]) Time() time.Time {
	return r.time
}

// Type returns the type of the event.
func (r RecordOf[T]) Type() Type {
	return r.Event.Type()
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

// MarshalJSON marshals the record as JSON data.
//
// TODO: Consider encoding data that can be gleaned from the program counter.
func (r RecordOf[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(recordOf[T]{
		Time: r.time,
		Type: r.Type(),
		Data: r.Event,
	})
}

// UnmarshalJSON attempts to unmarshal the given JSON data into r.
//
// If the data is for an event record of a different type, or if the
// unmarshaling fails, an error is returned.
func (r *RecordOf[T]) UnmarshalJSON(b []byte) error {
	var aux recordOf[T]
	if err := json.Unmarshal(b, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal event record of type \"%s\": %w", r.Type(), err)
	}
	if aux.Type != r.Type() {
		return fmt.Errorf("attempted to unmarshal event record of type \"%s\" into a structure for type \"%s\"", aux.Type, r.Type())
	}
	*r = RecordOf[T]{
		time:  aux.Time,
		Event: aux.Data,
	}
	return nil
}

type recordOf[T Interface] struct {
	Time time.Time `json:"time"`
	Type Type      `json:"type"`
	Data T         `json:"data"`
}
