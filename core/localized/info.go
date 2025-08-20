package localized

import (
	"encoding/json"

	"golang.org/x/text/language"
)

// Info is a structure the encodes information for a particular locale.
type Info[T any] struct {
	Locale language.Tag `json:"locale,omitzero"`
	Data   T            // TODO: When Go 1.26 adds support: `json:",inline"``json:",inline"`

	// TODO: Consider directly embedding T if Go supports that in the future.
}

// UnmarshalJSON attempts to unmarshal the given JSON data into info.
//
// TODO: Remove this when the JSON v2 library is no longer experimental and
// we can inline the Data field.
func (info *Info[T]) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &info.Data); err != nil {
		return err
	}

	aux := struct {
		Locale *language.Tag `json:"locale,omitempty"`
	}{
		Locale: &info.Locale,
	}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	return nil
}

// MarshalJSON marshals the localized info as JSON data.
//
// TODO: Remove this when the JSON v2 library is no longer experimental and
// we can inline the Data field.
func (info Info[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(info.Data)
}
