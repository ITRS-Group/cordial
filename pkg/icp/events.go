package icp

import (
	"context"
	"net/http"
)

// EventsBaselineViewRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
type EventsBaselineViewRequest struct {
	BaselineviewID         string `json:"BaselineviewId"`
	Filter                 string `json:"Filter"`
	ExcludePrecedingEvents bool   `json:"ExcludePrecedingEvents"`
	IncludeClusterEvents   bool   `json:"IncludeClusterEvents"`
}

// EventsBaselineViewResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
type EventsBaselineViewResponse []PredictedEvent

// EventsBaselineView request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
func (i *ICP) EventsBaselineView(ctx context.Context, request EventsBaselineViewRequest) (response EventsBaselineViewResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, EventsBaselineViewEndpoint, request, &response)
	return
}

// EventFilterSetRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-eventfilter
type EventFilterSetRequest struct {
	ProjectID   int    `json:"ProjectID"`
	Start       *Time  `json:"Start,omitempty"`
	End         *Time  `json:"End,omitempty"`
	FiltersJSON string `json:"FiltersJson"`
}

// EventFilterSet request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-eventfilter
func (i *ICP) EventFilterSet(ctx context.Context, request EventFilterSetRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, EventsEventFilterEndpoint, request, nil)
	resp.Body.Close()
	return
}

// EventFiltersRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-events-eventfilter_projectId
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/DELETE-api-events-eventfilter_projectId
type EventFiltersRequest struct {
	ProjectID int `url:"projectId"`
}

// EventFiltersResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-events-eventfilter_projectId
type EventFiltersResponse string

// EventFilters request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-events-eventfilter_projectId
func (i *ICP) EventFilters(ctx context.Context, request EventFiltersRequest) (response EventFiltersResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, EventsEventFilterEndpoint, request, &response)
	return
}

// EventFilterDelete request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/DELETE-api-events-eventfilter_projectId
func (i *ICP) EventFilterDelete(ctx context.Context, request EventFiltersRequest) (resp *http.Response, err error) {
	return i.Delete(ctx, EventsEventFilterEndpoint, request)
}
