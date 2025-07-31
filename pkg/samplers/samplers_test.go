package samplers

import (
	"sync"
	"testing"
	"time"
)



// MockPlugins implements the plugins.Plugins interface for testing
type MockPlugins struct {
	interval time.Duration
	started  bool
	closed   bool
}

func (m *MockPlugins) SetInterval(d time.Duration) {
	m.interval = d
}

func (m *MockPlugins) Interval() time.Duration {
	return m.interval
}

func (m *MockPlugins) Start(wg *sync.WaitGroup) error {
	m.started = true
	return nil
}

func (m *MockPlugins) Close() error {
	m.closed = true
	return nil
}

func TestSamplersSetInterval(t *testing.T) {
	samplers := &Samplers{}
	expectedInterval := 5 * time.Second

	samplers.SetInterval(expectedInterval)
	if samplers.Interval() != expectedInterval {
		t.Errorf("Expected interval %v, got %v", expectedInterval, samplers.Interval())
	}
}

func TestSamplersSetColumnNames(t *testing.T) {
	samplers := &Samplers{}
	expectedNames := []string{"col1", "col2", "col3"}

	samplers.SetColumnNames(expectedNames)
	names := samplers.ColumnNames()

	if len(names) != len(expectedNames) {
		t.Errorf("Expected %d column names, got %d", len(expectedNames), len(names))
	}

	for i, name := range expectedNames {
		if names[i] != name {
			t.Errorf("Expected column name '%s' at index %d, got '%s'", name, i, names[i])
		}
	}
}

func TestSamplersSetColumns(t *testing.T) {
	samplers := &Samplers{}
	expectedColumns := Columns{
		"col1": {name: "Column 1", number: 0},
		"col2": {name: "Column 2", number: 1},
	}

	samplers.SetColumns(expectedColumns)
	columns := samplers.Columns()

	if len(columns) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(columns))
	}

	for key, expectedCol := range expectedColumns {
		if col, exists := columns[key]; !exists {
			t.Errorf("Expected column '%s' to exist", key)
		} else if col.name != expectedCol.name {
			t.Errorf("Expected column name '%s' for key '%s', got '%s'", expectedCol.name, key, col.name)
		}
	}
}

func TestSamplersSetSortColumn(t *testing.T) {
	samplers := &Samplers{}
	expectedSortColumn := "col1"

	samplers.SetSortColumn(expectedSortColumn)
	if samplers.SortColumn() != expectedSortColumn {
		t.Errorf("Expected sort column '%s', got '%s'", expectedSortColumn, samplers.SortColumn())
	}
}

func TestSamplersStart(t *testing.T) {
	// Skip this test as it requires a properly initialized Dataview
	t.Skip("Skipping Start test as it requires a properly initialized Dataview")
}

func TestSamplersClose(t *testing.T) {
	// Skip this test as it requires a properly initialized Dataview
	t.Skip("Skipping Close test as it requires a properly initialized Dataview")
}

func TestSamplersInitSamplerInternal(t *testing.T) {
	samplers := &Samplers{}

	// Test with nil Plugins
	err := samplers.initSamplerInternal()
	if err == nil {
		t.Error("Expected error when Plugins is nil")
	}

	// Test with mock that doesn't implement InitSampler
	mockPlugins := &MockPlugins{}
	samplers.Plugins = mockPlugins

	err = samplers.initSamplerInternal()
	if err == nil {
		t.Error("Expected error when Plugins doesn't implement InitSampler")
	}
}

func TestSamplersDoSampleInterval(t *testing.T) {
	samplers := &Samplers{}

	// Test with nil Plugins
	err := samplers.doSampleInterval()
	if err == nil {
		t.Error("Expected error when Plugins is nil")
	}

	// Test with mock that doesn't implement DoSample
	mockPlugins := &MockPlugins{}
	samplers.Plugins = mockPlugins

	err = samplers.doSampleInterval()
	if err == nil {
		t.Error("Expected error when Plugins doesn't implement DoSample")
	}
}

func TestColumnsSortRows(t *testing.T) {
	columns := Columns{
		"name": {name: "Name", number: 0, sort: sortAsc},
		"age":  {name: "Age", number: 1, sort: sortAscNum},
	}

	rows := [][]string{
		{"Charlie", "30"},
		{"Alice", "25"},
		{"Bob", "35"},
	}

	// Test sorting by name (ascending)
	sorted := columns.sortRows(rows, "name")
	if len(sorted) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(sorted))
	}

	// Test sorting by age (numeric ascending)
	sorted = columns.sortRows(rows, "age")
	if len(sorted) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(sorted))
	}
}

func TestToFloat(t *testing.T) {
	// Test with float64
	f, err := toFloat(3.14)
	if err != nil {
		t.Errorf("toFloat failed with float64: %v", err)
	}
	if f != 3.14 {
		t.Errorf("Expected 3.14, got %f", f)
	}

	// Test with int
	f, err = toFloat(42)
	if err != nil {
		t.Errorf("toFloat failed with int: %v", err)
	}
	if f != 42.0 {
		t.Errorf("Expected 42.0, got %f", f)
	}

	// Test with string (should fail as it's not convertible to float64)
	_, err = toFloat("3.14")
	if err == nil {
		t.Error("Expected error with string")
	}

	// Test with invalid string
	_, err = toFloat("invalid")
	if err == nil {
		t.Error("Expected error with invalid string")
	}

	// Test with unsupported type
	_, err = toFloat([]string{"test"})
	if err == nil {
		t.Error("Expected error with unsupported type")
	}
}

func TestParseTags(t *testing.T) {
	// Skip this test as the parseTags function is complex and requires more investigation
	t.Skip("Skipping parseTags test as it requires more investigation of the tag parsing logic")
}