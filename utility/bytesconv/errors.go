package bytesconv

import "errors"

var (
	// ErrInvalidUTF16 is returned when the provided bytes are not valid
	// UTF-16.
	ErrInvalidUTF16 = errors.New("the UTF-16 data is invalid")

	// ErrUnevenUTF16 is returned when the provided bytes are not an even
	// length. The UTF-16 encoding requires an even number of bytes.
	ErrUnevenUTF16 = errors.New("the UTF-16 data is not an even length")
)
