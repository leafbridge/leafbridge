package lbevent

// Registration holds information about an event that can be added to an event
// [Registry].
type Registration struct {
	Type        Type
	Unmarshaler RecordUnmarshaler
}
