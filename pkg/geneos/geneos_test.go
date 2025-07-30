package geneos

import (
	"testing"
	"time"
)

func TestRemoveDuplicates(t *testing.T) {
	// Test with empty slice
	empty := []Attribute{}
	result := RemoveDuplicates(empty)
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(result))
	}

	// Test with no duplicates
	attrs := []Attribute{
		{Name: "attr1", Value: "value1"},
		{Name: "attr2", Value: "value2"},
		{Name: "attr3", Value: "value3"},
	}
	result = RemoveDuplicates(attrs)
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	// Test with duplicates
	attrsWithDups := []Attribute{
		{Name: "attr1", Value: "value1"},
		{Name: "attr2", Value: "value2"},
		{Name: "attr1", Value: "value1-dup"}, // duplicate name
		{Name: "attr3", Value: "value3"},
		{Name: "attr2", Value: "value2-dup"}, // duplicate name
	}
	result = RemoveDuplicates(attrsWithDups)
	if len(result) != 3 {
		t.Errorf("Expected 3 items after removing duplicates, got %d", len(result))
	}

	// Check that first occurrence is kept
	expectedNames := []string{"attr1", "attr2", "attr3"}
	for i, attr := range result {
		if attr.Name != expectedNames[i] {
			t.Errorf("Expected name '%s', got '%s'", expectedNames[i], attr.Name)
		}
	}
}

func TestAttributeGetKey(t *testing.T) {
	attr := Attribute{Name: "test-name", Value: "test-value"}
	key := attr.GetKey()
	if key != "test-name" {
		t.Errorf("Expected key 'test-name', got '%s'", key)
	}
}

func TestUnrollTypes(t *testing.T) {
	// Test with nil input
	result := UnrollTypes(nil)
	if result == nil {
		t.Error("Expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil input, got %d items", len(result))
	}

	// Test with empty types
	types := &Types{}
	result = UnrollTypes(types)
	if result == nil {
		t.Error("Expected non-nil map for empty types")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for empty types, got %d items", len(result))
	}

	// Test with simple types
	types = &Types{
		Types: []Type{
			{Name: "type1"},
			{Name: "type2"},
		},
	}
	result = UnrollTypes(types)
	if len(result) != 2 {
		t.Errorf("Expected 2 types, got %d", len(result))
	}
	if _, exists := result["type1"]; !exists {
		t.Error("Expected type1 to exist in result")
	}
	if _, exists := result["type2"]; !exists {
		t.Error("Expected type2 to exist in result")
	}
}

func TestUnrollSamplers(t *testing.T) {
	// Test with nil input
	result := UnrollSamplers(nil)
	if result == nil {
		t.Error("Expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil input, got %d items", len(result))
	}

	// Test with empty samplers
	samplers := &Samplers{}
	result = UnrollSamplers(samplers)
	if result == nil {
		t.Error("Expected non-nil map for empty samplers")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for empty samplers, got %d items", len(result))
	}

	// Test with simple samplers
	samplers = &Samplers{
		Samplers: []Sampler{
			{Name: "sampler1"},
			{Name: "sampler2"},
		},
	}
	result = UnrollSamplers(samplers)
	if len(result) != 2 {
		t.Errorf("Expected 2 samplers, got %d", len(result))
	}
	if _, exists := result["sampler1"]; !exists {
		t.Error("Expected sampler1 to exist in result")
	}
	if _, exists := result["sampler2"]; !exists {
		t.Error("Expected sampler2 to exist in result")
	}
}

func TestUnrollProcessDescriptors(t *testing.T) {
	// Test with nil input
	result := UnrollProcessDescriptors(nil)
	if result == nil {
		t.Error("Expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil input, got %d items", len(result))
	}

	// Test with empty process descriptors
	descriptors := &ProcessDescriptors{}
	result = UnrollProcessDescriptors(descriptors)
	if result == nil {
		t.Error("Expected non-nil map for empty process descriptors")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for empty process descriptors, got %d items", len(result))
	}

	// Test with simple process descriptors
	descriptors = &ProcessDescriptors{
		ProcessDescriptors: []ProcessDescriptor{
			{Name: "proc1"},
			{Name: "proc2"},
		},
	}
	result = UnrollProcessDescriptors(descriptors)
	if len(result) != 2 {
		t.Errorf("Expected 2 process descriptors, got %d", len(result))
	}
	if _, exists := result["proc1"]; !exists {
		t.Error("Expected proc1 to exist in result")
	}
	if _, exists := result["proc2"]; !exists {
		t.Error("Expected proc2 to exist in result")
	}
}

