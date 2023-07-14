package icp

import (
	"context"
	"net/http"
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

// Projects queries ICP for the authorised projects for the user. The
// response contains the unmarshalled data while resp and err return
// information from the underlying request. The resp.Body is already
// closed, if there wa no error.
func (i *ICP) Projects(ctx context.Context) (response ProjectsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsEndpoint, nil, &response)
	return
}

type ProjectsSolutionIDRequest struct {
	ProjectID int `url:"projectId"`
}

type ProjectsSolutionIDResponse int32

func (i *ICP) ProjectsSolutionID(ctx context.Context, request *ProjectsSolutionIDRequest) (response ProjectsSolutionIDResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsSolutionIDEndpoint, request, &response)
	return
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
	SummaryLevelID int    `json:"SummaryLevelID"`
	StartDate      *Time  `json:"StartDate,omitempty"`
	EndDate        *Time  `json:"EndDate,omitempty"`
	Description    string `json:"Description"`
}

// ProjectsModels gets a list of models for a specified project for the
// authenticated user
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
func (i *ICP) ProjectsModels(ctx context.Context, request *ProjectsModelsRequest) (response ProjectsModelsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsModelsEndpoint, request, &response)
	return
}

// ProjectsIgnoreListRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId
type ProjectsIgnoreListRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId,omitempty"`
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

func (i *ICP) ProjectsIgnoreList(ctx context.Context, request *ProjectsIgnoreListRequest) (response ProjectsIgnoreListResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, ProjectsIgnoreListEndpoint, request, &response)
	return
}
