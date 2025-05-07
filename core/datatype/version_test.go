package datatype_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leafbridge/leafbridge/core/datatype"
)

var basicVersionFixtures = []datatype.Version{
	"",
	"1",
	"1.2",
	"1.2.3.4.5.6",
	"10.2.78.212341.2",
	"1.2..4",
}

type versionInOut struct {
	In, Out datatype.Version
}

var complexVersionFixtures = []versionInOut{
	{In: "vA5", Out: "A5"},
	{In: "v52.21A", Out: "52.21A"},
	{In: "1.2.", Out: "1.2"},
}

func TestVersion(t *testing.T) {
	for i, in := range basicVersionFixtures {
		t.Run(fmt.Sprintf("Basic.%d:%s", i, in), func(t *testing.T) {
			var segments []string
			for segment := range in.Segments() {
				segments = append(segments, string(segment))
			}
			out := strings.Join(segments, ".")
			if in != datatype.Version(out) {
				t.Fatalf("version not equal after round trip: %v → %v", in, out)
			}
		})
	}

	for i, fixture := range complexVersionFixtures {
		t.Run(fmt.Sprintf("Complex.%d:%s:%s", i, fixture.In, fixture.Out), func(t *testing.T) {
			var segments []string
			for segment := range fixture.In.Segments() {
				segments = append(segments, string(segment))
			}
			out := strings.Join(segments, ".")
			if fixture.Out != datatype.Version(out) {
				t.Fatalf("unexpected version segmentation: %v → %v (want %v)", fixture.In, out, fixture.Out)
			}
		})
	}
}

type versionComparison struct {
	A, B   datatype.Version
	Result int
}

func compSymbol(result int) string {
	switch result {
	case 0:
		return "=="
	case -1:
		return "<"
	case 1:
		return ">"
	default:
		return "?"
	}
}

var versionComparisonFixtures = []versionComparison{
	{A: "1", B: "1", Result: 0},
	{A: "1.", B: "1", Result: 0},
	{A: "1", B: "1.", Result: 0},
	{A: "v1", B: "1", Result: 0},
	{A: "v1.", B: "1", Result: 0},
	{A: "v1", B: "1.", Result: 0},
	{A: "1A", B: "1A", Result: 0},
	{A: "2024.41.2.A", B: "2024.41.2.A", Result: 0},
	{A: "1", B: "1.1", Result: -1},
	{A: "1.", B: "1.1", Result: -1},
	{A: "1.1", B: "1.2", Result: -1},
	{A: "1.1", B: "2.1", Result: -1},
	{A: "000001", B: "0000010", Result: -1},
	{A: "01.1", B: "1.2", Result: -1},
	{A: "1.1", B: "01.2", Result: -1},
	{A: "1.2.3.4.5", B: "5.4.3.2.1", Result: -1},
	{A: "1.2.3.4.5", B: "5.4.3.2", Result: -1},
	{A: "1.2.3.4.5", B: "5.4.3", Result: -1},
	{A: "1.2.", B: "5.4.3", Result: -1},
	{A: "1A", B: "1B", Result: -1},
	{A: "1.A", B: "1.B", Result: -1},
	{A: "A.B", B: "A.C", Result: -1},
	{A: "A.B", B: "A.C.", Result: -1},
	{A: "A.B", B: "A.C..", Result: -1},
	{A: "A", B: "A.A", Result: -1},
	{A: "A", B: "A.1", Result: -1},
	{A: "B100", B: "A1000", Result: -1},
	{A: "2024.41.2.A", B: "2024.41.2.B", Result: -1},
	{A: "2024.41.0.2.A", B: "2024.41.2.A", Result: -1},
	{A: "000000000000000000000000000000000000000000001", B: "0000010", Result: -1},
	{A: "100000000000000000000000000000000000000000000", B: "200000000000000000000000000000000000000000000", Result: -1},
	{A: "200000000000000000000000000000000000000000000", B: "1000000000000000000000000000000000000000000000", Result: -1},
}

func TestCompareVersions(t *testing.T) {
	for i, fixture := range versionComparisonFixtures {
		t.Run(fmt.Sprintf("Comparison.%d:%s%s%s", i, fixture.A, compSymbol(fixture.Result), fixture.B), func(t *testing.T) {
			result := datatype.CompareVersions(fixture.A, fixture.B)
			if result != fixture.Result {
				t.Fatalf("unexpected comparison result: %s (want %s)", compSymbol(result), compSymbol(fixture.Result))
			}
			reversed := datatype.CompareVersions(fixture.B, fixture.A)
			if reversed != -fixture.Result {
				t.Fatalf("unexpected inverted comparison result: %s (want %s)", compSymbol(reversed), compSymbol(-fixture.Result))
			}
		})
	}
}
