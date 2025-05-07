package lbevent

// MultiHandler is a LeafBridge event handler that sends events to multiple
// underlying handlers.
type MultiHandler []Handler

// Name returns a name for the handler.
func (h MultiHandler) Name() string {
	return "multi-handler"
}

// Handle processes the given event record.
func (h MultiHandler) Handle(r Record) error {
	var errs []error
	for _, handler := range h {
		if err := handler.Handle(r); err != nil {
			errs = append(errs, WrapHandlerError(handler, r, err))
		}
	}

	return WrapHandlerError(h, r, errs...)
}
