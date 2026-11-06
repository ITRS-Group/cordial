package testhelpers

import (
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func Marshal(t testing.TB, value any) []byte {
	t.Helper()
	data, err := xml.Marshal(value)
	if err != nil {
		t.Fatalf("marshal XML: %v", err)
	}
	return data
}

func Unmarshal(t testing.TB, data []byte, value any) {
	t.Helper()
	if err := xml.Unmarshal(data, value); err != nil {
		t.Fatalf("unmarshal XML: %v", err)
	}
}

func Contains(t testing.TB, data []byte, expected string) {
	t.Helper()
	if !strings.Contains(string(data), expected) {
		t.Errorf("XML missing %q in %s", expected, data)
	}
}

func AssertStable(t testing.TB, value any, original []byte) {
	t.Helper()
	encoded := Marshal(t, value)
	if string(encoded) != string(original) {
		t.Errorf("XML changed on round-trip:\noriginal: %s\nencoded:  %s", original, encoded)
	}
}

func RoundTrip(t testing.TB, value any) []byte {
	t.Helper()
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Pointer {
		t.Fatalf("round-trip value must be a pointer, got %T", value)
	}
	data := Marshal(t, value)
	decoded := reflect.New(valueType.Elem()).Interface()
	Unmarshal(t, data, decoded)
	AssertStable(t, decoded, data)
	return data
}
