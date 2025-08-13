package xpath

import (
	"encoding/json"
	"testing"
)

func TestNew(t *testing.T) {
	// Test with nil element
	xpath := New(nil)
	if xpath == nil {
		t.Fatal("New should not return nil")
	}

	// Test with empty struct
	xpath = New(&XPath{})
	if xpath == nil {
		t.Fatal("New should not return nil")
	}
}

func TestNewDataviewPath(t *testing.T) {
	xpath := NewDataviewPath("testDataview")
	if xpath == nil {
		t.Fatal("NewDataviewPath should not return nil")
	}

	if xpath.Dataview == nil {
		t.Error("Dataview should not be nil")
	}

	if xpath.Dataview.Name != "testDataview" {
		t.Errorf("Expected dataview name 'testDataview', got '%s'", xpath.Dataview.Name)
	}
}

func TestNewTableCellPath(t *testing.T) {
	xpath := NewTableCellPath("testRow", "testColumn")
	if xpath == nil {
		t.Fatal("NewTableCellPath should not return nil")
	}

	if xpath.Row == nil {
		t.Error("Row should not be nil")
	}

	if xpath.Column == nil {
		t.Error("Column should not be nil")
	}

	if xpath.Row.Name != "testRow" {
		t.Errorf("Expected row name 'testRow', got '%s'", xpath.Row.Name)
	}

	if xpath.Column.Name != "testColumn" {
		t.Errorf("Expected column name 'testColumn', got '%s'", xpath.Column.Name)
	}

	if !xpath.Rows {
		t.Error("Rows should be true for table cell path")
	}
}

func TestNewHeadlinePath(t *testing.T) {
	xpath := NewHeadlinePath("testHeadline")
	if xpath == nil {
		t.Fatal("NewHeadlinePath should not return nil")
	}

	if xpath.Headline == nil {
		t.Error("Headline should not be nil")
	}

	if xpath.Headline.Name != "testHeadline" {
		t.Errorf("Expected headline name 'testHeadline', got '%s'", xpath.Headline.Name)
	}

	if xpath.Rows {
		t.Error("Rows should be false for headline path")
	}
}

func TestXPathIsEmpty(t *testing.T) {
	// Test empty XPath
	emptyXPath := &XPath{}
	if !emptyXPath.IsEmpty() {
		t.Error("Empty XPath should return true for IsEmpty")
	}

	// Test non-empty XPath
	nonEmptyXPath := &XPath{
		Gateway: &Gateway{Name: "test"},
	}
	if nonEmptyXPath.IsEmpty() {
		t.Error("Non-empty XPath should return false for IsEmpty")
	}
}

func TestXPathSetGatewayName(t *testing.T) {
	xpath := &XPath{}
	gatewayName := "testGateway"

	xpath.SetGatewayName(gatewayName)

	if xpath.Gateway == nil {
		t.Error("Gateway should not be nil after SetGatewayName")
	}

	if xpath.Gateway.Name != gatewayName {
		t.Errorf("Expected gateway name '%s', got '%s'", gatewayName, xpath.Gateway.Name)
	}
}

func TestXPathTypeChecks(t *testing.T) {
	// Test IsTableCell
	tableCellPath := NewTableCellPath("row", "col")
	if !tableCellPath.IsTableCell() {
		t.Error("TableCell path should return true for IsTableCell")
	}

	// Test IsHeadline
	headlinePath := NewHeadlinePath("headline")
	if !headlinePath.IsHeadline() {
		t.Error("Headline path should return true for IsHeadline")
	}

	// Test IsDataview
	dataviewPath := NewDataviewPath("dataview")
	if !dataviewPath.IsDataview() {
		t.Error("Dataview path should return true for IsDataview")
	}

	// Test IsSampler
	samplerPath := &XPath{
		Sampler: &Sampler{Name: "test"},
	}
	if !samplerPath.IsSampler() {
		t.Error("Sampler path should return true for IsSampler")
	}

	// Test IsEntity
	entityPath := &XPath{
		Entity: &Entity{Name: "test"},
	}
	if !entityPath.IsEntity() {
		t.Error("Entity path should return true for IsEntity")
	}

	// Test IsProbe
	probePath := &XPath{
		Probe: &Probe{Name: "test"},
	}
	if !probePath.IsProbe() {
		t.Error("Probe path should return true for IsProbe")
	}

	// Test IsGateway
	gatewayPath := &XPath{
		Gateway: &Gateway{Name: "test"},
	}
	if !gatewayPath.IsGateway() {
		t.Error("Gateway path should return true for IsGateway")
	}
}

