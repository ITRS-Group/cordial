package gwhub

import (
	"context"
	"net/http"
	"time"
)

type QueryEventRecordsRequest struct {
	Skip    int      `json:"skip,omitempty"`
	Filter  Filter   `json:"filter"`
	Metrics []string `json:"metrics,omitempty"`
	ZoneID  string   `json:"zoneId,omitempty"`
	Limit   int      `json:"limit"`
	Events  []string `json:"events"`
}

type QueryEventRecordsResponse map[string][]Event

type Event struct {
	EntityID  int       `json:"entityId"`
	Timestamp time.Time `json:"timestamp"`
	Data      EventData `json:"data"`
	Metric    string    `json:"metric"`
	Type      string    `json:"type"`
}

type EventData struct {
	Severity string            `json:"severity"`
	Active   bool              `json:"active"`
	Value    map[string]string `json:"value"`
}

// QueryEventRecords request
func (h *Hub) QueryEventRecords(ctx context.Context, request QueryEventRecordsRequest) (response QueryEventRecordsResponse, resp *http.Response, err error) {
	resp, err = h.Post(ctx, QueryEventRecordsEndpoint, request, &response)
	return
}
