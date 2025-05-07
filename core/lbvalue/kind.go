package lbvalue

import (
	"fmt"
)

// Kind identifies the type of a [Value].
type Kind int

// Supported variable kinds.
const (
	KindUnknown Kind = iota
	KindBool
	KindInt64
	KindString
	KindVersion

	// TODO: Add types from the netip package to be used in network detection.
	//KindNetAddr
	//KindNetPrefix
)

var kindStrings = []string{
	"Unknown",
	"Bool",
	"Int64",
	"String",
	"Version",
}

var kindStringsLower = []string{
	"unknown",
	"bool",
	"int64",
	"string",
	"version",
}

// String returns a string representation of k.
func (k Kind) String() string {
	if k := int(k); k >= 0 && k < len(kindStrings) {
		return kindStrings[k]
	}
	return fmt.Sprintf("<unknown kind \"%d\">", k)
}

// UnmarshalText attempts to unmarshal the given text into k.
func (k *Kind) UnmarshalText(b []byte) error {
	switch string(b) {
	case "unknown":
		*k = KindUnknown
	case "bool":
		*k = KindBool
	case "int64":
		*k = KindInt64
	case "string":
		*k = KindString
	case "version":
		*k = KindVersion
	default:
		return fmt.Errorf("unrecognized kind: %s", b)
	}
	return nil
}

// MarshalText marshals the kind as text.
func (k Kind) MarshalText() ([]byte, error) {
	if k := int(k); k >= 0 && k < len(kindStringsLower) {
		return []byte(kindStringsLower[k]), nil
	}
	return nil, fmt.Errorf("unrecognized kind: %d", k)
}
