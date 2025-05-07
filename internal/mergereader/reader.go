package mergereader

import (
	"io"
	"sync"
)

// MergeReader multiplexes multiple readers into a single reader.
type MergeReader struct {
	ch     <-chan dataChunk
	unread *unreadChunk
}

// New returns a new MergeReader for the given set of readers.
func New(readers ...io.Reader) MergeReader {
	ch := make(chan dataChunk)

	var wg sync.WaitGroup
	wg.Add(len(readers))

	for _, reader := range readers {
		go copyToChannel(reader, ch, wg.Done)
	}

	go func() {
		defer close(ch)
		wg.Wait()
	}()

	return MergeReader{
		ch:     ch,
		unread: &unreadChunk{},
	}
}

type unreadChunk struct {
	offset int
	size   int
	data   [chunkSize]byte
}

func (unread *unreadChunk) TryRead(p []byte) (n int) {
	if unread.offset >= unread.size {
		return 0
	}
	copied := copy(p, unread.data[unread.offset:unread.size])
	unread.offset += copied
	return copied
}

func (unread *unreadChunk) Write(incoming *dataChunk, offset int) {
	unread.offset = 0
	unread.size = copy(unread.data[:], incoming.data[offset:incoming.size])
}

const chunkSize = 4096

type dataChunk struct {
	err  error
	size int
	data [chunkSize]byte
}

type dataChannel chan dataChunk

func (r MergeReader) Read(p []byte) (n int, err error) {
	if n := r.unread.TryRead(p); n > 0 {
		return n, nil
	}
	chunk, ok := <-r.ch
	if !ok {
		return 0, io.EOF
	}
	copied := copy(p, chunk.data[:chunk.size])
	if copied < chunk.size {
		r.unread.Write(&chunk, copied)
	}
	return copied, chunk.err
}

func copyToChannel(r io.Reader, ch chan<- dataChunk, done func()) {
	defer done()

	var chunk dataChunk
	for {
		var err error
		chunk.size, err = r.Read(chunk.data[:])
		if err != nil && err != io.EOF {
			chunk.err = err
		} else {
			chunk.err = nil
		}
		if chunk.size > 0 || chunk.err != nil {
			ch <- chunk
		}
		if err != nil {
			return
		}
	}
}
