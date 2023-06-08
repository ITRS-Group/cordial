package gwhub

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// EntitiesRequest type
type EntitiesRequest struct {
	Filter         string `url:"filter,omitempty"`
	IncludeDeleted bool   `url:"includeDeleted"`
}

// EntitiesResponse type
type EntitiesResponse []Entity

// Entity type
type Entity struct {
	ID         int               `json:"id"`
	Attributes map[string]string `json:"attributes"`
	IsDeleted  bool              `json:"isDeleted"`
}

// Entities request
func (h *Hub) Entities(ctx context.Context, request EntitiesRequest, response *EntitiesResponse) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesEndpoint, request, response)
}

// EntityMetricsRequest type
type EntityMetricsRequest struct {
	EntityID          int  `url:"-"`
	IncludeDeleted    bool `url:"includeDeleted,omitempty"`
	IncludeNonNumeric bool `url:"includeNonNumeric,omitempty"`
}

// EntityMetricsResponse type
type EntityMetricsResponse []EntityMetric

// EntityMetric type
type EntityMetric struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Unit     string `json:"unit"`
	Mappings []EntityMetricMapping
}

// EntityMetricMapping type
type EntityMetricMapping struct {
	LegacyPath string `json:"legacyPath"`
	DataviewID int    `json:"dataviewId"`
	IsDeleted  bool   `json:"isDeleted"`
}

// EntityMetrics request
func (h *Hub) EntityMetrics(ctx context.Context, request EntityMetricsRequest, response *EntityMetricsResponse) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesEndpoint, fmt.Sprint(request.EntityID), "metrics")
	return h.Get(ctx, endpoint, request, response)
}

// EntitiesSummaryResponse type
type EntitiesSummaryResponse []EntitiesSummary

// EntitiesSummary type
type EntitiesSummary struct {
	Attributes        map[string]string        `json:"attributes"`
	Count             int                      `json:"count"`
	SampleEntityCount int                      `json:"sampleEntityCount"`
	Counts            []EntitiesSummariesCount `json:"counts"`
}

// EntitiesSummariesCount type
type EntitiesSummariesCount struct {
	States         map[string]string `json:"states"`
	Count          int               `json:"count"`
	SampleEntities []int             `json:"sampleEntities"`
}

// EntitiesSummary request
func (h *Hub) EntitiesSummary(ctx context.Context, response *EntitiesSummaryResponse) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesSummaryEndpoint, nil, response)
}

// Entity request
func (h *Hub) Entity(ctx context.Context, entity int, response *Entity) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesEndpoint, fmt.Sprint(entity))
	return h.Get(ctx, endpoint, nil, response)
}

// EntityMetricsRowsRequest type
type EntityMetricsRowsRequest struct {
	EntityID     int       `url:"-"`
	Path         string    `url:"path"`
	UpdatedSince time.Time `url:"updatedSince,omitempty"`
}

// EntityMetricsRowsResponse type
type EntityMetricsRowsResponse []string

// EntityMetricsRows request
func (h *Hub) EntityMetricsRows(ctx context.Context, request EntityMetricsRowsRequest, response *EntityMetricsRowsResponse) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesEndpoint, fmt.Sprint(request.EntityID), "metrics", "rows")
	return h.Get(ctx, endpoint, request, response)
}

// EntitiesMetricsRequest type
type EntitiesMetricsRequest struct {
	Filter            string `url:"filter,omitempty"`
	Plugin            string `url:"plugin,omitempty"`
	IncludeMappings   bool   `url:"includeMappings,omitempty"`
	IncludeDeleted    bool   `url:"includeDeleted,omitempty"`
	IncludeNonNumeric bool   `url:"includeNonNumeric,omitempty"`
}

// EntitiesMetricsResponse type
type EntitiesMetricsResponse []EntityMetric

// EntitiesMetrics request
func (h *Hub) EntitiesMetrics(ctx context.Context, request EntitiesMetricsRequest, response *EntitiesMetricsResponse) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesMetricsEndpoint, request, response)
}

// EntitiesMetricsRowsRequest type
type EntitiesMetricsRowsRequest struct {
	Filter       string    `url:"filter,omitempty"`
	Path         string    `url:"path"`
	UpdatedSince time.Time `url:"updatedSince,omitempty"`
}

// EntitiesMetricsRowsResponse type
type EntitiesMetricsRowsResponse []string

// EntitiesMetricsRows request
func (h *Hub) EntitiesMetricsRows(ctx context.Context, request EntityMetricsRowsRequest, response *EntityMetricsRowsResponse) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesMetricsRowsEndpoint, request, response)
}

// EntitiesAttributesRequest type
type EntitiesAttributesRequest struct {
	Filter         string `url:"filter,omitempty"`
	IncludeDeleted bool   `url:"includeDeleted,omitempty"`
}

// EntitiesAttributesResponse type
type EntitiesAttributesResponse []EntityAttributes

// EntityAttributes type
type EntityAttributes struct {
	Name       string `json:"name"`
	ValueCount int    `json:"valueCount"`
}

// EntitiesAttributes request
func (h *Hub) EntitiesAttributes(ctx context.Context, request EntitiesAttributesRequest, response *EntitiesAttributesResponse) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesAttributesEndpoint, request, response)
}

// EntitiesAttributeRequest type
type EntitiesAttributeRequest struct {
	Name           string `url:"-"`
	Filter         string `url:"filter,omitempty"`
	IncludeDeleted bool   `url:"includeDeleted,omitempty"`
}

// EntityAttributeResponse type
type EntityAttributeResponse []string

// EntityAttributeValues request
func (h *Hub) EntityAttributeValues(ctx context.Context, request EntitiesAttributeRequest, response *EntityAttributeResponse) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesAttributesEndpoint, request.Name)
	return h.Get(ctx, endpoint, nil, response)
}

// EntityMetricsDataRequest type
type EntityMetricsDataRequest struct {
	EntityID int       `url:"-"`
	Path     string    `url:"path"`
	From     time.Time `url:"from"`
	To       time.Time `url:"to"`
	Skip     int       `url:"skip"`
	Limit    int       `url:"limit"`
}

// EntityMetricsDataResponse type
type EntityMetricsDataResponse struct {
	TotalCount int `json:"totalCount"`
	Values     []EntityMetricsDataValue
}

// EntityMetricsDataValue type
type EntityMetricsDataValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
}

// EntityMetricsData request
func (h *Hub) EntityMetricsData(ctx context.Context, request EntityMetricsDataRequest, response *EntityMetricsDataResponse) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesEndpoint, fmt.Sprint(request.EntityID), "metrics", "data")
	return h.Get(ctx, endpoint, request, response)
}
