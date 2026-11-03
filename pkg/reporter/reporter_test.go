package reporter

import (
	"bytes"
	"encoding/csv"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestNewReporterRejectsUnknownFormat(t *testing.T) {
	if _, err := NewReporter("unknown", io.Discard); err == nil {
		t.Fatal("NewReporter accepted an unknown format")
	}
}

func TestCSVReporterRendersRows(t *testing.T) {
	var output bytes.Buffer
	r, err := NewReporter("csv", &output)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Prepare(Report{}); err != nil {
		t.Fatal(err)
	}
	r.UpdateTable([]string{"name", "description"}, [][]string{{"gateway", "Demo, west"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}

	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"name", "description"}, {"gateway", "Demo, west"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("CSV rows = %#v, want %#v", rows, want)
	}
}

func TestToolkitReporterEscapesUnsupportedCharacters(t *testing.T) {
	var output bytes.Buffer
	r, err := NewReporter("toolkit", &output)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Prepare(Report{}); err != nil {
		t.Fatal(err)
	}
	r.AddHeadline("state", "OK")
	r.UpdateTable([]string{"name", "detail"}, [][]string{{"gateway", "line,one\nline two"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	want := "name,detail\ngateway,line one\\nline two\n<!>state,OK\n"
	if output.String() != want {
		t.Fatalf("Toolkit output = %q, want %q", output.String(), want)
	}
}

func TestColumnReporterRendersAlignedRows(t *testing.T) {
	var output bytes.Buffer
	r, err := NewReporter("column", &output)
	if err != nil {
		t.Fatal(err)
	}
	r.UpdateTable([]string{"name", "state"}, [][]string{{"gateway", "OK"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name", "state", "gateway", "OK"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("column output %q does not contain %q", output.String(), want)
		}
	}
}