func TestUnrollRules(t *testing.T) {
	// Test with nil input
	result := UnrollRules(nil)
	if result == nil {
		t.Error("Expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for nil input, got %d items", len(result))
	}

	// Test with empty rules
	rules := &Rules{}
	result = UnrollRules(rules)
	if result == nil {
		t.Error("Expected non-nil map for empty rules")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map for empty rules, got %d items", len(result))
	}

	// Test with simple rules
	rules = &Rules{
		Rules: []Rule{
			{Name: "rule1"},
			{Name: "rule2"},
		},
	}
	result = UnrollRules(rules)
	if len(result) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(result))
	}
	if _, exists := result["rule1"]; !exists {
		t.Error("Expected rule1 to exist in result")
	}
	if _, exists := result["rule2"]; !exists {
		t.Error("Expected rule2 to exist in result")
	}
}

func TestExpandFileDates(t *testing.T) {
	// Test with no date patterns
	input := "test-file.txt"
	expectedTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	
	result, err := ExpandFileDates(input, expectedTime)
	if err != nil {
		t.Errorf("ExpandFileDates failed: %v", err)
	}
	if result != input {
		t.Errorf("Expected '%s', got '%s'", input, result)
	}

	// Test with Geneos date pattern
	input = "test-file-<today>.txt"
	result, err = ExpandFileDates(input, expectedTime)
	if err != nil {
		t.Errorf("ExpandFileDates failed: %v", err)
	}
	expected := "test-file-20230115.txt"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with Geneos date pattern with format
	input = "test-file-<today %Y%m%d>.txt"
	result, err = ExpandFileDates(input, expectedTime)
	if err != nil {
		t.Errorf("ExpandFileDates failed: %v", err)
	}
	expected = "test-file-20230115.txt"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with offset
	input = "test-file-<today +1>.txt"
	result, err = ExpandFileDates(input, expectedTime)
	if err != nil {
		t.Errorf("ExpandFileDates failed: %v", err)
	}
	expected = "test-file-20230116.txt"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestManagedEntityInfo(t *testing.T) {
	// Test ManagedEntityInfo struct
	info := ManagedEntityInfo{
		Attributes: []Attribute{
			{Name: "attr1", Value: "value1"},
			{Name: "attr2", Value: "value2"},
		},
		Vars: []Vars{
			{Name: "var1", String: "val1"},
			{Name: "var2", String: "val2"},
		},
	}

	if len(info.Attributes) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(info.Attributes))
	}
	if len(info.Vars) != 2 {
		t.Errorf("Expected 2 vars, got %d", len(info.Vars))
	}
}

func TestTypeStruct(t *testing.T) {
	// Test Type struct
	typ := Type{
		Name:     "test-type",
		Disabled: false,
		Vars: []Vars{
			{Name: "var1", String: "val1"},
		},
		Samplers: []SamplerRef{
			{Name: "sampler1", Disabled: false},
		},
	}

	if typ.Name != "test-type" {
		t.Errorf("Expected name 'test-type', got '%s'", typ.Name)
	}
	if typ.Disabled {
		t.Error("Expected Disabled to be false")
	}
	if len(typ.Vars) != 1 {
		t.Errorf("Expected 1 var, got %d", len(typ.Vars))
	}
	if len(typ.Samplers) != 1 {
		t.Errorf("Expected 1 sampler, got %d", len(typ.Samplers))
	}
}

func TestSamplerRef(t *testing.T) {
	// Test SamplerRef struct
	ref := SamplerRef{
		Name:     "test-sampler",
		Disabled: false,
	}

	if ref.Name != "test-sampler" {
		t.Errorf("Expected name 'test-sampler', got '%s'", ref.Name)
	}
	if ref.Disabled {
		t.Error("Expected Disabled to be false")
	}
}