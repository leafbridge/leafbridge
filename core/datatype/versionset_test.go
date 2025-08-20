package datatype_test

import (
	"strings"
	"testing"

	"github.com/leafbridge/leafbridge/core/datatype"
)

func TestVersionSet(t *testing.T) {
	set := make(datatype.VersionSet)
	set.Add("1")
	set.Add("1")
	set.Add("1.0")
	set.Add("1.0.0")
	set.Remove("1.0")
	if set.Contains("1") {
		t.Fatalf("the set should not contain the version \"1\"")
	}
	if len(set.List()) != 0 {
		t.Fatalf("the set should be empty")
	}
	set.Add("1.0")
	set.Remove("1.0")
	set.Add("2")
	set.Add("1")
	if !set.Contains("2") || !set.Contains("2.0.0.0.0") {
		t.Fatalf("the set should contain the version \"2\"")
	}
	if set.Max() != "2" {
		t.Fatalf("the set's maximum value should be \"2\"")
	}
	set.Add("5.2.1")
	if set.Max() != "5.2.1" {
		t.Fatalf("the set's maximum value should be \"5.2.1\"")
	}
	set.Add("27.2")
	set.Add("2.2.1.5")
	if set.Max() != "27.2" {
		t.Fatalf("the set's maximum value should be \"27.2\"")
	}
	var values []string
	for _, version := range set.List() {
		values = append(values, string(version))
	}
	result := strings.Join(values, ", ")
	const expected = "1, 2, 2.2.1.5, 5.2.1, 27.2"
	if result != expected {
		t.Fatalf("the resulting set did not contain the expected values: got \"%s\" (wanted \"%s\")", result, expected)
	}
}
