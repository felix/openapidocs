package openapidocs

import (
	"os"
	"testing"
)

func TestJSONSpecParsing(t *testing.T) {
	pathCount := 2
	jsonFile := mustOpen(t, "testdata/example.json")
	defer jsonFile.Close()
	jsonSpec, err := ReadSpecFromJSON(jsonFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(jsonSpec.Paths) != pathCount {
		t.Fatalf("got %d, want %d", len(jsonSpec.Paths), pathCount)
	}

	yamlFile := mustOpen(t, "testdata/example.json")
	defer yamlFile.Close()
	yamlSpec, err := ReadSpecFromYAML(yamlFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(yamlSpec.Paths) != pathCount {
		t.Fatalf("got %d, want %d", len(yamlSpec.Paths), pathCount)
	}
}

func mustOpen(t *testing.T, s string) *os.File {
	f, err := os.Open(s)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
