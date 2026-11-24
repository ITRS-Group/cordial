package reporter

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"strconv"
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
	r.UpdateTable([]string{"name", "description"}, [][]string{{"gateway", "Demo, west\nnext\rrow\tC:\\temp\x00"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}

	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"name", "description"}, {"gateway", `Demo, west\nnext\rrow\tC:\\temp`}}
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
	r.AddHeadline("state", "first\rsecond\tthird\b\f\v\a\x00\x1b\x7f")
	r.UpdateTable([]string{"name", "detail"}, [][]string{{"gateway", "line,one\nC:\\temp"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	want := `name,detail
gateway,line one\nC:\\temp
<!>state,first\rsecond\tthird\b\f\v\a
`
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

func TestToolkitReporterNoQuotingAndHeadlineOrder(t *testing.T) {
	var output bytes.Buffer
	r, err := NewReporter("toolkit", &output)
	if err != nil {
		t.Fatal(err)
	}
	r.AddHeadline("zeta", "1")
	r.AddHeadline("alpha", "2")
	r.AddHeadline("zeta", "3") // replaces the value, keeps the position
	r.AddHeadlines(map[string]string{"m": "x", "b": "y", "k": "z"})
	r.UpdateTable([]string{"a b", "c"}, [][]string{{`say "hi"`, " lead"}, {"", "x"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	want := "a b,c\nsay \"hi\", lead\n,x\n<!>zeta,3\n<!>alpha,2\n<!>b,y\n<!>k,z\n<!>m,x\n"
	if output.String() != want {
		t.Fatalf("Toolkit output = %q, want %q", output.String(), want)
	}
}

// every reporter that scrambles must do so without changing the
// caller's table
func TestScrambleDoesNotModifyCallerRows(t *testing.T) {
	for _, format := range []string{"toolkit", "csv", "table", "column", "html", "markdown"} {
		t.Run(format, func(t *testing.T) {
			var output bytes.Buffer
			r, err := NewReporter(format, &output, Scramble(true))
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Prepare(Report{ScrambleColumns: []string{"secret"}}); err != nil {
				t.Fatal(err)
			}
			columns := []string{"name", "secret"}
			rows := [][]string{{"gateway", "hostname.example.com"}, {"short"}}
			r.UpdateTable(columns, rows)
			if err := r.Render(); err != nil {
				t.Fatal(err)
			}
			if want := [][]string{{"gateway", "hostname.example.com"}, {"short"}}; !reflect.DeepEqual(rows, want) {
				t.Errorf("caller rows modified: %v", rows)
			}
			if strings.Contains(output.String(), "hostname.example.com") {
				t.Errorf("value not scrambled in %q", output.String())
			}
			if !strings.Contains(output.String(), "gateway") {
				t.Errorf("unscrambled column missing in %q", output.String())
			}
		})
	}
}

func TestScrambleFuncAndColumnsOptions(t *testing.T) {
	var output bytes.Buffer
	r, _ := NewReporter("toolkit", &output, Scramble(true), ScrambleColumns([]string{"b"}),
		ScrambleFunc(func(s string) string { return "<" + s + ">" }))
	_ = r.Prepare(Report{ScrambleColumns: []string{"a"}})
	r.UpdateTable([]string{"a", "b", "c"}, [][]string{{"1", "2", "3"}})
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	if want := "a,b,c\n<1>,<2>,3\n"; output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestColumnReporterDoesNotRerenderStaleRows(t *testing.T) {
	var output bytes.Buffer
	r, _ := NewReporter("column", &output)
	r.UpdateTable([]string{"name"}, [][]string{{"first"}})
	if err := r.Prepare(Report{Title: "next"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(output.String(), "first"); got != 1 {
		t.Errorf("rows rendered %d times, want 1:\n%s", got, output.String())
	}
}

func TestFormattedReporter(t *testing.T) {
	t.Run("short rows do not panic", func(t *testing.T) {
		var output bytes.Buffer
		r, _ := NewReporter("csv", &output)
		_ = r.Prepare(Report{})
		r.UpdateTable([]string{"a", "b", "c"}, [][]string{{"1"}, {"2", "3", "4"}, {}})
		if err := r.Render(); err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(&output).ReadAll()
		if err != nil || len(rows) != 4 {
			t.Fatalf("rows = %v, err = %v", rows, err)
		}
	})

	t.Run("empty columns do not panic", func(t *testing.T) {
		r, _ := NewReporter("csv", io.Discard)
		r.UpdateTable(nil, [][]string{{"1"}})
		if err := r.Render(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("default order is the first column only", func(t *testing.T) {
		var output bytes.Buffer
		r, _ := NewReporter("csv", &output)
		_ = r.Prepare(Report{})
		r.UpdateTable([]string{"k", "v"}, [][]string{{"b", "1"}, {"a", "9"}, {"a", "2"}})
		_ = r.Render()
		if want := "k,v\na,9\na,2\nb,1\n"; output.String() != want {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	})

	t.Run("order by columns", func(t *testing.T) {
		var output bytes.Buffer
		r, _ := NewReporter("csv", &output, OrderByColumns(0, 1))
		_ = r.Prepare(Report{})
		r.UpdateTable([]string{"k", "v"}, [][]string{{"b", "1"}, {"a", "9"}, {"a", "2"}})
		_ = r.Render()
		if want := "k,v\na,2\na,9\nb,1\n"; output.String() != want {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	})

	t.Run("Writer option", func(t *testing.T) {
		var output bytes.Buffer
		r, err := NewReporter("csv", nil, Writer(&output))
		if err != nil {
			t.Fatal(err)
		}
		_ = r.Prepare(Report{})
		r.UpdateTable([]string{"k"}, [][]string{{"x"}})
		if err := r.Render(); err != nil {
			t.Fatal(err)
		}
		if want := "k\nx\n"; output.String() != want {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	})

	t.Run("AddHeadlines are rendered in sorted order", func(t *testing.T) {
		var output bytes.Buffer
		r, _ := NewReporter("table", &output)
		_ = r.Prepare(Report{Title: "T"})
		r.AddHeadline("first", "1")
		r.AddHeadlines(map[string]string{"c": "3", "b": "2"})
		r.UpdateTable([]string{"k"}, [][]string{{"x"}})
		_ = r.Render()
		out := output.String()
		i1, i2, i3 := strings.Index(out, "first"), strings.Index(out, " b "), strings.Index(out, " c ")
		if i1 < 0 || i2 < i1 || i3 < i2 {
			t.Errorf("headlines missing or out of order:\n%s", out)
		}
	})

	t.Run("html with no writer does not panic", func(t *testing.T) {
		var z bytes.Buffer
		r, err := NewReporter("html", nil, ZipWriter(zip.NewWriter(&z)))
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestFormattedReporterZipHasNoUnnamedEntry(t *testing.T) {
	var buf bytes.Buffer
	r, err := NewReporter("csv", nil, ZipWriter(zip.NewWriter(&buf)))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.csv", "b.csv"} {
		if err := r.Prepare(Report{FilePath: name}); err != nil {
			t.Fatal(err)
		}
		r.UpdateTable([]string{"k"}, [][]string{{"x"}})
	}
	if err := r.Render(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	if want := []string{"a.csv", "b.csv"}; !reflect.DeepEqual(names, want) {
		t.Errorf("zip entries = %q, want %q", names, want)
	}
}

func TestAPIReporterMissingEntityIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such thing", http.StatusInternalServerError)
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())

	r, err := NewReporter("api", nil, APIHostname(u.Hostname()), APIPort(port), APIEntity("e"), APISampler("s"))
	if err == nil || r != nil {
		t.Fatalf("NewReporter() = %v, %v; want a nil reporter and an error", r, err)
	}
}

func TestScrambleColumns(t *testing.T) {
	cols := []string{"a", "b"}
	tests := []struct {
		name  string
		scr   []string
		table [][]string
		want  [][]string
	}{
		{"none named", nil, [][]string{{"1", "2"}}, [][]string{{"1", "2"}}},
		{"unknown column", []string{"zz"}, [][]string{{"1", "2"}}, [][]string{{"1", "2"}}},
		{"short and empty rows", []string{"b"}, [][]string{{"1"}, {}, {"1", "2"}}, [][]string{{"1"}, {}, {"1", "*2*"}}},
		{"repeated column", []string{"a", "a"}, [][]string{{"1", "2"}}, [][]string{{"*1*", "2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := make([][]string, len(tt.table))
			for i, r := range tt.table {
				orig[i] = slices.Clone(r)
			}
			got := scrambleColumns(cols, tt.scr, tt.table, func(s string) string { return "*" + s + "*" })
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("scrambleColumns() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(tt.table, orig) {
				t.Errorf("input modified: %v", tt.table)
			}
		})
	}
}
