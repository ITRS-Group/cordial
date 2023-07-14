package icp

import (
	"context"
	"net/http"
)

// AssetServersRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-servers_projectId_baselineId
type AssetServersRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId"`
}

// AssetServersResponse type
// XXX Unknown response from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-servers_projectId_baselineId
type AssetServersResponse []AssetServersItem

// AssetServersItem type
//
// Not documented
type AssetServersItem struct {
	ServerID  int               `json:"ServerId"`
	Assets    Assets            `json:"Assets"`
	Groupings map[string]string `json:"Groupings"`
}

// Assets Type
//
// Not documented
type Assets struct {
	CPURatio             float64 `json:"CPURatio,omitempty"`
	EntityInternalID     string  `json:"EntityInternalId,omitempty"`
	HostID               int     `json:"HostID,omitempty"`
	ServerType           string  `json:"ServerType,omitempty"`
	BenchmarkScore       string  `json:"benchmarkScore,omitempty"`
	CoreMHzCapacity      string  `json:"coreMHzCapacity,omitempty"`
	Cores                string  `json:"cores,omitempty"`
	DecommisionDate      *Time   `json:"decommisionDate,omitempty"`
	DeployDate           *Time   `json:"deployDate,omitempty"`
	DeriveredCPUCapacity string  `json:"deriveredCPUCapacity,omitempty"`
	IOLimit              string  `json:"ioLimit,omitempty"`
	IOLimitActual        string  `json:"ioLimitActual,omitempty"`
	MemoryCapacity       string  `json:"memoryCapacity,omitempty"`
	ParentType           string  `json:"parentType,omitempty"`
	ServerName           string  `json:"serverName,omitempty"`
}

// AssetServers request
func (i *ICP) AssetServers(ctx context.Context, request *AssetServersRequest) (response AssetServersResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetServersEndpoint, request, &response)

	return
}

// AssetStorageRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-storage_projectId_baselineId
type AssetStorageRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId,omitempty"`
}

// AssetStorageResponse type
// XXX Unknown response from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-storage_projectId_baselineId
type AssetStorageResponse struct {
	DataStores []DataStore `json:"DataStore"`
	VMFiles    []VMFile    `json:"VMFiles"`
}

// DataStore type
//
// Not documented
type DataStore struct {
	CapacityMB       float64 `json:"capacityMB"`
	DataCenter       string  `json:"dataCenter"`
	DataStoreCluster string  `json:"datastoreCluster"`
	DataStoreID      int     `json:"datastoreID"`
	DataStoreName    string  `json:"datastoreName"`
	EntitySource     string  `json:"entitySource"`
	InternalID       string  `json:"internalId"`
	ProvisionedMB    float64 `json:"provisionedMB"`
	ReservedRatio    float64 `json:"reserveRatio"`
	Shared           bool    `json:"shared"`
	UsedMB           float64 `json:"usedMB"`
}

// VMFile type
//
// Not documented
type VMFile struct {
	CapacityMB      float64 `json:"capacityMB"`
	DataStoreID     int     `json:"datastoreID"`
	DemandMB        float64 `json:"demandMB"`
	EntityID        int     `json:"entityID"`
	FileSizeMB      float64 `json:"fileSizeMB"`
	Filename        string  `json:"filename"`
	HasTrueDemand   bool    `json:"hasTrueDemand"`
	StorageFileType int     `json:"storageFileType"`
	StorageFormat   int     `json:"storageFormat"`
	VMFileID        int     `json:"vmFileID"`
}

// AssetStorage request
func (i *ICP) AssetStorage(ctx context.Context, request *AssetStorageRequest) (response AssetStorageResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetStorageEndpoint, request, &response)
	return
}

// AssetGroupingsRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings_projectId_baselineId
type AssetGroupingsRequest struct {
	ProjectID  int `url:"projectId"`
	BaselineID int `url:"baselineId,omitempty"`
}

// AssetGroupingsResponse type
// XXX Unknown response from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings_projectId_baselineId
type AssetGroupingsResponse []string

// AssetGroupings request
func (i *ICP) AssetGroupings(ctx context.Context, request *AssetGroupingsRequest) (response AssetGroupingsResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetGroupingsEndpoint, request, &response)
	return
}

// AssetGroupingsGroupingRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-grouping_projectId_groupingName_baselineId
type AssetGroupingsGroupingRequest struct {
	ProjectID    int    `url:"projectId"`
	GroupingName string `url:"groupingName"`
	BaselineID   int    `url:"baselineId,omitempty"`
}

// AssetGroupingsGroupingResponse type
// XXX Unknown response from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-grouping_projectId_groupingName_baselineId
type AssetGroupingsGroupingResponse []string

