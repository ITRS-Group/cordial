package icp

import (
	"bytes"
	"fmt"
	"time"
)

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
	DataSourceName string `json:"DataSourceName"`
	DataDateTime   *Time  `json:"DataDateTime"`
	SourceKey1     string `json:"SourceKey1"`
	SourceKey2     string `json:"SourceKey2"`
	SourceKey3     string `json:"SourceKey3"`
	SourceKey4     string `json:"SourceKey4"`
	SourceKey5     string `json:"SourceKey5"`
	MetricName     string `json:"MetricName"`
	MetricValue    string `json:"MetricValue"`
}

// EntityRelation type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityRelation
type EntityRelation struct {
	DataSourceName    string `json:"DataSourceName"`
	Entity1SourceKey1 string `json:"Entity1SourceKey1"`
	Entity1SourceKey2 string `json:"Entity1SourceKey2"`
	Entity1SourceKey3 string `json:"Entity1SourceKey3"`
	Entity1SourceKey4 string `json:"Entity1SourceKey4"`
	Entity1SourceKey5 string `json:"Entity1SourceKey5"`
	Entity2SourceKey1 string `json:"Entity2SourceKey1"`
	Entity2SourceKey2 string `json:"Entity2SourceKey2"`
	Entity2SourceKey3 string `json:"Entity2SourceKey3"`
	Entity2SourceKey4 string `json:"Entity2SourceKey4"`
	Entity2SourceKey5 string `json:"Entity2SourceKey5"`
	RelationName      string `json:"RelationName"`
	EffectiveFrom     *Time  `json:"EffectiveFrom"`
}

// PropertiesEntity type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=PropertiesEntity
type PropertiesEntity struct {
	DataSourceName string `json:"DataSourceName"`
	SourceKey1     string `json:"SourceKey1"`
	SourceKey2     string `json:"SourceKey2"`
	SourceKey3     string `json:"SourceKey3"`
	SourceKey4     string `json:"SourceKey4"`
	SourceKey5     string `json:"SourceKey5"`
	PropertyName   string `json:"PropertyName"`
	PropertyValue  string `json:"PropertyValue"`
	EffectiveFrom  *Time  `json:"EffectiveFrom"`
}

// Metrics type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=Metrics
type Metrics struct {
	MetricName   string `json:"MetricName"`
	MetricValue  string `json:"MetricValue"`
	DataDateTime *Time  `json:"DataDateTime"`
}

// MetricTimeSeries type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=MetricTimeseries
type MetricTimeSeries struct {
	MetricValue  string `json:"MetricValue"`
	DataDateTime *Time  `json:"DataDateTime"`
}

// EntityProperties type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityProperties
type EntityProperties struct {
	PropertyName  string `json:"PropertyName"`
	PropertyValue string `json:"PropertyValue"`
	EffectiveFrom *Time  `json:"EffectiveFrom"`
}

// MetricCapacities type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=MetricCapacities
type MetricCapacities struct {
	CapacityValue string `json:"CapacityValue"`
	EffectiveFrom *Time  `json:"EffectiveFrom"`
}

// PredictedEvent type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=PredictedEvent
type PredictedEvent struct {
	ID             int     `json:"Id,omitempty"`
	Priority       string  `json:"Priority,omitempty"`
	Confidence     float64 `json:"Confidence,omitempty"`
	EventTime      *Time   `json:"EventTime,omitempty"`
	Metric         string  `json:"Metric,omitempty"`
	Description    string  `json:"Description,omitempty"`
	Entity         string  `json:"Entity,omitempty"`
	EntityType     string  `json:"EntityType,omitempty"`
	Element        string  `json:"Element,omitempty"`
	InternalID     string  `json:"InternalId,omitempty"`
	SourceServer   string  `json:"SourceServer,omitempty"`
	IsClusterEvent bool    `json:"IsClusterEvent,omitempty"`
	Critical       string  `json:"Critical,omitempty"`
	Major          string  `json:"Major,omitempty"`
	Warning        string  `json:"Warning,omitempty"`
}

// Grouping type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=Grouping
type Grouping struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ICP date times may not have a timezone, work around it

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	if bytes.Equal(data, []byte("null")) || bytes.Contains(data, []byte("0001-01-01T00:00:00")) {
		return
	}
	if bytes.Contains(data, []byte("Z")) {
		t.Time, err = time.Parse(`"2006-01-02T15:04:05Z"`, string(data))
	} else {
		t.Time, err = time.Parse(`"2006-01-02T15:04:05"`, string(data))
	}
	if err == nil {
		return
	}

	// try default
	return t.Time.UnmarshalJSON(data)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	ts := fmt.Sprintf("%q", t.UTC().Format(time.RFC3339))
	return []byte(ts), nil
}
