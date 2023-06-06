package icp

import "time"

// Version type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=PropertiesEntity
type Version struct {
	Major         int `json:"Major,omitempty"`
	Minor         int `json:"Minor,omitempty"`
	Build         int `json:"Build,omitempty"`
	Revision      int `json:"Revision,omitempty"`
	MajorRevision int `json:"MajorRevision,omitempty"`
	MinorRevision int `json:"MinorRevision,omitempty"`
}

// EntityPerformance type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityPerformance
type EntityPerformance struct {
	DataSourceName string    `json:"DataSourceName"`
	DataDateTime   time.Time `json:"DataDateTime"`
	SourceKey1     string    `json:"SourceKey1,omitempty"`
	SourceKey2     string    `json:"SourceKey2,omitempty"`
	SourceKey3     string    `json:"SourceKey3,omitempty"`
	SourceKey4     string    `json:"SourceKey4,omitempty"`
	SourceKey5     string    `json:"SourceKey5,omitempty"`
	MetricName     string    `json:"MetricName"`
	MetricValue    string    `json:"MetricValue"`
}

// EntityRelation type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityRelation
type EntityRelation struct {
	DataSourceName    string    `json:"DataSourceName"`
	Entity1SourceKey1 string    `json:"Entity1SourceKey1,omitempty"`
	Entity1SourceKey2 string    `json:"Entity1SourceKey2,omitempty"`
	Entity1SourceKey3 string    `json:"Entity1SourceKey3,omitempty"`
	Entity1SourceKey4 string    `json:"Entity1SourceKey4,omitempty"`
	Entity1SourceKey5 string    `json:"Entity1SourceKey5,omitempty"`
	Entity2SourceKey1 string    `json:"Entity2SourceKey1,omitempty"`
	Entity2SourceKey2 string    `json:"Entity2SourceKey2,omitempty"`
	Entity2SourceKey3 string    `json:"Entity2SourceKey3,omitempty"`
	Entity2SourceKey4 string    `json:"Entity2SourceKey4,omitempty"`
	Entity2SourceKey5 string    `json:"Entity2SourceKey5,omitempty"`
	RelationName      string    `json:"RelationName"`
	EffectiveFrom     time.Time `json:"EffectiveFrom"`
}

// PropertiesEntity type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=PropertiesEntity
type PropertiesEntity struct {
	DataSourceName string    `json:"DataSourceName"`
	SourceKey1     string    `json:"SourceKey1,omitempty"`
	SourceKey2     string    `json:"SourceKey2,omitempty"`
	SourceKey3     string    `json:"SourceKey3,omitempty"`
	SourceKey4     string    `json:"SourceKey4,omitempty"`
	SourceKey5     string    `json:"SourceKey5,omitempty"`
	PropertyName   string    `json:"PropertyName"`
	PropertyValue  string    `json:"PropertyValue"`
	EffectiveFrom  time.Time `json:"EffectiveFrom"`
}

// Metrics type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=Metrics
type Metrics struct {
	MetricName   string    `json:"MetricName"`
	MetricValue  string    `json:"MetricValue"`
	DataDateTime time.Time `json:"DataDateTime"`
}

// MetricTimeSeries type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=MetricTimeseries
type MetricTimeSeries struct {
	MetricValue  string    `json:"MetricValue"`
	DataDateTime time.Time `json:"DataDateTime"`
}

// EntityProperties type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityProperties
type EntityProperties struct {
	PropertyName  string    `json:"PropertyName"`
	PropertyValue string    `json:"PropertyValue"`
	EffectiveFrom time.Time `json:"EffectiveFrom"`
}

// MetricCapacities type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=MetricCapacities
type MetricCapacities struct {
	CapacityValue string    `json:"CapacityValue"`
	EffectiveFrom time.Time `json:"EffectiveFrom"`
}

// PredictedEvent type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=PredictedEvent
type PredictedEvent struct {
	ID             int       `json:"Id,omitempty"`
	Priority       string    `json:"Priority,omitempty"`
	Confidence     float64   `json:"Confidence,omitempty"`
	EventTime      time.Time `json:"EventTime,omitempty"`
	Metric         string    `json:"Metric,omitempty"`
	Description    string    `json:"Description,omitempty"`
	Entity         string    `json:"Entity,omitempty"`
	EntityType     string    `json:"EntityType,omitempty"`
	Element        string    `json:"Element,omitempty"`
	InternalID     string    `json:"InternalId,omitempty"`
	SourceServer   string    `json:"SourceServer,omitempty"`
	IsClusterEvent bool      `json:"IsClusterEvent,omitempty"`
	Critical       string    `json:"Critical,omitempty"`
	Major          string    `json:"Major,omitempty"`
	Warning        string    `json:"Warning,omitempty"`
}
