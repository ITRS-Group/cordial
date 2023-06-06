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

type TimeWindow struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	On   []string  `json:"on"`
}

type Metric struct {
	Identifier string   `json:"identifier"`
	Include    []string `json:"include"`
}

type Bucketing struct {
	Duration time.Duration `json:"duration,omitempty"`
	Count    int           `json:"count,omitempty"`
}

// MetricsQuery request
func (h *Hub) MetricsQuery(ctx context.Context, request MetricsQueryRequest, response *MetricsQueryResponse) (resp *http.Response, err error) {
	resp, err = h.Post(ctx, MetricsQueryEndpoint, request)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		err = ErrServerError
		return
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&response)
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
func (h *Hub) MetricsAggregations(ctx context.Context, response *MetricsAggregationsResponse) (resp *http.Response, err error) {
	return h.Get(ctx, MetricsAggregationsEndpoint, nil, &response)
}
