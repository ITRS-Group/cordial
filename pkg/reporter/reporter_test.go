package reporter

import (
	"bytes"
	"testing"
)

// MockReporter implements the Reporter interface for testing
type MockReporter struct {
	prepared     bool
	headlines    []string
	tableData    [][]string
	removed      bool
	rendered     bool
	closed       bool
	scrambleFunc func(in string) string
}

func (m *MockReporter) Prepare(report Report) error {
	m.prepared = true
	return nil
}

func (m *MockReporter) AddHeadline(name, value string) {
	m.headlines = append(m.headlines, name+":"+value)
}

func (m *MockReporter) UpdateTable(headings []string, rows [][]string) {
	m.tableData = append([][]string{headings}, rows...)
}

func (m *MockReporter) Remove(report Report) error {
	m.removed = true
	return nil
}

func (m *MockReporter) Render() {
	m.rendered = true
}

func (m *MockReporter) Close() {
	m.closed = true
}

func TestReporterInterface(t *testing.T) {
	mock := &MockReporter{}

	// Test Prepare
	report := Report{Name: "test", Title: "Test Report"}
	err := mock.Prepare(report)
	if err != nil {
		t.Errorf("Prepare failed: %v", err)
	}
	if !mock.prepared {
		t.Error("Prepare should set prepared flag")
	}

	// Test AddHeadline
	mock.AddHeadline("testHeadline", "testValue")
	if len(mock.headlines) != 1 {
		t.Errorf("Expected 1 headline, got %d", len(mock.headlines))
	}
	if mock.headlines[0] != "testHeadline:testValue" {
		t.Errorf("Expected 'testHeadline:testValue', got '%s'", mock.headlines[0])
	}

	// Test UpdateTable
	headings := []string{"Col1", "Col2", "Col3"}
	rows := [][]string{
		{"row1col1", "row1col2", "row1col3"},
		{"row2col1", "row2col2", "row2col3"},
	}
	mock.UpdateTable(headings, rows)
	if len(mock.tableData) != 3 { // headings + 2 rows
		t.Errorf("Expected 3 rows in table data, got %d", len(mock.tableData))
	}

	// Test Remove
	err = mock.Remove(report)
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}
	if !mock.removed {
		t.Error("Remove should set removed flag")
	}

	// Test Render
	mock.Render()
	if !mock.rendered {
		t.Error("Render should set rendered flag")
	}

	// Test Close
	mock.Close()
	if !mock.closed {
		t.Error("Close should set closed flag")
	}
}

func TestReportStruct(t *testing.T) {
	report := Report{
		Name:            "testReport",
		Title:           "Test Report",
		Columns:         []string{"col1", "col2", "col3"},
		ScrambleColumns: []string{"col1"},
	}

	// Test basic fields
	if report.Name != "testReport" {
		t.Errorf("Expected name 'testReport', got '%s'", report.Name)
	}

	if report.Title != "Test Report" {
		t.Errorf("Expected title 'Test Report', got '%s'", report.Title)
	}

	if len(report.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(report.Columns))
	}

	if len(report.ScrambleColumns) != 1 {
		t.Errorf("Expected 1 scramble column, got %d", len(report.ScrambleColumns))
	}

	// Test Dataview settings
	enable := true
	report.Dataview.Group = "testGroup"
	report.Dataview.Enable = &enable

	if report.Dataview.Group != "testGroup" {
		t.Errorf("Expected dataview group 'testGroup', got '%s'", report.Dataview.Group)
	}

	if report.Dataview.Enable == nil || !*report.Dataview.Enable {
		t.Error("Expected dataview enable to be true")
	}

	// Test XLSX settings
	report.XLSX.FreezeColumn = "B"
	report.XLSX.Enable = &enable

	if report.XLSX.FreezeColumn != "B" {
		t.Errorf("Expected freeze column 'B', got '%s'", report.XLSX.FreezeColumn)
	}

	if report.XLSX.Enable == nil || !*report.XLSX.Enable {
		t.Error("Expected xlsx enable to be true")
	}
}

