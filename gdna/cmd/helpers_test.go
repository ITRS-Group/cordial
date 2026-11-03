package cmd

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMatchReport(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		wantMatch     bool
		wantSubreport string
	}{
		{name: "literal", pattern: "gateway-status,licd-summary", wantMatch: true},
		{name: "glob", pattern: "gateway-*", wantMatch: true},
		{name: "matched subreport", pattern: "gateway-status:west,licd-summary", wantMatch: true, wantSubreport: "west"},
		{name: "suffix belongs to matched pattern", pattern: "other:stale,gateway-status", wantMatch: true},
		{name: "no match", pattern: "other,licd-*"},
	}
	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			gotMatch, gotSubreport := matchReport("gateway-status", test.pattern)
			if gotMatch != test.wantMatch || gotSubreport != test.wantSubreport {
				t.Errorf("matchReport() = (%t, %q), want (%t, %q)", gotMatch, gotSubreport, test.wantMatch, test.wantSubreport)
			}
		})
	}
}

func TestColOrNull(t *testing.T) {
	columns := map[string]int{"name": 0, "empty": 1}
	fields := []string{"Gateway A", ""}

	if got := colOrNull("name", columns, fields); !got.Valid || got.String != "Gateway A" {
		t.Errorf("colOrNull(name) = %#v, want valid Gateway A", got)
	}
	for _, column := range []string{"empty", "missing"} {
		if got := colOrNull(column, columns, fields); got.Valid {
			t.Errorf("colOrNull(%q) = %#v, want NULL", column, got)
		}
	}
}

func TestReadLicdReportSections(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "licd.csv")
	contents := "samplingStatus,status\n1,ok\x00Group,usage\ngroup-a,5\x00Req Number,details\n123,detail\x00"
	if err := os.WriteFile(filename, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	summary, tokenUsage, details, err := readLicdReport(filename)
	if err != nil {
		t.Fatalf("readLicdReport() error = %v", err)
	}
	for _, section := range []struct {
		name   string
		reader interface{ Read() ([]string, error) }
		want   [][]string
	}{
		{name: "summary", reader: summary, want: [][]string{{"samplingStatus", "status"}, {"1", "ok"}}},
		{name: "token usage", reader: tokenUsage, want: [][]string{{"Group", "usage"}, {"group-a", "5"}}},
		{name: "details", reader: details, want: [][]string{{"Req Number", "details"}, {"123", "detail"}}},
	} {
		t.Run(section.name, func(t *testing.T) {
			var got [][]string
			for {
				row, err := section.reader.Read()
				if err != nil {
					if err != io.EOF {
						t.Fatalf("Read() error = %v", err)
					}
					break
				}
				got = append(got, row)
			}
			if !reflect.DeepEqual(got, section.want) {
				t.Errorf("section rows = %#v, want %#v", got, section.want)
			}
		})
	}
}

func TestReadLicdReportRejectsMissingSection(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "incomplete.csv")
	if err := os.WriteFile(filename, []byte("samplingStatus\n1\x00Group\n2\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := readLicdReport(filename); err == nil {
		t.Fatal("readLicdReport() succeeded without all three sections")
	}
}

func TestColOrNullUsesSQLNullString(t *testing.T) {
	got := colOrNull("count", map[string]int{"count": 0}, []string{"0"})
	want := sql.NullString{String: "0", Valid: true}
	if got != want {
		t.Errorf("colOrNull(count) = %#v, want %#v", got, want)
	}
}

func TestEmailConfToString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "string", in: "alerts@example.com", want: "alerts@example.com"},
		{name: "string slice", in: []string{"a@example.com", "b@example.com"}, want: "a@example.com,b@example.com"},
		{name: "generic slice", in: []any{"a@example.com", 7}, want: "a@example.com,7"},
		{name: "unsupported", in: 42, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := emailConfToString(test.in); got != test.want {
				t.Errorf("emailConfToString() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReportLookupTableMasksIdentity(t *testing.T) {
	lookup := reportLookupTable("licd-summary", "licd", true)
	if lookup["hostname"] != "********" || lookup["username"] != "********" {
		t.Errorf("scrambled identity = (%q, %q), want masked values", lookup["hostname"], lookup["username"])
	}
	if lookup["report-name"] != "licd-summary" || lookup["report-group"] != "licd" {
		t.Errorf("report metadata = %#v", lookup)
	}
	if _, err := time.Parse(time.DateOnly, lookup["date"]); err != nil {
		t.Errorf("date value %q is not a date: %v", lookup["date"], err)
	}
	if _, err := time.Parse(time.TimeOnly, lookup["time"]); err != nil {
		t.Errorf("time value %q is not a time: %v", lookup["time"], err)
	}
}
