package samplers

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type testRow struct {
	Name   string `column:"Name"`
	Hidden string `column:"OMIT"`
	Count  int    `column:"Count,sort=-num,format=%03d"`
	Other  string
}

func TestColumnInfoOmitAndSortIndex(t *testing.T) {
	cols, names, sorting, err := Samplers{}.ColumnInfo(testRow{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Name", "Count", "Other"}; !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
	if sorting != "Count" {
		t.Errorf("sorting = %q", sorting)
	}
	// rendered rows have no OMIT column so Count is the 2nd cell
	if cols["Count"].number != 1 || cols["Hidden"].number != -1 || cols["Other"].number != 2 {
		t.Errorf("numbers = %d %d %d", cols["Count"].number, cols["Hidden"].number, cols["Other"].number)
	}
}

func TestRowsAreSortedByRenderedColumn(t *testing.T) {
	s := &Samplers{}
	cols, names, sorting, err := s.ColumnInfo(testRow{})
	if err != nil {
		t.Fatal(err)
	}
	s.SetColumns(cols)
	s.SetColumnNames(names)
	s.SetSortColumn(sorting)

	rows, err := s.RowsFromMap(map[string]testRow{
		"a": {Name: "a", Hidden: "x", Count: 5, Other: "o"},
		"b": {Name: "b", Hidden: "y", Count: 20, Other: "o"},
		"c": {Name: "c", Hidden: "z", Count: 7, Other: "o"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"b", "020", "o"}, {"c", "007", "o"}, {"a", "005", "o"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}

	// pointers to structs in slices are rendered, and errors are returned
	rows, err = s.RowsFromSlice([]*testRow{{Name: "a", Count: 1}})
	if err != nil || len(rows) != 1 || rows[0][0] != "a" {
		t.Errorf("rows = %v, err = %v", rows, err)
	}
	if _, err := s.RowsFromMap(map[string]int{"a": 1}); err == nil {
		t.Error("non-struct rows should fail")
	}
	if _, err := s.RowsFromMapDelta(map[string]testRow{"a": {}}, map[string]testRow{}, 0); err == nil {
		t.Error("a key missing from the old data should fail, not panic")
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		tag     string
		name    string
		sort    sortType
		format  string
		wantErr bool
	}{
		{tag: "Col", name: "Col", format: "%v"},
		{tag: "sort=", name: "F", sort: sortAsc, format: "%v"},
		{tag: "Col,sort=-", name: "Col", sort: sortDesc, format: "%v"},
		{tag: "Col,sort=-num,format=%.2f", name: "Col", sort: sortDescNum, format: "%.2f"},
		{tag: ",sort=num", name: "F", sort: sortAscNum, format: "%v"},
		{tag: "A,B", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			c, err := parseTags("F", tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if c.name != tt.name || c.sort != tt.sort || c.format != tt.format {
				t.Errorf("got name=%q sort=%v format=%q", c.name, c.sort, c.format)
			}
		})
	}
}

func TestSortRowsShortRowsDoNotPanic(t *testing.T) {
	c := Columns{"X": {number: 5, sort: sortAscNum}, "Y": {number: -1}}
	rows := [][]string{{"b"}, {"a", "x"}}
	c.sortRows(rows, "X")
	c.sortRows(rows, "Y")
}

// a zero interval would otherwise panic in time.NewTicker
func TestStartRejectsNoInterval(t *testing.T) {
	var wg sync.WaitGroup
	for _, interval := range []time.Duration{0, -time.Second} {
		s := &Samplers{}
		s.SetInterval(interval)
		if err := s.Start(&wg); err == nil || !strings.Contains(err.Error(), "interval") {
			t.Errorf("Start() with interval %s = %v", interval, err)
		}
	}
	// and with no dataview it fails rather than panics
	s := &Samplers{}
	s.SetInterval(time.Second)
	if err := s.Start(&wg); err == nil {
		t.Error("Start() with no dataview succeeded")
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close() with no dataview = %v", err)
	}
	wg.Wait()
}
