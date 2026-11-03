package gwhub

import (
	"context"
	"net/http"
	"time"
)

// QueryEventRecordsRequest selects event records, optionally filtering by events,
// metrics, zone, and entity filters.
type QueryEventRecordsRequest struct {
	Skip    int      `json:"skip,omitempty"`
	Filter  Filter   `json:"filter"`
	Metrics []string `json:"metrics,omitempty"`
	ZoneID  string   `json:"zoneId,omitempty"`
	Limit   int      `json:"limit"`
	Events  []string `json:"events"`
}

// QueryEventRecordsResponse maps each requested event key to its returned records.
type QueryEventRecordsResponse map[string][]Event

// Event is one event record returned by GW Hub.
type Event struct {
	EntityID  int       `json:"entityId"`
	Timestamp time.Time `json:"timestamp"`
	Data      EventData `json:"data"`
	Metric    string    `json:"metric"`
	Type      string    `json:"type"`
}

// EventData contains the severity, activity state, and values for an event record.
type EventData struct {
	Severity string            `json:"severity"`
	Active   bool              `json:"active"`
	Value    map[string]string `json:"value"`
}

// QueryEventRecords returns event records matching request.
func (h *Hub) QueryEventRecords(ctx context.Context, request QueryEventRecordsRequest) (response QueryEventRecordsResponse, resp *http.Response, err error) {
	resp, err = h.Post(ctx, QueryEventRecordsEndpoint, request, &response)
	return
}
