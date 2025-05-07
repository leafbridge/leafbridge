package lbevent

import (
	"fmt"
	"io"
	"log/slog"
)

const timestampFormat = "2006-01-02 15:04:05"

// BasicHandler is a LeafBridge event handler that prints timestamped event
// messages to an io.Writer.
type BasicHandler struct {
	w   io.Writer
	min slog.Level
}

// NewBasicHandler returns a BasicHandler that will write to w.
// Events below the provided minimum level will be ignored.
func NewBasicHandler(w io.Writer, min slog.Level) BasicHandler {
	return BasicHandler{
		w:   w,
		min: min,
	}
}

// Name returns a name for the handler.
func (h BasicHandler) Name() string {
	return "basic"
}

// Handle processes the given event record.
func (h BasicHandler) Handle(r Record) error {
	if r.Level() < h.min {
		return nil
	}
	fmt.Fprintf(h.w, "%s: %-6s %s\n", r.Time().Local().Format(timestampFormat), r.Level().String()+":", r.Message())
	return nil
}
