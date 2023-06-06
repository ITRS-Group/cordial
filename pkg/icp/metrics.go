package icp

import (
	"context"
	"encoding/json"
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
type MetricsSummariesResponse map[string]string

// MetricsSummaries request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
func (i *ICP) MetricsSummaries(ctx context.Context, request MetricsSummariesRequest) (response MetricsSummariesResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, MetricsSummariesEndpoint, request)
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
	resp, err = i.Post(ctx, MetricsSummariesDateRangeEndpoint, request)
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
