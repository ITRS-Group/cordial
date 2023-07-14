package icp

import (
	"context"
	"net/http"
)

// EntityPropertiesRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entityproperties
type EntityPropertiesRequest []EntitiesPropertiesItem

// EntitiesPropertiesItem type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=EntityPropertiesItem
type EntitiesPropertiesItem struct {
	KeyField1  string     `json:"KeyField1"`
	KeyField2  string     `json:"KeyField2"`
	KeyField3  string     `json:"KeyField3"`
	KeyField4  string     `json:"KeyField4"`
	KeyField5  string     `json:"KeyField5"`
	EntityType string     `json:"EntityType"`
	EntityName string     `json:"EntityName"`
	Timestamp  string     `json:"Timestamp"`
	Properties []Grouping `json:"Properties"`
}

// EntityProperties type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entityproperties
func (i *ICP) EntityProperties(ctx context.Context, request *EntityPropertiesRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, EntityPropertiesEndpoint, request, nil)
	resp.Body.Close()
	return
}
