package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"testing"
)

func TestMethodCall(t *testing.T) {
	// Test methodCall struct
	method := "testMethod"
	args := []interface{}{"arg1", 42, true}
	
	methodCall := &methodCall{
		Name: method,
		Params: methodParams{
			Params: args,
		},
	}

	if methodCall.Name != method {
		t.Errorf("Expected method name '%s', got '%s'", method, methodCall.Name)
	}

	if len(methodCall.Params.Params) != len(args) {
		t.Errorf("Expected %d params, got %d", len(args), len(methodCall.Params.Params))
	}
}

func TestMethodScalar(t *testing.T) {
	// Test methodScalar with string
	scalar := methodScalar{
		Scalar: methodString{Value: "test"},
	}

	if scalar.Scalar.(methodString).Value != "test" {
		t.Errorf("Expected scalar value 'test', got '%v'", scalar.Scalar)
	}

	// Test methodScalar with int
	scalar = methodScalar{
		Scalar: methodInt{Value: 42},
	}

	if scalar.Scalar.(methodInt).Value != 42 {
		t.Errorf("Expected scalar value 42, got %v", scalar.Scalar)
	}
}

func TestMethodArray(t *testing.T) {
	// Test methodArray
	array := methodArray{
		Array: methodArrayData{
			Data: []methodData{
				{Value: "item1"},
				{Value: "item2"},
			},
		},
	}

	arrayData := array.Array.(methodArrayData)
	if len(arrayData.Data) != 2 {
		t.Errorf("Expected 2 items in array, got %d", len(arrayData.Data))
	}
}

func TestMethodString(t *testing.T) {
	// Test methodString
	methodStr := methodString{Value: "test string"}

	if methodStr.Value != "test string" {
		t.Errorf("Expected value 'test string', got '%s'", methodStr.Value)
	}
}

func TestMethodInt(t *testing.T) {
	// Test methodInt
	methodInt := methodInt{Value: 123}

	if methodInt.Value != 123 {
		t.Errorf("Expected value 123, got %d", methodInt.Value)
	}
}

func TestMethodBool(t *testing.T) {
	// Test methodBool
	methodBool := methodBool{Value: 1}

	if methodBool.Value != 1 {
		t.Errorf("Expected value 1, got %d", methodBool.Value)
	}
}

func TestMethodDouble(t *testing.T) {
	// Test methodDouble
	methodDouble := methodDouble{Value: 3.14}

	if methodDouble.Value != 3.14 {
		t.Errorf("Expected value 3.14, got %f", methodDouble.Value)
	}
}

func TestMembers(t *testing.T) {
	// Test members struct
	member := members{
		Name:   "testName",
		Int:    42,
		String: "testString",
	}

	if member.Name != "testName" {
		t.Errorf("Expected name 'testName', got '%s'", member.Name)
	}

	if member.Int != 42 {
		t.Errorf("Expected int 42, got %d", member.Int)
	}

	if member.String != "testString" {
		t.Errorf("Expected string 'testString', got '%s'", member.String)
	}
}

func TestMethodResponse(t *testing.T) {
	// Test methodResponse struct
	response := methodResponse{
		Boolean:      true,
		String:       "test response",
		Int:          42,
		SliceStrings: []string{"item1", "item2"},
		Fault:        []members{},
	}

	if !response.Boolean {
		t.Error("Expected boolean to be true")
	}

	if response.String != "test response" {
		t.Errorf("Expected string 'test response', got '%s'", response.String)
	}

	if response.Int != 42 {
		t.Errorf("Expected int 42, got %d", response.Int)
	}

	if len(response.SliceStrings) != 2 {
		t.Errorf("Expected 2 strings, got %d", len(response.SliceStrings))
	}
}

func TestXMLMarshaling(t *testing.T) {
	// Test XML marshaling of methodCall
	methodCall := &methodCall{
		Name: "testMethod",
		Params: methodParams{
			Params: []interface{}{"arg1", 42},
		},
	}

	data, err := xml.Marshal(methodCall)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodCall
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Name != methodCall.Name {
		t.Errorf("Expected method name '%s', got '%s'", methodCall.Name, unmarshaled.Name)
	}
}

func TestMethodStringXML(t *testing.T) {
	// Test XML marshaling of methodString
	methodStr := methodString{Value: "test"}

	data, err := xml.Marshal(methodStr)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodString
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Value != methodStr.Value {
		t.Errorf("Expected value '%s', got '%s'", methodStr.Value, unmarshaled.Value)
	}
}

func TestMethodIntXML(t *testing.T) {
	// Test XML marshaling of methodInt
	methodInt := methodInt{Value: 123}

	data, err := xml.Marshal(methodInt)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodInt
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Value != methodInt.Value {
		t.Errorf("Expected value %d, got %d", methodInt.Value, unmarshaled.Value)
	}
}

func TestMethodBoolXML(t *testing.T) {
	// Test XML marshaling of methodBool
	methodBool := methodBool{Value: 1}

	data, err := xml.Marshal(methodBool)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodBool
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Value != methodBool.Value {
		t.Errorf("Expected value %d, got %d", methodBool.Value, unmarshaled.Value)
	}
}

func TestMethodDoubleXML(t *testing.T) {
	// Test XML marshaling of methodDouble
	methodDouble := methodDouble{Value: 3.14}

	data, err := xml.Marshal(methodDouble)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodDouble
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Value != methodDouble.Value {
		t.Errorf("Expected value %f, got %f", methodDouble.Value, unmarshaled.Value)
	}
}

func TestMethodArrayXML(t *testing.T) {
	// Test XML marshaling of methodArray
	array := methodArray{
		Array: methodArrayData{
			Data: []methodData{
				{Value: "item1"},
				{Value: "item2"},
			},
		},
	}

	data, err := xml.Marshal(array)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodArray
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}
}

func TestMethodResponseXML(t *testing.T) {
	// Test XML marshaling of methodResponse
	response := methodResponse{
		Boolean:      true,
		String:       "test",
		Int:          42,
		SliceStrings: []string{"item1", "item2"},
		Fault:        []members{},
	}

	data, err := xml.Marshal(response)
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("XML marshaling should return non-empty data")
	}

	// Test XML unmarshaling
	var unmarshaled methodResponse
	err = xml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("XML unmarshaling failed: %v", err)
	}

	if unmarshaled.Boolean != response.Boolean {
		t.Errorf("Expected boolean %v, got %v", response.Boolean, unmarshaled.Boolean)
	}

	if unmarshaled.String != response.String {
		t.Errorf("Expected string '%s', got '%s'", response.String, unmarshaled.String)
	}

	if unmarshaled.Int != response.Int {
		t.Errorf("Expected int %d, got %d", response.Int, unmarshaled.Int)
	}
}

func TestXMLEncoding(t *testing.T) {
	// Test that XML encoding produces valid XML
	methodCall := &methodCall{
		Name: "testMethod",
		Params: methodParams{
			Params: []interface{}{"arg1", 42, true},
		},
	}

	data, err := xml.MarshalIndent(methodCall, "", "  ")
	if err != nil {
		t.Errorf("XML marshaling failed: %v", err)
	}

	// Check that it starts with <?xml
	if !bytes.HasPrefix(data, []byte("<?xml")) {
		t.Error("XML output should start with <?xml")
	}

	// Check that it contains the method name
	if !bytes.Contains(data, []byte("testMethod")) {
		t.Error("XML output should contain the method name")
	}
}