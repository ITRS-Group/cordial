package icp

import (
	"context"
	"net/http"
)

// MetricsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metrics_projectId_baselineId
type MetricsRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId,omitempty"`
}

// MetricsResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metrics_projectId_baselineId
type MetricsResponse []Metrics

// Metrics request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metrics_projectId_baselineId
func (i *ICP) Metrics(ctx context.Context, request MetricsRequest) (response MetricsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, MetricsEndpoint, request, &response)
	return
}

// MetricsSummariesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
type MetricsSummariesRequest struct {
	ProjectID        int    `json:"ProjectId,omitempty"`
	BaselineID       int    `json:"BaselineId,omitempty"`
	EntityIDList     string `json:"EntityIDList,omitempty"`
	SummaryLevelID   int    `json:"SummaryLevelID,omitempty"`
	SummaryStartDate string `json:"SummaryStartDate,omitempty"`
}

// MetricsSummariesResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
type MetricsSummariesResponse []MetricsSummaryItem

type MetricsSummaryItem struct {
	EntityID            int     `json:"EntityID,omitempty"`
	Metric              string  `json:"Metric,omitempty"`
	SummaryDate         *Time   `json:"SummaryDate,omitempty"`
	RecordCount         float64 `json:"RecordCount,omitempty"`
	LatestValue         float64 `json:"LatestValue,omitempty"`
	LatestDataDateTime  *Time   `json:"LatestDataDateTime,omitempty"`
	MinModelMetricValue float64 `json:"MinModelMetricValue,omitempty"`
	MaxModelMetricValue float64 `json:"MaxModelMetricValue,omitempty"`
	C5thPercentile      float64 `json:"c5thPercentile,omitempty"`
	C25thPercentile     float64 `json:"c25thPercentile,omitempty"`
	C50thPercentile     float64 `json:"c50thPercentile,omitempty"`
	C75thPercentile     float64 `json:"c75thPercentile,omitempty"`
	C95thPercentile     float64 `json:"c95thPercentile,omitempty"`
	C99thPercentile     float64 `json:"c99thPercentile,omitempty"`
	STDDev              float64 `json:"STDDev,omitempty"`
	Mean                float64 `json:"Mean,omitempty"`
	Total               float64 `json:"Total,omitempty"`
}

// MetricsSummaries request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
func (i *ICP) MetricsSummaries(ctx context.Context, request MetricsSummariesRequest) (response MetricsSummariesResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, MetricsSummariesEndpoint, request, &response)
	return
}

// MetricsSummariesDateRangeRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summariesdaterange
type MetricsSummariesDateRangeRequest struct {
	MetricID         int    `json:"MetricID,omitempty"`
	SummaryEndDate   string `json:"SummaryEndDate,omitempty"`
	ProjectID        int    `json:"ProjectId,omitempty"`
	BaselineID       int    `json:"BaselineId,omitempty"`
	EntityIDList     string `json:"EntityIDList,omitempty"`
	SummaryLevelID   int    `json:"SummaryLevelID,omitempty"`
	SummaryStartDate string `json:"SummaryStartDate,omitempty"`
}

// MetricsSummariesDateRange request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summariesdaterange
func (i *ICP) MetricsSummariesDateRange(ctx context.Context, request MetricsSummariesDateRangeRequest) (response MetricsSummariesResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, MetricsSummariesDateRangeEndpoint, request, &response)
	return
}
