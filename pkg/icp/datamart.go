// Package icp provides binding the the ITRS Capacity Planner data model
package icp

// DataMartEntityPerformance body type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
type DataMartEntityPerformance struct {
	ProjectID         int                 `json:"ProjectId"`
	EntityPerformance []EntityPerformance `json:"EntityPerformance"`
}

// DataMartEntityRelation type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
type DataMartEntityRelation struct {
	ProjectID      int              `json:"ProjectId"`
	EntityRelation []EntityRelation `json:"EntityRelation"`
}

// DataMartPropertiesEntity body type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
type DataMartPropertiesEntity struct {
	ProjectID        int                `json:"ProjectId"`
	PropertiesEntity []PropertiesEntity `json:"PropertiesEntity"`
}

// DataMartMetrics type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
type DataMartMetrics struct {
	ProjectID      int       `json:"ProjectId"`
	DataSourceName string    `json:"DataSourceName"`
	SourceKey1     string    `json:"SourceKey1,omitempty"`
	SourceKey2     string    `json:"SourceKey2,omitempty"`
	SourceKey3     string    `json:"SourceKey3,omitempty"`
	SourceKey4     string    `json:"SourceKey4,omitempty"`
	SourceKey5     string    `json:"SourceKey5,omitempty"`
	Metrics        []Metrics `json:"Metrics"`
}

// DataMartMetricTimeSeries type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
type DataMartMetricTimeSeries struct {
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

// DataMartEntityProperties type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
type DataMartEntityProperties struct {
	ProjectID        int                `json:"ProjectId"`
	DataSourceName   string             `json:"DataSourceName"`
	SourceKey1       string             `json:"SourceKey1,omitempty"`
	SourceKey2       string             `json:"SourceKey2,omitempty"`
	SourceKey3       string             `json:"SourceKey3,omitempty"`
	SourceKey4       string             `json:"SourceKey4,omitempty"`
	SourceKey5       string             `json:"SourceKey5,omitempty"`
	EntityProperties []EntityProperties `json:"Metrics"`
}

// DataMartMetricCapacities type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
type DataMartMetricCapacities struct {
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

// DataMartGetEntities type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
type DataMartGetEntities struct {
	ProjectID  int      `json:"ProjectId"`
	EntityType []string `json:"EntityType"`
}

// DataMartStartProcessing type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
type DataMartStartProcessing struct {
	ProjectID int `json:"ProjectId"`
}
