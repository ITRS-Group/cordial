package responses

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestNewAndFinished(t *testing.T) {
	resp := New[General](nil)
	if resp.Dataview == nil {
		t.Fatal("New() returned a response without a dataview")
	}
	Finished(resp)
	if resp.Duration < 0 {
		t.Errorf("Finished() set negative duration %s", resp.Duration)
	}
}

func TestMergeResponse(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")
	first := &General{
		Value:      "first",
		Completed:  []string{"created"},
		ResultText: []string{"first detail"},
		Duration:   2,
		Err:        firstErr,
		Dataview:   &Dataview{Table: [][]string{{"first"}}},
		Values:     []any{"one"},
	}
	second := &General{
		Value:     "second",
		Completed: []string{"updated"},
		Duration:  3,
		Err:       secondErr,
		Dataview:  &Dataview{Table: [][]string{{"second"}}},
		Values:    []any{"two"},
	}

	got := MergeResponse(first, second)
	values, ok := got.Value.([]any)
	if !ok || !reflect.DeepEqual(values, []any{"first", "second"}) {
		t.Errorf("merged Value = %#v, want both values", got.Value)
	}
	if !errors.Is(got.Err, firstErr) || !errors.Is(got.Err, secondErr) {
		t.Errorf("merged Err = %v, want both original errors", got.Err)
	}
	if !reflect.DeepEqual(got.Completed, []string{"created", "updated"}) {
		t.Errorf("merged Completed = %#v", got.Completed)
	}
	if !reflect.DeepEqual(got.Dataview.Table, [][]string{{"first"}, {"second"}}) {
		t.Errorf("merged table = %#v", got.Dataview.Table)
	}
	if !reflect.DeepEqual(got.Values, []any{"one", "two"}) {
		t.Errorf("merged Values = %#v", got.Values)
	}
	if !reflect.DeepEqual(first.ResultText, []string{"first detail"}) {
		t.Errorf("MergeResponse() did not preserve first result text: %#v", first.ResultText)
	}
}

func TestFormattedCSVOrdersResponsesAndSkipsErrors(t *testing.T) {
	responses := GeneralResponses{
		"b": {Dataview: &Dataview{Table: [][]string{{"b", "second"}}}},
		"a": {Dataview: &Dataview{Table: [][]string{{"a", "first"}}}},
		"x": {Err: errors.New("failed"), Dataview: &Dataview{Table: [][]string{{"x", "hidden"}}}},
	}
	var output bytes.Buffer
	if err := responses.Formatted(&output, "csv", []string{"name", "value"}, nil); err != nil {
		t.Fatalf("Formatted() error = %v", err)
	}
	got, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"name", "value"},
		{"a", "first"},
		{"b", "second"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CSV rows = %#v, want %#v", got, want)
	}
}

func TestFormattedJSONMergesValuesAndSkipsErrors(t *testing.T) {
	responses := GeneralResponses{
		"b": {Values: []any{map[string]string{"name": "second"}}},
		"a": {Value: map[string]string{"name": "first"}},
		"x": {Err: errors.New("failed"), Value: map[string]string{"name": "hidden"}},
	}
	var output bytes.Buffer
	if err := responses.Formatted(&output, "json", nil, nil); err != nil {
		t.Fatalf("Formatted() error = %v", err)
	}
	var got []map[string]string
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	want := []map[string]string{{"name": "first"}, {"name": "second"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON values = %#v, want %#v", got, want)
	}
}
