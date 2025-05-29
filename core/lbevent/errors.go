package lbevent

import (
	"errors"
	"fmt"
	"log/slog"
)

// HandlerError is an error returned by a recorder when an event handler
// is unable to process an event.
type HandlerError struct {
	HandlerName string
	Record      Record
	Err         error
}

// Component identifies the component that generated the event.
func (e HandlerError) Component() string {
	return "event-handler"
}

// Level returns the level of the event.
func (e HandlerError) Level() slog.Level {
	return slog.LevelError
}

// Message returns a description of the event.
func (e HandlerError) Message() string {
	return e.Error()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e HandlerError) Details() string {
	return ""
}

// Attrs returns a set of structured logging attributes for the event.
func (e HandlerError) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("handler", string(e.HandlerName)),
		slog.String("error", e.Error()),
	}
}

// Error returns a string describing the error.
func (e HandlerError) Error() string {
	return fmt.Sprintf("the \"%s\" event handler failed to record a \"%s\" event: %s", e.HandlerName, e.Record.Type(), e.Err)
}

// Unwrap returns the error wrapped by e.
func (e HandlerError) Unwrap() error {
	return e.Err
}

// MultiHandlerError is an error returned by a recorder when one or more
// members of a multiple event handler are unable to process an event.
type MultiHandlerError struct {
	HandlerName string
	Record      Record
	Errors      []error
}

// Component identifies the component that generated the event.
func (e MultiHandlerError) Component() string {
	return "event-handler"
}

// Level returns the level of the event.
func (e MultiHandlerError) Level() slog.Level {
	return slog.LevelError
}

// Message returns a description of the event.
func (e MultiHandlerError) Message() string {
	return e.Error()
}

// Details returns additional details about the event. It might include
// multiple lines of text. An empty string is returned when no details
// are available.
func (e MultiHandlerError) Details() string {
	return ""
}

// Attrs returns a set of structured logging attributes for the event.
func (e MultiHandlerError) Attrs() []slog.Attr {
	return []slog.Attr{
		slog.String("handler", string(e.HandlerName)),
		slog.String("error", e.Error()),
	}
}

// Error returns a string describing the error.
func (e MultiHandlerError) Error() string {
	var affected string
	if n := len(e.Errors); n == 1 {
		affected = "1 member"
	} else {
		affected = fmt.Sprintf("%d members", n)
	}
	return fmt.Sprintf("%s of the \"%s\" event handler failed to record a \"%s\" event: %s", affected, e.HandlerName, e.Record.Type(), errors.Join(e.Errors...))
}

// Unwrap returns the errors wrapped by e.
func (e MultiHandlerError) Unwrap() []error {
	return e.Errors
}

// WrapHandlerError returns an error for the given handler, record, and
// underlying errors.
func WrapHandlerError(handler Handler, r Record, errs ...error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		candidate := errs[0]
		if candidate == nil {
			return nil
		}
		_, ok1 := candidate.(HandlerError)
		_, ok2 := candidate.(MultiHandlerError)
		if ok1 || ok2 {
			return candidate
		}
		return HandlerError{
			HandlerName: handler.Name(),
			Record:      r,
			Err:         candidate,
		}
	default:
		return MultiHandlerError{
			HandlerName: handler.Name(),
			Record:      r,
			Errors:      errs,
		}
	}
}
