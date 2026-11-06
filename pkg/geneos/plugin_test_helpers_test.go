package geneos

import (
	"encoding/xml"
	"strings"
	"testing"
)

func marshalPlugin(t *testing.T, value any) []byte {
	t.Helper()
	data, err := xml.Marshal(value)
	if err != nil {
		t.Fatalf("marshal plugin: %v", err)
	}
	return data
}

func unmarshalPlugin(t *testing.T, data []byte, value any) {
	t.Helper()
	if err := xml.Unmarshal(data, value); err != nil {
		t.Fatalf("unmarshal plugin: %v", err)
	}
}

func assertPluginContains(t *testing.T, data []byte, expected string) {
	t.Helper()
	if !strings.Contains(string(data), expected) {
		t.Errorf("plugin XML missing %q in %s", expected, data)
	}
}

func assertXMLRoundTrip(t *testing.T, value any, original []byte) {
	t.Helper()
	encoded, err := xml.Marshal(value)
	if err != nil {
		t.Fatalf("remarshal XML: %v", err)
	}
	if string(encoded) != string(original) {
		t.Errorf("XML changed on round-trip:\noriginal: %s\nencoded:  %s", original, encoded)
	}
}
