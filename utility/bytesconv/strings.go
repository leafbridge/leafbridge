package bytesconv

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"unicode/utf8"
)

// DecodeString attempts to interpret the given bytes as a string. If the
// bytes are valid UTF-8, they are returned as a string without modification.
//
// If the bytes are not valid UTF-8, it attempts to interpret them as UTF-16.
// If successful, it converts the bytes to UTF-8 and returns the resulting
// string.
//
// If a unicode encoding is not detected, or conversion to a string is not
// successful, it returns the bytes as a Base64 raw URL-encoded string.
func DecodeString(p []byte) string {
	// If there is no data, return an empty string
	if len(p) == 0 {
		return ""
	}

	// If the data has an obvious unicode byte order mark at the start of it,
	// obey it.
	switch {
	case HasUTF16BOM(p, binary.LittleEndian):
		return DecodeUTF16(p[2:], binary.LittleEndian)
	case HasUTF16BOM(p, binary.BigEndian):
		return DecodeUTF16(p[2:], binary.BigEndian)
	}

	// If the data is already valid UTF-8 and it doesn't have a null character
	// in it, return it as-is.
	if utf8.Valid(p) && !bytes.ContainsRune(p, 0) {
		return string(p)
	}

	// Attempt to parse the data as UTF-16 LE.
	if s, err := ParseUTF16(p, binary.LittleEndian); err == nil {
		return s
	}

	// Attempt to parse the data as UTF-16 BE.
	if s, err := ParseUTF16(p, binary.BigEndian); err == nil {
		return s
	}

	// Encode the data as Base64 as a last resort.
	return base64.RawURLEncoding.EncodeToString(p)
}
