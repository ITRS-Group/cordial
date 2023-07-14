package gwhub

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type MetricsQueryRequest struct {
	Grouping     []string  `json:"grouping,omitempty"`
	Filter       Filter    `json:"filter"`
	Aggregations []string  `json:"aggregations"`
	Metrics      []Metric  `json:"metrics"`
	ZoneID       string    `json:"zoneId,omitempty"`
	Bucketing    Bucketing `json:"bucketing,omitempty"`
}

type MetricsQueryResponse struct {
	Schema interface{} `json:"schema"`
	Data   interface{} `json:"data"`
}

type Filter struct {
	Entities    string       `json:"entities"`
	Range       TimeRange    `json:"range"`
	TimeWindows []TimeWindow `json:"timeWindows"`
}

type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// MarshalJSON is needed as Hub only accepts short form ISO times
func (t TimeRange) MarshalJSON() ([]byte, error) {
	timerange := struct {
		From string `json:"from"`
		To   string `json:"to"`
	}{
		From: t.From.Format(time.RFC3339),
		To:   t.To.Format(time.RFC3339),
	}
	return json.Marshal(timerange)
}

// TimeWindow is only time-of-day
type TimeWindow struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	On   []string  `json:"on,omitempty"`
}

func (t TimeWindow) MarshalJSON() ([]byte, error) {
	timewindow := struct {
		From string   `json:"from"`
		To   string   `json:"to"`
		On   []string `json:"on,omitempty"`
	}{
		From: t.From.Format("15:04:05"),
		To:   t.To.Format("15:04:05"),
		On:   t.On,
	}
	return json.Marshal(timewindow)
}

type Metric struct {
	Identifier string   `json:"identifier"`
	Include    []string `json:"include"`
}

type Bucketing struct {
	Duration string `json:"duration,omitempty"`
	Count    int    `json:"count,omitempty"`
}

// MetricsQuery request
func (h *Hub) MetricsQuery(ctx context.Context, request MetricsQueryRequest) (response MetricsQueryResponse, resp *http.Response, err error) {
	resp, err = h.Post(ctx, MetricsQueryEndpoint, request, &response)
	return
}

// MetricsAggregationsResponse type
type MetricsAggregationsResponse map[string]MetricsAggregation

// MetricsAggregation type
type MetricsAggregation struct {
	Parameters  []map[string]string `json:"parameters"`
	Description string              `json:"description"`
}

// MetricsAggregations request
func (h *Hub) MetricsAggregations(ctx context.Context) (response MetricsAggregationsResponse, resp *http.Response, err error) {
	resp, err = h.Get(ctx, MetricsAggregationsEndpoint, nil, response)
	return
}
