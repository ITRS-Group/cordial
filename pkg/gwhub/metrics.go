package gwhub

import (
	"context"
	"net/http"
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
	Schema Schema `json:"schema"`
	Data   []Data `json:"data"`
}

type Schema struct {
	Bucketing Bucketing                         `json:"bucketing"`
	Grouping  []string                          `json:"grouping"`
	Metrics   map[string]map[string]interface{} `json:"metrics"`
}

type Data struct {
	Bucket   string                            `json:"bucket"`
	Grouping DataGrouping                      `json:"group"`
	Metrics  map[string]map[string]interface{} `json:"metrics"`
}

type DataGrouping struct {
	Rowname   string `json:"rowname,omitempty"`
	Entity    int    `json:"entity,omitempty"`
	Attribute string `json:"attribute,omitempty"`
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
