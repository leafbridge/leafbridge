package windowsevent

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/leafbridge/leafbridge/core/lbevent"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc/eventlog"
)

const lbEventSource = "LeafBridge"

// EventMapper is an interface that is capable of mapping event types to IDs.
type EventMapper interface {
	EventID(event lbevent.Type) (id lbevent.ID, ok bool)
}

// Handler is a LeafBridge event handler that sends events to the
// Windows event log.
type Handler struct {
	elog   *eventlog.Log
	mapper EventMapper
}

// NewWindowsHandler returns a WindowsHandler that sends events to the
// Windows event log. The provided event mapper is used to determine
// event IDs.
func NewHandler(mapper EventMapper) (Handler, error) {
	if mapper == nil {
		return Handler{}, errors.New("failed to prepare a new windowsevent.Handler: a nil event mapper was provided")
	}

	// Register the event source if it isn't already registered.
	alreadyRegisterd, err := IsWindowsEventSourceRegistered(lbEventSource)
	if err != nil {
		return Handler{}, err
	}

	if !alreadyRegisterd {
		const eventTypes = eventlog.Error | eventlog.Warning | eventlog.Info
		if err := eventlog.InstallAsEventCreate(lbEventSource, eventTypes); err != nil {
			// Report the error but press on regardless
			return Handler{}, fmt.Errorf("failed to register event log source for \"%s\": %w", lbEventSource, err)
		}
	}

	// Open the event source.
	elog, err := eventlog.Open(lbEventSource)
	if err != nil {
		return Handler{}, fmt.Errorf("failed to open event log source for \"%s\": %w", lbEventSource, err)
	}
	return Handler{
		elog:   elog,
		mapper: mapper,
	}, nil
}

// Name returns a name for the handler.
func (h Handler) Name() string {
	return "windows-application-log"
}

// Handle processes the given event record.
func (h Handler) Handle(r lbevent.Record) (err error) {
	id, ok := h.mapper.EventID(r.Type())
	if !ok {
		id = 1000
	}
	eid := uint32(id)

	// Log the event according to the event level.
	switch level := r.Level(); {
	case level >= slog.LevelError:
		err = h.elog.Error(eid, eventMessageWithDetails(r))
	case level >= slog.LevelWarn:
		err = h.elog.Warning(eid, eventMessageWithDetails(r))
	case level >= slog.LevelInfo:
		err = h.elog.Info(eid, eventMessageWithDetails(r))
	default:
		return nil // Drop debug messages.
	}

	// If we failed to log the event, try again without the message details.
	if err != nil {
		switch level := r.Level(); {
		case level >= slog.LevelError:
			h.elog.Error(eid, r.Message())
		case level >= slog.LevelWarn:
			h.elog.Warning(eid, r.Message())
		case level >= slog.LevelInfo:
			h.elog.Info(eid, r.Message())
		}
	}

	return err
}

// Close releases any resources consumed by the Windows event handler.
func (handler Handler) Close() error {
	return handler.elog.Close()

	// Remove the event source if needed
	// eventlog.Remove(eventSource)
}

// IsWindowsEventSourceRegistered checks to see whether an event log with the
// given source name has been registered.
func IsWindowsEventSourceRegistered(eventSource string) (bool, error) {
	const addKeyName = `SYSTEM\CurrentControlSet\Services\EventLog\Application`

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, addKeyName+`\`+eventSource, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	defer key.Close()

	return true, nil
}

func eventMessageWithDetails(r lbevent.Record) string {
	message := r.Message()
	if details := r.Details(); details != "" {
		return fmt.Sprintf("%s\n\n%s", message, details)
	}
	return message
}
