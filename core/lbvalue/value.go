package lbvalue

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/leafbridge/leafbridge/core/datatype"
)

// Value is a value that can be stored in a variable or constant.
//
// Its design is inspired by the Value type in the slog package.
type Value struct {
	num  uint64
	data any
}

// Bool returns a [Value] representing the boolean v.
func Bool(v bool) Value {
	var num uint64
	if v {
		num = 1
	}
	return Value{num: num, data: KindBool}
}

// Int64 returns a [Value] representing the int64 v.
func Int64(v int64) Value {
	return Value{num: uint64(v), data: KindInt64}
}

// String returns a [Value] representing the string v.
func String(v string) Value {
	return Value{data: v}
}

// Version returns a [Value] representing the version v.
func Version(v datatype.Version) Value {
	return Value{data: v}
}

// Kind returns the kind of the value.
func (v Value) Kind() Kind {
	switch data := v.data.(type) {
	case Kind:
		return data
	case string:
		return KindString
	case datatype.Version:
		return KindVersion
	default:
		return KindUnknown
	}
}

// Bool returns the value as a bool.
func (v Value) Bool() bool {
	if kind, ok := v.data.(Kind); ok && kind == KindBool {
		return v.num == 1
	}
	return false
}

// Int64 returns the value as an int64.
func (v Value) Int64() int64 {
	if kind, ok := v.data.(Kind); ok && kind == KindInt64 {
		return int64(v.num)
	}
	return 0
}

// String returns the value as a string.
//
// If the underlying data type is not a string, a string represenation of
// the value is returned.
func (v Value) String() string {
	switch data := v.data.(type) {
	case Kind:
		switch data {
		case KindBool:
			return strconv.FormatBool(v.Bool())
		case KindInt64:
			return strconv.FormatInt(int64(v.num), 10)
		}
	case string:
		return data
	case datatype.Version:
		return string(data)
	}
	return ""
}

// Version returns the value as a [datatype.Version].
func (v Value) Version() datatype.Version {
	if value, ok := v.data.(datatype.Version); ok {
		return value
	}
	return ""
}

// UnmarshalJSON attempts to unmarshal the given JSON data into v.
func (v *Value) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return errors.New("the value type could not be determined")
	}

	switch symbol := b[0]; {
	case symbol == '"':
		var aux string
		if err := json.Unmarshal(b, &aux); err != nil {
			return err
		}
		*v = String(aux)
	case symbol == '-', '0' <= symbol && symbol <= '9':
		var aux int64
		if err := json.Unmarshal(b, &aux); err != nil {
			return err
		}
		*v = Int64(aux)
	case symbol == '{':
		var keys keySet
		if err := json.Unmarshal(b, &keys); err != nil {
			return err
		}
		switch {
		case keys.Contains("version"):
			var aux versionJSON
			if err := json.Unmarshal(b, &aux); err != nil {
				return err
			}
			*v = Version(aux.Version)
		default:
			return errors.New("the value type could not be determined")
		}
	default:
		return errors.New("the value type could not be determined")
	}

	return nil
}

// MarshalJSON marshals the value as JSON data.
func (v Value) MarshalJSON() ([]byte, error) {
	switch data := v.data.(type) {
	case Kind:
		switch data {
		case KindBool:
			return json.Marshal(v.Bool())
		case KindInt64:
			return json.Marshal(v.Int64())
		default:
			return nil, errors.New("cannot marshal value of unknown kind")
		}
	case string:
		return json.Marshal(data)
	case datatype.Version:
		return json.Marshal(versionJSON{Version: data})
	default:
		return nil, errors.New("cannot marshal value of unknown kind")
	}
}

type keySet map[string]any

func (set keySet) Contains(key string) bool {
	_, ok := set[key]
	return ok
}

type versionJSON struct {
	Version datatype.Version `json:"version"`
}
