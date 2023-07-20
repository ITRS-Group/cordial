// Package icp provides binding the the ITRS Capacity Planner data model
package icp

import (
	"context"
	"net/http"
)

// DataMartEntityPerformanceRequest body type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
type DataMartEntityPerformanceRequest struct {
	ProjectID         int                 `json:"ProjectId"`
	EntityPerformance []EntityPerformance `json:"EntityPerformance"`
}

// DataMartEntityPerformance request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
func (i *ICP) DataMartEntityPerformance(ctx context.Context, request *DataMartEntityPerformanceRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartEntityPerformanceEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartEntityRelationRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
type DataMartEntityRelationRequest struct {
	ProjectID      int              `json:"ProjectId"`
	EntityRelation []EntityRelation `json:"EntityRelation"`
}

// DataMartEntityRelation request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
func (i *ICP) DataMartEntityRelation(ctx context.Context, request *DataMartEntityRelationRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartEntityRelationEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartPropertiesEntityRequest body type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
type DataMartPropertiesEntityRequest struct {
	ProjectID        int                `json:"ProjectId"`
	PropertiesEntity []PropertiesEntity `json:"PropertiesEntity"`
}

// DataMartPropertiesEntity request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
func (i *ICP) DataMartPropertiesEntity(ctx context.Context, request *DataMartPropertiesEntityRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartPropertiesEntityEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartMetricsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
type DataMartMetricsRequest struct {
	ProjectID      int       `json:"ProjectId"`
	DataSourceName string    `json:"DataSourceName"`
	SourceKey1     string    `json:"SourceKey1"`
	SourceKey2     string    `json:"SourceKey2"`
	SourceKey3     string    `json:"SourceKey3"`
	SourceKey4     string    `json:"SourceKey4"`
	SourceKey5     string    `json:"SourceKey5"`
	Metrics        []Metrics `json:"Metrics"`
}

// DataMartMetrics request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
func (i *ICP) DataMartMetrics(ctx context.Context, request *DataMartMetricsRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartMetricsEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartMetricTimeSeriesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
type DataMartMetricTimeSeriesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1"`
	SourceKey2       string             `json:"SourceKey2"`
	SourceKey3       string             `json:"SourceKey3"`
	SourceKey4       string             `json:"SourceKey4"`
	SourceKey5       string             `json:"SourceKey5"`
	MetricName       string             `json:"MetricName"`
	MetricTimeseries []MetricTimeSeries `json:"MetricTimeSeries"`
}

// DataMartMetricTimeseries request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
func (i *ICP) DataMartMetricTimeseries(ctx context.Context, request *DataMartMetricTimeSeriesRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartMetricTimeseriesEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartEntityPropertiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
type DataMartEntityPropertiesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1"`
	SourceKey2       string             `json:"SourceKey2"`
	SourceKey3       string             `json:"SourceKey3"`
	SourceKey4       string             `json:"SourceKey4"`
	SourceKey5       string             `json:"SourceKey5"`
	EntityProperties []EntityProperties `json:"EntityProperties"`
}

// DataMartEntityProperties request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
func (i *ICP) DataMartEntityProperties(ctx context.Context, request *DataMartEntityPropertiesRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartEntityPropertiesEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartMetricCapacitiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
type DataMartMetricCapacitiesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1"`
	SourceKey2       string             `json:"SourceKey2"`
	SourceKey3       string             `json:"SourceKey3"`
	SourceKey4       string             `json:"SourceKey4"`
	SourceKey5       string             `json:"SourceKey5"`
	MetricName       string             `json:"MetricName"`
	MetricCapacities []MetricCapacities `json:"MetricCapacities"`
}

// DataMartMetricCapacities request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
func (i *ICP) DataMartMetricCapacities(ctx context.Context, request *DataMartMetricCapacitiesRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartMetricCapacitiesEndpoint, request, nil)
	resp.Body.Close()
	return
}

// DataMartGetEntitiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
type DataMartGetEntitiesRequest struct {
	ProjectID  int      `json:"ProjectId"`
	EntityType []string `json:"EntityType"`
}

type DataMartGetEntitiesResponse []DataMartGetEntitiesItem

type DataMartGetEntitiesItem struct {
	SourceKey1 string `json:"SourceKey1"`
	SourceKey2 string `json:"SourceKey2"`
	SourceKey3 string `json:"SourceKey3"`
	SourceKey4 string `json:"SourceKey4"`
	SourceKey5 string `json:"SourceKey5"`
	EntityType string `json:"EntityType"`
}

// DataMartGetEntities request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
func (i *ICP) DataMartGetEntities(ctx context.Context, request *DataMartGetEntitiesRequest) (response DataMartGetEntitiesResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartGetEntitiesEndpoint, request, &response)
	resp.Body.Close()
	return
}

// DataMartStartProcessingRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
type DataMartStartProcessingRequest struct {
	ProjectID int `json:"ProjectId"`
}

// DataMartStartProcessing request
//
// (Response format unknown)
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
func (i *ICP) DataMartStartProcessing(ctx context.Context, request *DataMartStartProcessingRequest) (response interface{}, resp *http.Response, err error) {
	resp, err = i.Post(ctx, DataMartStartProcessingEndpoint, request, response)
	resp.Body.Close()
	return
}
