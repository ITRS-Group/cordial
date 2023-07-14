package icp

import (
	"context"
	"net/http"
)

type RecommendationsCloudInstancesRequest struct {
	ProjectID      int    `json:"ProjectId"`
	Provider       string `json:"Provider"`
	MatchType      string `json:"MatchType"`
	SummaryLevelID int    `json:"SummaryLevelID"`
	SummaryDate    string `json:"SummaryDate"`
}

type RecommendationsCloudInstancesResponse []CloudInstanceRecommendation

type CloudInstanceRecommendation struct {
	AWSInstanceID   string `json:"AWSInstanceID,omitempty"`
	RegionCode      string `json:"RegionCode,omitempty"`
	ResourceType    string `json:"ResourceType,omitempty"`
	SubscriptionID  string `json:"SubscriptionID,omitempty"`
	ResourceGroup   string `json:"ResourceGroup,omitempty"`
	EntityID        string `json:"EntityID,omitempty"`
	EntityName      string `json:"EntityName,omitempty"`
	SourceCPUCores  int    `json:"SourceCPUCores,omitempty"`
	SourceMemoryGB  int    `json:"SourceMemoryGB,omitempty"`
	MatchType       string `json:"MatchType,omitempty"`
	InstanceType    string `json:"InstanceType,omitempty"`
	CPUCores        int    `json:"CPUCores,omitempty"`
	MemoryGB        int    `json:"MemoryGB,omitempty"`
	Location        string `json:"Location,omitempty"`
	LocationName    string `json:"LocationName,omitempty"`
	OperatingSystem string `json:"OperatingSystem,omitempty"`
	SKU             string `json:"SKU,omitempty"`
}

func (i *ICP) RecommendationsCloudInstances(ctx context.Context, request *RecommendationsCloudInstancesRequest) (response RecommendationsCloudInstancesResponse, resp *http.Response, err error) {
	resp, err = i.Post(ctx, RecommendationsCloudInstancesEndpoint, request, &response)
	return
}
