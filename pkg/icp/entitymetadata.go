package icp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// EntityMetaDataRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entitymetadata
type EntityMetaDataRequest []EntityMetaDataItem

// EntityMetaDataItem type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entitymetadata
type EntityMetaDataItem struct {
	SourceServer string
	InternalID   string
	Groupings    []Grouping
	Action       string
}

// EntityMetaData request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entitymetadata
func (i *ICP) EntityMetaData(ctx context.Context, request EntityMetaDataRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, EntityMetaDataEndpoint, request, nil)
	resp.Body.Close()
	return
}

// EntityMeteDataExportRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metadataexport-projectId_onlyInclude
type EntityMeteDataExportRequest struct {
	ProjectID   int    `url:"projectId,omitempty"`
	OnlyInclude string `url:"onlyInclude,omitempty"`
}

// EntityMetaDataExportResponse type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metadataexport-projectId_onlyInclude
type EntityMetaDataExportResponse []string

// EntityMetaDataExport type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metadataexport-projectId_onlyInclude
func (i *ICP) EntityMetaDataExport(ctx context.Context, request *EntityMeteDataExportRequest) (response EntityMetaDataExportResponse, resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntityMetaDataExportEndpoint, fmt.Sprint(request.ProjectID))
	request.ProjectID = 0
	resp, err = i.Get(ctx, endpoint, request, &response)
	return
}
