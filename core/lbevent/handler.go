package lbevent

// Handler is an event handler that is capable of processing events in
// LeafBridge.
type Handler interface {
	Name() string
	Handle(Record) error
}
