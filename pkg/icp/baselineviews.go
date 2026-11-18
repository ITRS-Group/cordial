package icp

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// BaselineViewsProjectResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-project-projectId
type BaselineViewsProjectResponse []BaselineView

// BaselineViewsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId
type BaselineViewsRequest string

// BaselineViewsResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId
type BaselineViewsResponse BaselineView

// BaselineView type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=BaselineView
type BaselineView struct {
	ProjectID         int    `json:"ProjectId"`
	ID                string `json:"ID"`
	Name              string `json:"Name"`
	StartDate         *Time  `json:"StartDate,omitempty"`
	EndDate           *Time  `json:"EndDate,omitempty"`
	LastProcessedDate *Time  `json:"LastProcessedDate"`
	BaselineID        int    `json:"BaselineId"`
}

// BaselineViewsProject request
func (i *ICP) BaselineViewsProject(ctx context.Context, request int) (response BaselineViewsProjectResponse, resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(BaselineViewsProjectEndpoint, strconv.Itoa(request))
	resp, err = i.Get(ctx, endpoint, nil, &response)
	return
}

// BaselineView request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId
func (i *ICP) BaselineView(ctx context.Context, request BaselineViewsRequest) (response BaselineViewsResponse, resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(BaselineViewsEndpoint, url.PathEscape(string(request)))
	resp, err = i.Get(ctx, endpoint, nil, &response)
	return
}