// AssetGroupingsGrouping request
func (i *ICP) AssetGroupingsGrouping(ctx context.Context, request *AssetGroupingsGroupingRequest) (response AssetGroupingsGroupingResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetGroupingsGroupingEndpoint, request, &response)
	return
}

// AssetGroupingsEntityRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-entity_projectId_entityId_baselineId
type AssetGroupingsEntityRequest struct {
	ProjectID  int `url:"projectId"`
	EntityID   int `url:"entityId"`
	BaselineID int `url:"baselineId,omitempty"`
}

// AssetGroupingsEntityResponse type
// XXX Unknown response from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-entity_projectId_entityId_baselineId
type AssetGroupingsEntityResponse []string

// AssetGroupingsDynamicRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-dynamic_projectId_summaryDate_groupingName_baselineId_entityId_summaryLevelID
type AssetGroupingsDynamicRequest struct {
	ProjectID      int    `url:"projectId"`
	SummaryDate    string `url:"summaryDate"`
	GroupingName   string `url:"groupingName"`
	BaselineID     int    `url:"baselineId,omitempty"`
	EntityID       string `url:"entityId"`
	SummaryLevelID int    `url:"summaryLevelID,omitempty"`
}

// AssetGroupingsDynamicResponse type
// XXX Unknown encoding from docs
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-dynamic_projectId_summaryDate_groupingName_baselineId_entityId_summaryLevelID
type AssetGroupingsDynamicResponse []string

// AssetRequest type
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-asset
type AssetRequest []AssetRegisterItem

// AssetRegisterItem type
//
// https://icp-api.itrsgroup.com/v2.0/Help/ResourceModel?modelName=AssetRegisterItem
type AssetRegisterItem struct {
	ServerName                   string     `json:"ServerName,omitempty"`
	TimeStamp                    *Time      `json:"TimeStamp,omitempty"`
	InternalID                   string     `json:"InternalID,omitempty"`
	SourceServer                 string     `json:"SourceServer,omitempty"`
	ServerType                   string     `json:"ServerType,omitempty"`
	HostName                     string     `json:"HostName,omitempty"`
	DataCentreName               string     `json:"DataCentreName,omitempty"`
	Cluster                      string     `json:"Cluster,omitempty"`
	Environment                  string     `json:"Environment,omitempty"`
	Hypervisor                   string     `json:"Hypervisor,omitempty"`
	HypervisorVersion            string     `json:"HypervisorVersion,omitempty"`
	OperatingSystem              string     `json:"OperatingSystem,omitempty"`
	MemoryMB                     int        `json:"MemoryMB,omitempty"`
	ClockSpeedMHz                int        `json:"ClockSpeedMHz,omitempty"`
	NumberOfPhysicalCoresOrvCPUs int        `json:"NumberOfPhysicalCoresOrvCPUs,omitempty"`
	HyperthreadingEnabled        bool       `json:"HyperthreadingEnabled,omitempty"`
	DeployedDate                 *Time      `json:"DeployedDate,omitempty"`
	DecommissionedDate           *Time      `json:"DecommissionedDate,omitempty"`
	FailoverServerName           string     `json:"FailoverServerName,omitempty"`
	CPURatio                     int        `json:"CPURatio,omitempty"`
	SpecintRate2006              int        `json:"specint_rate2006,omitempty"`
	MaximumIOKBS                 int        `json:"MaximumIOKBS,omitempty"`
	LogicalCoresPerPhysicalCore  int        `json:"LogicalCoresPerPhysicalCore,omitempty"`
	HardwareModel                string     `json:"HardwareModel,omitempty"`
	HardwareVendor               string     `json:"HardwareVendor,omitempty"`
	Groupings                    []Grouping `json:"Groupings,omitempty"`
}

// AssetGroupingsEntity request
func (i *ICP) AssetGroupingsEntity(ctx context.Context, request *AssetGroupingsEntityRequest) (response AssetGroupingsEntityResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetGroupingsEntityEndpoint, request, &response)
	return
}

// AssetGroupingsDynamic request
func (i *ICP) AssetGroupingsDynamic(ctx context.Context, request *AssetGroupingsDynamicRequest) (response AssetGroupingsDynamicResponse, resp *http.Response, err error) {
	resp, err = i.Get(ctx, AssetGroupingsDynamicEndpoint, request, &response)
	return
}

// Asset request
//
// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-asset
func (i *ICP) Asset(ctx context.Context, request *AssetRequest) (resp *http.Response, err error) {
	resp, err = i.Post(ctx, AssetEndpoint, request, nil)
	resp.Body.Close()
	return
}
