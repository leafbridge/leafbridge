package bytesconv

import (
	"encoding/binary"
	"unicode/utf16"
)

// ParseUTF16 parses the given bytes as UTF-16 with the specified byte order
// and returns the value as a string. If it fails, it returns an error.
func ParseUTF16(p []byte, order binary.ByteOrder) (string, error) {
	// If there is no data, return an empty string.
	if len(p) == 0 {
		return "", nil
	}

	// If the data is not an even length, it can't be properly encoded as
	// UTF-16.
	if len(p)%2 != 0 {
		return "", ErrUnevenUTF16
	}

	// Prepare a buffer for the decoded runes.
	output := make([]rune, 0, len(p)/2)

	// Consume two bytes of the input data at a time.
	for i := 0; i+1 < len(p); i += 2 {
		// Take the next rune in the desired byte order.
		r1 := rune(order.Uint16(p[i:]))

		// If this rune is not part of a surrogate pair, add it.
		if r1 < surr1 || surr3 <= r1 {
			output = append(output, r1)
			continue
		}

		// If this rune is part of a valid surrogate pair, add it.
		if surr1 <= r1 && r1 <= surr2 && i+3 < len(p) {
			r2 := rune(order.Uint16(p[i+2:]))
			if surr2 <= r2 && r2 <= surr3 {
				i += 2
				output = append(output, utf16.DecodeRune(r1, r2))
				continue
			}
		}

		// The UTF-16 data is invalid.
		return "", ErrInvalidUTF16
	}

	// Convert the runes to a string.
	return string(output), nil
}

// DecodeUTF16 interprets the given bytes as UTF-16 with the specified byte
// order and returns the value as a string.
//
// Any invalid characters will be replaced with the unicode replacement
// character.
func DecodeUTF16(p []byte, order binary.ByteOrder) string {
	// If the number of bytes is not an even number, drop the last byte.
	n := len(p)
	if n%2 != 0 {
		n--
	}

	// If there is no data, return an empty string.
	if n < 1 {
		return ""
	}

	// Prepare a buffer to receive the bytes.
	buf := make([]uint16, 0, n/2)

	// Parse the bytes in the desired order.
	for i := 0; i+1 < n; i += 2 {
		buf = append(buf, order.Uint16(p[i:]))
	}

	// Decode the runes and convert them to a string.
	return string(utf16.Decode(buf))
}

// HasUTF16BOM returns true if the given bytes have a unicode byte order mark
// with the specified byte order.
func HasUTF16BOM(p []byte, order binary.ByteOrder) bool {
	if len(p) < 2 {
		return false
	}

	return order.Uint16(p) == 0xFEFF
}

// These constants are taken from the standard library's utf16 package.
const (
	// 0xd800-0xdc00 encodes the high 10 bits of a pair.
	// 0xdc00-0xe000 encodes the low 10 bits of a pair.
	// the value is those 20 bits plus 0x10000.
	surr1 = 0xd800
	surr2 = 0xdc00
	surr3 = 0xe000
)
