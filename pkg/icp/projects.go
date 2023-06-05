package icp

import (
	"context"
	"net/http"
	"time"
)

// ProjectsResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects
type ProjectsResponse []Project

// Project type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=Project
type Project struct {
	ID   int    `json:"Id"`
	Name string `json:"Name"`
}

// ProjectsModelsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
type ProjectsModelsRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId,omitempty"`
}

// ProjectsModelsResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
type ProjectsModelsResponse []AvailableModel

// AvailableModel type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=AvailableModel
type AvailableModel struct {
	SummaryLevelID int       `json:"SummaryLevelID"`
	StartDate      time.Time `json:"StartDate"`
	EndDate        time.Time `json:"EndDate"`
	Description    string    `json:"Description"`
}

// ProjectsIgnoreListRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId
type ProjectsIgnoreListRequest struct {
	ProjectID  int `json:"projectId"`
	BaselineID int `json:"baselineId"`
}

// ProjectsIgnoreListResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId
type ProjectsIgnoreListResponse []DynamicProperty

// DynamicProperty type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=DynamicProperty
type DynamicProperty struct {
	EntityName    string `json:"EntityName"`
	EntityID      int    `json:"EntityID"`
	PropertyName  string `json:"PropertyName"`
	PropertyValue string `json:"PropertyValue"`
}

// Projects queries ICP for the authorised projects for the user. The
// response contains the unmarshalled data while resp and err return
// information from the underlying request. The resp.Body is already
// closed, if there wa no error.
func (i *ICP) Projects(ctx context.Context) (response ProjectsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsEndpoint, nil, &response)
	return
}

// ProjectsModels gets a list of models for a specified project for the
// authenticated user
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
func (i *ICP) ProjectsModels(ctx context.Context, request *ProjectsModelsRequest) (response ProjectsModelsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsModelsEndpoint, request, &response)
	return
}
