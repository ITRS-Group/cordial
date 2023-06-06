package icp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// EventBaselineRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
type EventBaselineRequest struct {
	BaselineviewID         string `json:"BaselineviewId"`
	Filter                 string `json:"Filter"`
	ExcludePrecedingEvents bool   `json:"ExcludePrecedingEvents"`
	IncludeClusterEvents   bool   `json:"IncludeClusterEvents"`
}

// EventBaselineResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
type EventBaselineResponse []PredictedEvent

// EventBaseline request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview
func (i *ICP) EventBaseline(ctx context.Context, request EventBaselineRequest) (response EventBaselineResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, EventsBaselineViewEndpoint, request)
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

// EventFilterRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-eventfilter
type EventFilterRequest struct {
	ProjectID   int       `json:"ProjectID"`
	Start       time.Time `json:"Start"`
	End         time.Time `json:"End"`
	FiltersJSON string    `json:"FiltersJson"`
}

// EventFilterSet request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-eventfilter
func (i *ICP) EventFilterSet(ctx context.Context, request EventFilterRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, EventsEventFilterEndpoint, request)
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
