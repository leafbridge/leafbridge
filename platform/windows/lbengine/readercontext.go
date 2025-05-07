package lbengine

import (
	"context"
	"io"
)

// readerWithContext holds an io.Reader that will stop reading data when a
// corresponding context is cancelled.
type readerWithContext struct {
	ctx context.Context
	r   io.Reader
}

// newReaderWithContext returns a wrapper for the given io.Reader that will
// stop reading when the provided context returns an error.
func newReaderWithContext(ctx context.Context, r io.Reader) readerWithContext {
	return readerWithContext{
		ctx: ctx,
		r:   r,
	}
}

// Read reads from an underlying io.Reader.
//
// If a corresponding context has been cancelled, it returns the error from
// the context.
func (r readerWithContext) Read(p []byte) (n int, err error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
