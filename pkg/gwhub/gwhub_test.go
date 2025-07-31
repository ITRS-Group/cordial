package gwhub

import (
	"testing"
)

func TestEndpoints(t *testing.T) {
	// Test that endpoints are defined
	if len(endpoints) == 0 {
		t.Error("Endpoints should not be empty")
	}

	// Test specific endpoints
	expectedEndpoints := []string{
		"entities",
		"events",
		"metrics",
		"schemas",
	}

	for _, endpoint := range expectedEndpoints {
		found := false
		for _, ep := range endpoints {
			if ep == endpoint {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected endpoint '%s' not found", endpoint)
		}
	}
}

func TestEntities(t *testing.T) {
	// Test entity structure
	entity := Entity{
		ID:          "test-entity-id",
		Name:        "Test Entity",
		Type:        "test-type",
		Description: "Test entity description",
		Status:      "active",
		Properties: map[string]interface{}{
			"prop1": "value1",
			"prop2": 42,
		},
	}

	if entity.ID != "test-entity-id" {
		t.Errorf("Expected ID 'test-entity-id', got '%s'", entity.ID)
	}

	if entity.Name != "Test Entity" {
		t.Errorf("Expected name 'Test Entity', got '%s'", entity.Name)
	}

	if entity.Type != "test-type" {
		t.Errorf("Expected type 'test-type', got '%s'", entity.Type)
	}

	if entity.Description != "Test entity description" {
		t.Errorf("Expected description 'Test entity description', got '%s'", entity.Description)
	}

	if entity.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", entity.Status)
	}

	if len(entity.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(entity.Properties))
	}

	if entity.Properties["prop1"] != "value1" {
		t.Errorf("Expected property 'value1', got '%v'", entity.Properties["prop1"])
	}

	if entity.Properties["prop2"] != 42 {
		t.Errorf("Expected property 42, got '%v'", entity.Properties["prop2"])
	}
}

func TestEvents(t *testing.T) {
	// Test event structure
	event := Event{
		ID:          "test-event-id",
		Type:        "test-event-type",
		Timestamp:   "2024-01-01T00:00:00Z",
		Severity:    "warning",
		Message:     "Test event message",
		EntityID:    "test-entity-id",
		Source:      "test-source",
		Properties: map[string]interface{}{
			"prop1": "value1",
		},
	}

	if event.ID != "test-event-id" {
		t.Errorf("Expected ID 'test-event-id', got '%s'", event.ID)
	}

	if event.Type != "test-event-type" {
		t.Errorf("Expected type 'test-event-type', got '%s'", event.Type)
	}

	if event.Timestamp != "2024-01-01T00:00:00Z" {
		t.Errorf("Expected timestamp '2024-01-01T00:00:00Z', got '%s'", event.Timestamp)
	}

	if event.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", event.Severity)
	}

	if event.Message != "Test event message" {
		t.Errorf("Expected message 'Test event message', got '%s'", event.Message)
	}

	if event.EntityID != "test-entity-id" {
		t.Errorf("Expected entity ID 'test-entity-id', got '%s'", event.EntityID)
	}

	if event.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got '%s'", event.Source)
	}

	if len(event.Properties) != 1 {
		t.Errorf("Expected 1 property, got %d", len(event.Properties))
	}
}

func TestMetrics(t *testing.T) {
	// Test metric structure
	metric := Metric{
		ID:          "test-metric-id",
		Name:        "Test Metric",
		Type:        "gauge",
		Unit:        "count",
		Description: "Test metric description",
		EntityID:    "test-entity-id",
		Value:       42.5,
		Timestamp:   "2024-01-01T00:00:00Z",
		Tags: map[string]string{
			"tag1": "value1",
			"tag2": "value2",
		},
	}

	if metric.ID != "test-metric-id" {
		t.Errorf("Expected ID 'test-metric-id', got '%s'", metric.ID)
	}

	if metric.Name != "Test Metric" {
		t.Errorf("Expected name 'Test Metric', got '%s'", metric.Name)
	}

	if metric.Type != "gauge" {
		t.Errorf("Expected type 'gauge', got '%s'", metric.Type)
	}

	if metric.Unit != "count" {
		t.Errorf("Expected unit 'count', got '%s'", metric.Unit)
	}

	if metric.Description != "Test metric description" {
		t.Errorf("Expected description 'Test metric description', got '%s'", metric.Description)
	}

	if metric.EntityID != "test-entity-id" {
		t.Errorf("Expected entity ID 'test-entity-id', got '%s'", metric.EntityID)
	}

	if metric.Value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", metric.Value)
	}

	if metric.Timestamp != "2024-01-01T00:00:00Z" {
		t.Errorf("Expected timestamp '2024-01-01T00:00:00Z', got '%s'", metric.Timestamp)
	}

	if len(metric.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metric.Tags))
	}

	if metric.Tags["tag1"] != "value1" {
		t.Errorf("Expected tag 'value1', got '%s'", metric.Tags["tag1"])
	}

	if metric.Tags["tag2"] != "value2" {
		t.Errorf("Expected tag 'value2', got '%s'", metric.Tags["tag2"])
	}
}

