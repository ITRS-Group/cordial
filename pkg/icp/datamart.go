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

// DataMartEntityRelationRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
type DataMartEntityRelationRequest struct {
	ProjectID      int              `json:"ProjectId"`
	EntityRelation []EntityRelation `json:"EntityRelation"`
}

// DataMartPropertiesEntityRequest body type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
type DataMartPropertiesEntityRequest struct {
	ProjectID        int                `json:"ProjectId"`
	PropertiesEntity []PropertiesEntity `json:"PropertiesEntity"`
}

// DataMartMetricsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
type DataMartMetricsRequest struct {
	ProjectID      int       `json:"ProjectId"`
	DataSourceName string    `json:"DataSourceName"`
	SourceKey1     string    `json:"SourceKey1,omitempty"`
	SourceKey2     string    `json:"SourceKey2,omitempty"`
	SourceKey3     string    `json:"SourceKey3,omitempty"`
	SourceKey4     string    `json:"SourceKey4,omitempty"`
	SourceKey5     string    `json:"SourceKey5,omitempty"`
	Metrics        []Metrics `json:"Metrics"`
}

// DataMartMetricTimeSeriesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
type DataMartMetricTimeSeriesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1,omitempty"`
	SourceKey2       string             `json:"SourceKey2,omitempty"`
	SourceKey3       string             `json:"SourceKey3,omitempty"`
	SourceKey4       string             `json:"SourceKey4,omitempty"`
	SourceKey5       string             `json:"SourceKey5,omitempty"`
	MetricName       string             `json:"MetricName"`
	MetricTimeseries []MetricTimeSeries `json:"MetricTimeSeries"`
}

// DataMartEntityPropertiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
type DataMartEntityPropertiesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1,omitempty"`
	SourceKey2       string             `json:"SourceKey2,omitempty"`
	SourceKey3       string             `json:"SourceKey3,omitempty"`
	SourceKey4       string             `json:"SourceKey4,omitempty"`
	SourceKey5       string             `json:"SourceKey5,omitempty"`
	EntityProperties []EntityProperties `json:"Metrics"`
}

// DataMartMetricCapacitiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
type DataMartMetricCapacitiesRequest struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1,omitempty"`
	SourceKey2       string             `json:"SourceKey2,omitempty"`
	SourceKey3       string             `json:"SourceKey3,omitempty"`
	SourceKey4       string             `json:"SourceKey4,omitempty"`
	SourceKey5       string             `json:"SourceKey5,omitempty"`
	MetricName       string             `json:"MetricName"`
	MetricCapacities []MetricCapacities `json:"MetricCapacities"`
}

// DataMartGetEntitiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
type DataMartGetEntitiesRequest struct {
	ProjectID  int      `json:"ProjectId"`
	EntityType []string `json:"EntityType"`
}

// DataMartStartProcessingRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
type DataMartStartProcessingRequest struct {
	ProjectID int `json:"ProjectId"`
}

// DataMartEntityPerformance request
func (i *ICP) DataMartEntityPerformance(ctx context.Context, request *DataMartEntityPerformanceRequest) (resp *http.Response, err error) {
	return i.Post(ctx, DataMartEntityPerformanceEndpoint, request)
}