func TestXPathString(t *testing.T) {
	// Test simple gateway path
	gatewayPath := &XPath{
		Gateway: &Gateway{Name: "testGateway"},
	}
	expected := "/geneos/gateway/testGateway"
	result := gatewayPath.String()
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test full path
	fullPath := &XPath{
		Gateway:  &Gateway{Name: "testGateway"},
		Probe:    &Probe{Name: "testProbe"},
		Entity:   &Entity{Name: "testEntity"},
		Sampler:  &Sampler{Name: "testSampler"},
		Dataview: &Dataview{Name: "testDataview"},
		Rows:     true,
		Row:      &Row{Name: "testRow"},
		Column:   &Column{Name: "testColumn"},
	}
	expected = "/geneos/gateway/testGateway/probe/testProbe/managedEntity/testEntity/sampler/testSampler/dataview/testDataview/rows/row/testRow/cell/testColumn"
	result = fullPath.String()
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test headline path
	headlinePath := &XPath{
		Gateway:  &Gateway{Name: "testGateway"},
		Dataview: &Dataview{Name: "testDataview"},
		Headline: &Headline{Name: "testHeadline"},
	}
	expected = "/geneos/gateway/testGateway/dataview/testDataview/headlines/cell/testHeadline"
	result = headlinePath.String()
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestParse(t *testing.T) {
	// Test valid gateway path
	xpath, err := Parse("/geneos/gateway[(@name=\"testGateway\")]")
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	if xpath == nil {
		t.Fatal("Parse should not return nil")
	}
	if xpath.Gateway.Name != "testGateway" {
		t.Errorf("Expected gateway name 'testGateway', got '%s'", xpath.Gateway.Name)
	}

	// Test valid full path
	xpath, err = Parse("/geneos/gateway[(@name=\"testGateway\")]/directory/probe[(@name=\"testProbe\")]/managedEntity[(@name=\"testEntity\")]/sampler[(@name=\"testSampler\")]/dataview[(@name=\"testDataview\")]/rows/row[(@name=\"testRow\")]/cell[(@column=\"testColumn\")]")
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	if xpath == nil {
		t.Fatal("Parse should not return nil")
	}

	// Test invalid path (relative)
	_, err = Parse("gateway/testGateway")
	if err != ErrRelativePath {
		t.Errorf("Expected ErrRelativePath, got %v", err)
	}

	// Test invalid path (doesn't start with /geneos)
	_, err = Parse("/invalid/path")
	if err != ErrInvalidPath {
		t.Errorf("Expected ErrInvalidPath, got %v", err)
	}
}

func TestXPathMarshalJSON(t *testing.T) {
	xpath := &XPath{
		Gateway: &Gateway{Name: "testGateway"},
		Probe:   &Probe{Name: "testProbe"},
	}

	data, err := xpath.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("MarshalJSON should return non-empty data")
	}

	// Test that it's valid JSON
	var unmarshaled XPath
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Unmarshaling failed: %v", err)
	}
}

func TestXPathUnmarshalJSON(t *testing.T) {
	jsonData := `{"gateway":{"name":"testGateway"},"probe":{"name":"testProbe"}}`
	xpath := &XPath{}

	err := xpath.UnmarshalJSON([]byte(jsonData))
	if err != nil {
		t.Errorf("UnmarshalJSON failed: %v", err)
	}

	if xpath.Gateway == nil {
		t.Error("Gateway should not be nil after unmarshaling")
	}

	if xpath.Gateway.Name != "testGateway" {
		t.Errorf("Expected gateway name 'testGateway', got '%s'", xpath.Gateway.Name)
	}

	if xpath.Probe == nil {
		t.Error("Probe should not be nil after unmarshaling")
	}

	if xpath.Probe.Name != "testProbe" {
		t.Errorf("Expected probe name 'testProbe', got '%s'", xpath.Probe.Name)
	}
}

func TestXPathLookupValues(t *testing.T) {
	xpath := &XPath{
		Gateway: &Gateway{Name: "testGateway"},
		Entity: &Entity{
			Name: "testEntity",
			Attributes: map[string]string{
				"attr1": "value1",
				"attr2": "value2",
			},
		},
	}

	lookup := xpath.LookupValues()
	if lookup == nil {
		t.Fatal("LookupValues should not return nil")
	}

	// Check that gateway name is included
	if gatewayName, exists := lookup["gateway"]; !exists {
		t.Error("Gateway name should be in lookup values")
	} else if gatewayName != "testGateway" {
		t.Errorf("Expected gateway name 'testGateway', got '%s'", gatewayName)
	}

	// Check that entity name is included
	if entityName, exists := lookup["entity"]; !exists {
		t.Error("Entity name should be in lookup values")
	} else if entityName != "testEntity" {
		t.Errorf("Expected entity name 'testEntity', got '%s'", entityName)
	}

	// Check that attributes are included
	if attr1, exists := lookup["attr1"]; !exists {
		t.Error("Attribute 'attr1' should be in lookup values")
	} else if attr1 != "value1" {
		t.Errorf("Expected attribute value 'value1', got '%s'", attr1)
	}
}