func TestSchemas(t *testing.T) {
	// Test schema structure
	schema := Schema{
		ID:          "test-schema-id",
		Name:        "Test Schema",
		Version:     "1.0.0",
		Description: "Test schema description",
		Type:        "object",
		Properties: map[string]SchemaProperty{
			"prop1": {
				Type:        "string",
				Description: "Test property",
				Required:    true,
			},
			"prop2": {
				Type:        "integer",
				Description: "Test number property",
				Required:    false,
			},
		},
		Required: []string{"prop1"},
	}

	if schema.ID != "test-schema-id" {
		t.Errorf("Expected ID 'test-schema-id', got '%s'", schema.ID)
	}

	if schema.Name != "Test Schema" {
		t.Errorf("Expected name 'Test Schema', got '%s'", schema.Name)
	}

	if schema.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", schema.Version)
	}

	if schema.Description != "Test schema description" {
		t.Errorf("Expected description 'Test schema description', got '%s'", schema.Description)
	}

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if len(schema.Properties) != 2 {
		t.Errorf("Expected 2 properties, got %d", len(schema.Properties))
	}

	if len(schema.Required) != 1 {
		t.Errorf("Expected 1 required property, got %d", len(schema.Required))
	}

	if schema.Required[0] != "prop1" {
		t.Errorf("Expected required property 'prop1', got '%s'", schema.Required[0])
	}

	// Test property
	prop := schema.Properties["prop1"]
	if prop.Type != "string" {
		t.Errorf("Expected property type 'string', got '%s'", prop.Type)
	}

	if prop.Description != "Test property" {
		t.Errorf("Expected property description 'Test property', got '%s'", prop.Description)
	}

	if !prop.Required {
		t.Error("Property should be required")
	}
}

func TestSchemaProperty(t *testing.T) {
	// Test SchemaProperty structure
	property := SchemaProperty{
		Type:        "string",
		Description: "Test property description",
		Required:    true,
		Default:     "default value",
		Enum:        []string{"option1", "option2", "option3"},
	}

	if property.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", property.Type)
	}

	if property.Description != "Test property description" {
		t.Errorf("Expected description 'Test property description', got '%s'", property.Description)
	}

	if !property.Required {
		t.Error("Property should be required")
	}

	if property.Default != "default value" {
		t.Errorf("Expected default 'default value', got '%s'", property.Default)
	}

	if len(property.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(property.Enum))
	}

	if property.Enum[0] != "option1" {
		t.Errorf("Expected enum value 'option1', got '%s'", property.Enum[0])
	}
}

func TestGWHub(t *testing.T) {
	// Test GWHub structure
	hub := GWHub{
		ID:          "test-hub-id",
		Name:        "Test Hub",
		Description: "Test hub description",
		Version:     "1.0.0",
		Status:      "active",
		URL:         "https://test-hub.example.com",
		APIKey:      "test-api-key",
		Config: map[string]interface{}{
			"setting1": "value1",
			"setting2": 42,
		},
	}

	if hub.ID != "test-hub-id" {
		t.Errorf("Expected ID 'test-hub-id', got '%s'", hub.ID)
	}

	if hub.Name != "Test Hub" {
		t.Errorf("Expected name 'Test Hub', got '%s'", hub.Name)
	}

	if hub.Description != "Test hub description" {
		t.Errorf("Expected description 'Test hub description', got '%s'", hub.Description)
	}

	if hub.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", hub.Version)
	}

	if hub.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", hub.Status)
	}

	if hub.URL != "https://test-hub.example.com" {
		t.Errorf("Expected URL 'https://test-hub.example.com', got '%s'", hub.URL)
	}

	if hub.APIKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", hub.APIKey)
	}

	if len(hub.Config) != 2 {
		t.Errorf("Expected 2 config settings, got %d", len(hub.Config))
	}

	if hub.Config["setting1"] != "value1" {
		t.Errorf("Expected config setting 'value1', got '%v'", hub.Config["setting1"])
	}

	if hub.Config["setting2"] != 42 {
		t.Errorf("Expected config setting 42, got '%v'", hub.Config["setting2"])
	}
}

func TestDataStructures(t *testing.T) {
	// Test that all data structures can be created
	entity := Entity{}
	if entity.ID != "" {
		t.Error("New entity should have empty ID")
	}

	event := Event{}
	if event.ID != "" {
		t.Error("New event should have empty ID")
	}

	metric := Metric{}
	if metric.ID != "" {
		t.Error("New metric should have empty ID")
	}

	schema := Schema{}
	if schema.ID != "" {
		t.Error("New schema should have empty ID")
	}

	property := SchemaProperty{}
	if property.Type != "" {
		t.Error("New property should have empty type")
	}

	hub := GWHub{}
	if hub.ID != "" {
		t.Error("New hub should have empty ID")
	}
}

func TestValidation(t *testing.T) {
	// Test entity validation
	entity := Entity{
		ID:   "test-id",
		Name: "Test Entity",
		Type: "test-type",
	}

	if entity.ID == "" {
		t.Error("Entity ID should not be empty")
	}

	if entity.Name == "" {
		t.Error("Entity name should not be empty")
	}

	if entity.Type == "" {
		t.Error("Entity type should not be empty")
	}

	// Test event validation
	event := Event{
		ID:        "test-event-id",
		Type:      "test-event-type",
		Timestamp: "2024-01-01T00:00:00Z",
		Severity:  "info",
		Message:   "Test message",
	}

	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}

	if event.Type == "" {
		t.Error("Event type should not be empty")
	}

	if event.Timestamp == "" {
		t.Error("Event timestamp should not be empty")
	}

	if event.Severity == "" {
		t.Error("Event severity should not be empty")
	}

	if event.Message == "" {
		t.Error("Event message should not be empty")
	}

	// Test metric validation
	metric := Metric{
		ID:    "test-metric-id",
		Name:  "Test Metric",
		Type:  "counter",
		Unit:  "count",
		Value: 100.0,
	}

	if metric.ID == "" {
		t.Error("Metric ID should not be empty")
	}

	if metric.Name == "" {
		t.Error("Metric name should not be empty")
	}

	if metric.Type == "" {
		t.Error("Metric type should not be empty")
	}

	if metric.Unit == "" {
		t.Error("Metric unit should not be empty")
	}
}