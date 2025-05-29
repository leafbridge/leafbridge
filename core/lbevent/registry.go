package lbevent

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Registry holds event registrations for event types. It facilitates the
// unmarshaling of event data, and it assigns unique event ID numbers to each
// event that is registered.
type Registry struct {
	mutex        sync.RWMutex
	types        []Type
	ids          map[Type]ID
	unmarshalers map[Type]RecordUnmarshaler
	next         ID
}

// NewRegistry prepares a new event registry and returns it.
//
// The provided starting ID will be assigned to the first event type that is
// registered.
func NewRegistry(start ID) *Registry {
	return &Registry{
		ids:          make(map[Type]ID),
		unmarshalers: make(map[Type]RecordUnmarshaler),
		next:         start,
	}
}

// Add adds the given events to the event registry in the order provided.
//
// As new events are added, monotonically increasing event IDs are assigned
// to them by the registry.
//
// If an existing registration exists for an event, the registration is
// updated but the previously assigned event ID is preserved.
func (r *Registry) Add(events ...Registration) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, event := range events {
		if _, exists := r.ids[event.Type]; !exists {
			id := r.next
			r.next++
			r.ids[event.Type] = id
			r.types = append(r.types, event.Type)
		}
		r.unmarshalers[event.Type] = event.Unmarshaler
	}
}

// EventID returns the registered event [ID] for the given event [Type].
//
// If the event type has not been registered, it returns false.
func (r *Registry) EventID(event Type) (id ID, ok bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	id, ok = r.ids[event]
	return
}

// Types returns an ordered list of all event types that have been registered.
func (r *Registry) Types() []Type {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.types
}

// UnmarshalRecord attempts to unmarshal the given JSON data as an event
// record.
//
// It consults the registered event types to locate an appropriate event
// record unmarshaling function. If it cannot find an unmarshaling function,
// or if the unmarshaling function fails, an error is returned.
func (r *Registry) UnmarshalRecord(b []byte) (Record, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var header struct {
		Type Type `json:"type"`
	}
	if err := json.Unmarshal(b, &header); err != nil {
		return nil, fmt.Errorf("failed to parse event record header: %w", err)
	}

	unmarshaler, found := r.unmarshalers[header.Type]
	if !found {
		return nil, fmt.Errorf("the event type \"%s\" could not be umarshaled because it is not recognized", header.Type)
	}

	return unmarshaler(b)
}
