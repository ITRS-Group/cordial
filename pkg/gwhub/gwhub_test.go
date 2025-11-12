package gwhub

import (
	"testing"
	"time"
)

func TestEndpoints(t *testing.T) {
	// Test that endpoints are defined
	endpoints := []string{
		PingEndpoint,
		MetricsQueryEndpoint,
		MetricsAggregationsEndpoint,
		QueryEventRecordsEndpoint,
		EntitiesEndpoint,
		EntitiesSummaryEndpoint,
		EntitiesMetricsEndpoint,
		EntitiesMetricsRowsEndpoint,
		EntitiesAttributesEndpoint,
	}
	if len(endpoints) == 0 {
		t.Error("Endpoints should not be empty")
	}
}

func TestEntities(t *testing.T) {
	// Test entity structure
	entity := Entity{
		ID:         123,
		Attributes: map[string]string{"prop1": "value1", "prop2": "42"},
		IsDeleted:  false,
	}
	if entity.ID != 123 {
		t.Errorf("Expected ID 123, got %d", entity.ID)
	}
	if entity.Attributes["prop1"] != "value1" {
		t.Errorf("Expected property 'value1', got '%v'", entity.Attributes["prop1"])
	}
	if entity.Attributes["prop2"] != "42" {
		t.Errorf("Expected property '42', got '%v'", entity.Attributes["prop2"])
	}
	if entity.IsDeleted {
		t.Errorf("Expected IsDeleted false, got true")
	}
}

func TestEvents(t *testing.T) {
	// Test event structure
	event := Event{
		EntityID:  123,
		Timestamp: time.Now(),
		Data: EventData{
			Severity: "warning",
			Active:   true,
			Value:    map[string]string{"prop1": "value1"},
		},
		Metric: "cpu",
		Type:   "alert",
	}

	if event.EntityID != 123 {
		t.Errorf("Expected EntityID 123, got %d", event.EntityID)
	}
	if event.Data.Severity != "warning" {
		t.Errorf("Expected severity 'warning', got '%s'", event.Data.Severity)
	}
	if !event.Data.Active {
		t.Errorf("Expected Active true, got false")
	}
	if event.Data.Value["prop1"] != "value1" {
		t.Errorf("Expected property 'value1', got '%v'", event.Data.Value["prop1"])
	}
	if event.Metric != "cpu" {
		t.Errorf("Expected metric 'cpu', got '%s'", event.Metric)
	}
	if event.Type != "alert" {
		t.Errorf("Expected type 'alert', got '%s'", event.Type)
	}
}

func TestMetrics(t *testing.T) {
	// Test metric structure
	metric := Metric{
		Identifier: "cpu",
		Include:    []string{"max", "min"},
	}

	if metric.Identifier != "cpu" {
		t.Errorf("Expected Identifier 'cpu', got '%s'", metric.Identifier)
	}
	if len(metric.Include) != 2 {
		t.Errorf("Expected 2 includes, got %d", len(metric.Include))
	}
	if metric.Include[0] != "max" || metric.Include[1] != "min" {
		t.Errorf("Expected includes ['max', 'min'], got %v", metric.Include)
	}
}

func TestDataStructures(t *testing.T) {
	// Test that all data structures can be created
	entity := Entity{}
	if entity.ID != 0 {
		t.Error("New entity should have ID 0")
	}
	if entity.Attributes == nil {
		t.Error("New entity should have non-nil Attributes map")
	}
	if entity.IsDeleted {
		t.Error("New entity should not be deleted")
	}

	// Only test valid data structures
}
