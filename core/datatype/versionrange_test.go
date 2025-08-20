package datatype_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/leafbridge/leafbridge/core/datatype"
)

type versionRangeFixture struct {
	Version       datatype.Version
	RangeString   string
	MarshaledJSON string
	Range         datatype.VersionRange
	Included      bool
}

var versionRangeFixtures = []versionRangeFixture{
	{Version: "", RangeString: "", Range: datatype.VersionRange{}, Included: true},
	{Version: "", RangeString: "<=1", Range: datatype.VersionRange{End: "1"}, Included: false},
	{Version: "1.0.0", RangeString: "", Range: datatype.VersionRange{}, Included: true},
	{Version: "1.0.0", RangeString: ">=0.9.0", Range: datatype.VersionRange{Start: "0.9.0"}, Included: true},
	{Version: "1.0.0", RangeString: ">=1.0.0", Range: datatype.VersionRange{Start: "1.0.0"}, Included: true},
	{Version: "1.0.0", RangeString: ">1.0.0", Range: datatype.VersionRange{Start: "1.0.0", StartIsExclusive: true}, Included: false},
	{Version: "1.0.0", RangeString: "<=1.0.1", Range: datatype.VersionRange{End: "1.0.1"}, Included: true},
	{Version: "1.0.0", RangeString: "<=1.0.0", Range: datatype.VersionRange{End: "1.0.0"}, Included: true},
	{Version: "1.0.0", RangeString: "<1.0.0", Range: datatype.VersionRange{End: "1.0.0", EndIsExclusive: true}, Included: false},
	{Version: "0.5.2", RangeString: "(0.1.8, 0.7)", Range: datatype.VersionRange{Start: "0.1.8", End: "0.7"}, Included: true},
	{Version: "0.5.2", RangeString: "(0.4, 8.25.1.3)", Range: datatype.VersionRange{Start: "0.4", End: "8.25.1.3"}, Included: true},
	{Version: "1.0.0", RangeString: "(0.0, 0.9)", Range: datatype.VersionRange{Start: "0.0", End: "0.9"}, Included: false},
	{Version: "0.1.1", RangeString: "(0.2, 9)", Range: datatype.VersionRange{Start: "0.2", End: "9"}, Included: false},
	{Version: "1.0.0", RangeString: "(1.0.0, 1.2)", Range: datatype.VersionRange{Start: "1.0.0", End: "1.2"}, Included: true},
	{Version: "1.0.0", RangeString: "[1.0.0, 1.2)", Range: datatype.VersionRange{Start: "1.0.0", StartIsExclusive: true, End: "1.2"}, Included: false},
	{Version: "1.2.0", RangeString: "(1.0.0, 1.2)", Range: datatype.VersionRange{Start: "1.0.0", End: "1.2"}, Included: true},
	{Version: "1.2.0", RangeString: "(1.0.0, 1.2]", Range: datatype.VersionRange{Start: "1.0.0", End: "1.2", EndIsExclusive: true}, Included: false},
	{Version: "2", RangeString: "[1, 3]", Range: datatype.VersionRange{Start: "1", StartIsExclusive: true, End: "3", EndIsExclusive: true}, Included: true},
	{Version: "2", RangeString: "[1, 2]", Range: datatype.VersionRange{Start: "1", StartIsExclusive: true, End: "2", EndIsExclusive: true}, Included: false},
	{Version: "2", RangeString: "[2, 3]", Range: datatype.VersionRange{Start: "2", StartIsExclusive: true, End: "3", EndIsExclusive: true}, Included: false},
	{Version: "2", RangeString: "[2, 2]", Range: datatype.VersionRange{Start: "2", StartIsExclusive: true, End: "2", EndIsExclusive: true}, Included: false},
	{Version: "1.2.0", RangeString: "[1.2.0, 1.2]", Range: datatype.VersionRange{Start: "1.2.0", StartIsExclusive: true, End: "1.2", EndIsExclusive: true}, Included: false},
}

func TestVersionRange(t *testing.T) {
	for i, fixture := range versionRangeFixtures {
		t.Run(fmt.Sprintf("%d:%s:%s", i, fixture.RangeString, fixture.Version), func(t *testing.T) {
			{
				result := fixture.Range.Includes(fixture.Version)
				if result != fixture.Included {
					t.Fatalf("version range inclusion test doesn't match the expected result: got %t (wanted %t)", result, fixture.Included)
				}
			}
			{
				result := fixture.Range.String()
				if result != fixture.RangeString {
					t.Fatalf("version range string doesn't match the expected result: got %s (wanted %s)", result, fixture.RangeString)
				}
			}
		})
	}
}

type versionRangeMarshalingFixture struct {
	JSON                      string
	Range                     datatype.VersionRange
	ExpectedUnmarshalingError bool
}

var versionRangeMarshalingFixtures = []versionRangeMarshalingFixture{
	{Range: datatype.VersionRange{}, JSON: `{}`},
	{Range: datatype.VersionRange{}, JSON: `{"start":"1","after":"1","end":"3"}`, ExpectedUnmarshalingError: true},
	{Range: datatype.VersionRange{}, JSON: `{"start":"1","before":"2","end":"3"}`, ExpectedUnmarshalingError: true},
	{Range: datatype.VersionRange{}, JSON: `5`, ExpectedUnmarshalingError: true},
	{Range: datatype.VersionRange{Start: "1", End: "2"}, JSON: `{"start":"1","end":"2"}`},
	{Range: datatype.VersionRange{Start: "0.1", End: "2.1.1"}, JSON: `{"start":"0.1","end":"2.1.1"}`},
	{Range: datatype.VersionRange{Start: "1.0", StartIsExclusive: true, End: "4"}, JSON: `{"after":"1.0","end":"4"}`},
	{Range: datatype.VersionRange{Start: "4", StartIsExclusive: true, End: "5.8.2B", EndIsExclusive: true}, JSON: `{"after":"4","before":"5.8.2B"}`},
	{Range: datatype.VersionRange{Start: "v21.5.25128", End: "v21.6.27197", EndIsExclusive: true}, JSON: `{"start":"v21.5.25128","before":"v21.6.27197"}`},
}

func TestVersionRangeMarshaling(t *testing.T) {
	for i, fixture := range versionRangeMarshalingFixtures {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			// Unmarshaling.
			var unmarshaled datatype.VersionRange
			if err := json.Unmarshal([]byte(fixture.JSON), &unmarshaled); err != nil {
				if fixture.ExpectedUnmarshalingError {
					return
				} else {
					t.Fatalf("unmarshaling %q encountered an unexpected error: %v", fixture.JSON, err)
				}
			} else if fixture.ExpectedUnmarshalingError {
				t.Fatalf("unmarshaling %q did not produce an error as expected", fixture.JSON)
			}

			if unmarshaled != fixture.Range {
				t.Fatalf("unmarshaled version range doesn't match the expected result: got \"%#v\" (wanted \"%#v\")", unmarshaled, fixture.Range)
			}

			// Marshaling.
			marshaled, err := json.Marshal(fixture.Range)
			if err != nil {
				t.Fatalf("marshaling \"%#v\" encountered an unexpected error: %v", fixture.Range, err)
			}

			if string(marshaled) != fixture.JSON {
				t.Fatalf("marshaled version range doesn't match the expected result: got %s (wanted %s)", string(marshaled), fixture.JSON)
			}
		})
	}
}
