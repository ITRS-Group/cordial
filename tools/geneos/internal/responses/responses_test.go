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

func TestMergeResponseKeepsAllOutput(t *testing.T) {
	first := &General{
		ResultText: []string{"moved from X to Y"},
		Audited:    true,
		Dataview: &Dataview{
			Columns:   []string{"name"},
			Table:     [][]string{{"first"}},
			Headlines: map[string]string{"a": "1", "b": "2"},
		},
	}
	second := &General{
		ResultText: []string{"updated"},
		Dataview: &Dataview{
			Columns:   []string{"ignored"},
			Table:     [][]string{{"second"}},
			Headlines: map[string]string{"b": "3"},
		},
	}

	got := MergeResponse(first, second)
	if !reflect.DeepEqual(got.ResultText, []string{"moved from X to Y", "updated"}) {
		t.Errorf("merged ResultText = %#v", got.ResultText)
	}
	if !got.Audited {
		t.Error("merged Audited = false when one response was audited")
	}
	if !reflect.DeepEqual(got.Dataview.Columns, []string{"name"}) {
		t.Errorf("merged Columns = %#v", got.Dataview.Columns)
	}
	if !reflect.DeepEqual(got.Dataview.Headlines, map[string]string{"a": "1", "b": "3"}) {
		t.Errorf("merged Headlines = %#v", got.Dataview.Headlines)
	}
	if !reflect.DeepEqual(first.ResultText, []string{"moved from X to Y"}) || len(first.Dataview.Headlines) != 2 {
		t.Error("MergeResponse() changed its first argument")
	}

	// a response without a dataview, or with no columns of its own, can be merged
	got = MergeResponse(&General{ResultText: []string{"only"}}, &General{Dataview: &Dataview{Columns: []string{"c"}}})
	if !reflect.DeepEqual(got.ResultText, []string{"only"}) || !reflect.DeepEqual(got.Dataview.Columns, []string{"c"}) {
		t.Errorf("merge with nil dataview = %#v, %#v", got.ResultText, got.Dataview)
	}
}

func TestGeneralReportWritesErrors(t *testing.T) {
	boom := errors.New("boom")

	for name, tt := range map[string]struct {
		err         error
		wantStdout  bool
		wantStderr  string
		ignoreError error
	}{
		"error":   {err: boom, wantStderr: "boom"},
		"ignored": {err: boom, ignoreError: boom, wantStdout: true},
		"none":    {wantStdout: true},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			options := []Option{StderrWriter(&stderr), PlainValue()}
			if tt.ignoreError != nil {
				options = append(options, IgnoreErr(tt.ignoreError))
			}
			General{Err: tt.err, Value: "value", Dataview: &Dataview{}}.Report(&stdout, options...)

			if got := stdout.Len() > 0; got != tt.wantStdout {
				t.Errorf("stdout = %q, want output = %v", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr == "" && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want nothing", stderr.String())
			}
			if tt.wantStderr != "" && !bytes.Contains(stderr.Bytes(), []byte(tt.wantStderr)) {
				t.Errorf("stderr = %q, want the error %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestReportSliceValueIsValidJSON(t *testing.T) {
	responses := GeneralResponses{
		"a": {Value: []any{map[string]int{"n": 1}, map[string]int{"n": 2}, map[string]int{"n": 3}}, Dataview: &Dataview{}},
		"b": {Value: map[string]int{"n": 4}, Dataview: &Dataview{}},
	}

	for _, indent := range []bool{false, true} {
		var out bytes.Buffer
		responses.Report(&out, StderrWriter(&bytes.Buffer{}), IndentJSON(indent))

		var got []map[string]int
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("indent=%v output is not valid JSON: %v\n%s", indent, err, out.String())
		}
		if want := []map[string]int{{"n": 1}, {"n": 2}, {"n": 3}, {"n": 4}}; !reflect.DeepEqual(got, want) {
			t.Errorf("indent=%v values = %v, want %v", indent, got, want)
		}
	}
}

func TestFormattedJSONWithNoData(t *testing.T) {
	var output bytes.Buffer
	if err := (GeneralResponses{}).Formatted(&output, "json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := bytes.TrimSpace(output.Bytes()); string(got) != "[]" {
		t.Errorf("JSON with no data = %q, want []", got)
	}
}
