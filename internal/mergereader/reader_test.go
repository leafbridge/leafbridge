package mergereader_test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/leafbridge/leafbridge/internal/mergereader"
)

func TestReader(t *testing.T) {
	r1, r2, err := makeTwoPipes(10)
	if err != nil {
		t.Fatal(err)
	}

	merged := mergereader.New(r1, r2)
	data, err := io.ReadAll(merged)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))
}

func TestReaderConstrained(t *testing.T) {
	r1, r2, err := makeTwoPipes(10)
	if err != nil {
		t.Fatal(err)
	}

	merged := mergereader.New(r1, r2)
	constrained := newConstrainedReader(merged, 3)
	data, err := io.ReadAll(constrained)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))
}

func makeTwoPipes(writes int) (r1, r2 *os.File, err error) {
	r1, w1, err1 := os.Pipe()
	if err1 != nil {
		return nil, nil, err1
	}

	r2, w2, err2 := os.Pipe()
	if err2 != nil {
		w1.Close()
		r1.Close()
		return nil, nil, err2
	}

	go func() {
		for range writes {
			w1.WriteString("Hello from pipe 1\n")
			time.Sleep(10 * time.Millisecond)
		}
		w1.Close()
	}()

	go func() {
		for range writes {
			w2.WriteString("Hello from pipe 2\n")
			time.Sleep(10 * time.Millisecond)
		}
		w2.Close()
	}()

	return r1, r2, nil
}

type constrainedReader struct {
	r   io.Reader
	max int
}

func newConstrainedReader(r io.Reader, max int) io.Reader {
	return &constrainedReader{
		r:   r,
		max: max,
	}
}

func (c constrainedReader) Read(p []byte) (n int, err error) {
	if len(p) > c.max {
		p = p[:c.max]
	}
	return c.r.Read(p)
}