func TestReporterOptions(t *testing.T) {
	// Test default options
	options := evalReporterOptions()
	if options.scrambleFunc == nil {
		t.Error("Default scramble function should not be nil")
	}

	if len(options.scrambleColumns) != 0 {
		t.Errorf("Expected 0 scramble columns, got %d", len(options.scrambleColumns))
	}

	// Test custom scramble function
	customScramble := func(in string) string {
		return "scrambled:" + in
	}
	options = evalReporterOptions(ScrambleFunc(customScramble))
	if options.scrambleFunc == nil {
		t.Error("Custom scramble function should not be nil")
	}

	// Test scramble columns
	columns := []string{"col1", "col2"}
	options = evalReporterOptions(ScrambleColumns(columns))
	if len(options.scrambleColumns) != 2 {
		t.Errorf("Expected 2 scramble columns, got %d", len(options.scrambleColumns))
	}

	// Test multiple options
	options = evalReporterOptions(
		ScrambleFunc(customScramble),
		ScrambleColumns(columns),
	)
	if options.scrambleFunc == nil {
		t.Error("Custom scramble function should not be nil")
	}
	if len(options.scrambleColumns) != 2 {
		t.Errorf("Expected 2 scramble columns, got %d", len(options.scrambleColumns))
	}
}

func TestScrambleFunc(t *testing.T) {
	// Test default scramble function
	result := scrambleWords("test")
	if result == "test" {
		t.Error("Scramble function should modify the input")
	}

	// Test with empty string
	result = scrambleWords("")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	// Test with single character
	result = scrambleWords("a")
	if result != "a" {
		t.Errorf("Expected 'a', got '%s'", result)
	}

	// Test with multiple words
	result = scrambleWords("hello world")
	if result == "hello world" {
		t.Error("Scramble function should modify multiple words")
	}
}

func TestScrambleColumns(t *testing.T) {
	// Test ScrambleColumns option
	columns := []string{"col1", "col2", "col3"}
	option := ScrambleColumns(columns)
	
	options := &reporterOptions{}
	option(options)
	
	if len(options.scrambleColumns) != 3 {
		t.Errorf("Expected 3 scramble columns, got %d", len(options.scrambleColumns))
	}
	
	for i, col := range columns {
		if options.scrambleColumns[i] != col {
			t.Errorf("Expected column '%s' at index %d, got '%s'", col, i, options.scrambleColumns[i])
		}
	}
}

func TestNewReporter(t *testing.T) {
	// Test with unsupported type
	writer := &bytes.Buffer{}
	_, err := NewReporter("unsupported", writer)
	if err == nil {
		t.Error("Expected error with unsupported reporter type")
	}

	// Test with nil writer
	_, err = NewReporter("csv", nil)
	if err == nil {
		t.Error("Expected error with nil writer")
	}
}

func TestReporterCommon(t *testing.T) {
	common := ReporterCommon{
		scrambleFunc: func(in string) string {
			return "scrambled:" + in
		},
	}

	if common.scrambleFunc == nil {
		t.Error("Scramble function should not be nil")
	}

	// Test scramble function
	result := common.scrambleFunc("test")
	if result != "scrambled:test" {
		t.Errorf("Expected 'scrambled:test', got '%s'", result)
	}
}

func TestConditionalFormat(t *testing.T) {
	// Test ConditionalFormat struct
	format := ConditionalFormat{
		Column: "testColumn",
		Rule:   "testRule",
	}

	if format.Column != "testColumn" {
		t.Errorf("Expected column 'testColumn', got '%s'", format.Column)
	}

	if format.Rule != "testRule" {
		t.Errorf("Expected rule 'testRule', got '%s'", format.Rule)
	}
}

func TestReporterIntegration(t *testing.T) {
	mock := &MockReporter{}

	// Test full workflow
	report := Report{
		Name:  "integrationTest",
		Title: "Integration Test Report",
		Columns: []string{"Name", "Value", "Status"},
	}

	// Prepare
	err := mock.Prepare(report)
	if err != nil {
		t.Errorf("Prepare failed: %v", err)
	}

	// Add headlines
	mock.AddHeadline("Report", "Integration Test")
	mock.AddHeadline("Generated", "2024-01-01")

	if len(mock.headlines) != 2 {
		t.Errorf("Expected 2 headlines, got %d", len(mock.headlines))
	}

	// Update table
	headings := []string{"Name", "Value", "Status"}
	rows := [][]string{
		{"Item1", "100", "OK"},
		{"Item2", "200", "WARNING"},
		{"Item3", "300", "ERROR"},
	}

	mock.UpdateTable(headings, rows)
	if len(mock.tableData) != 4 { // headings + 3 rows
		t.Errorf("Expected 4 rows in table data, got %d", len(mock.tableData))
	}

	// Render
	mock.Render()
	if !mock.rendered {
		t.Error("Render should set rendered flag")
	}

	// Close
	mock.Close()
	if !mock.closed {
		t.Error("Close should set closed flag")
	}
}