package icp

const (
	// LoginEndpoint is the path used to authenticate with the ICP API.
	LoginEndpoint = "/api/login" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-login

	// EntityPropertiesEndpoint retrieves or updates entity properties.
	EntityPropertiesEndpoint = "/api/entityproperties" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entityproperties

	// ProjectsEndpoint lists ICP projects.
	ProjectsEndpoint = "/api/projects" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects
	// ProjectsSolutionIDEndpoint retrieves a project's solution ID.
	ProjectsSolutionIDEndpoint = "/api/projects/solutionid"
	// ProjectsModelsEndpoint retrieves the models for a project baseline.
	ProjectsModelsEndpoint = "/api/projects/models" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
	// ProjectsIgnoreListEndpoint retrieves the ignore list for a project baseline.
	ProjectsIgnoreListEndpoint = "/api/projects/ignorelist" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId

	// AssetServersEndpoint retrieves servers for a project baseline.
	AssetServersEndpoint = "/api/asset/servers" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-servers_projectId_baselineId
	// AssetStorageEndpoint retrieves storage assets for a project baseline.
	AssetStorageEndpoint = "/api/asset/storage" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-storage_projectId_baselineId
	// AssetGroupingsEndpoint retrieves asset groupings for a project baseline.
	AssetGroupingsEndpoint = "/api/asset/groupings" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings_projectId_baselineId
	// AssetGroupingsGroupingEndpoint retrieves a named asset grouping.
	AssetGroupingsGroupingEndpoint = "/api/asset/groupings/grouping" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-grouping_projectId_groupingName_baselineId
	// AssetGroupingsEntityEndpoint retrieves groupings for an entity.
	AssetGroupingsEntityEndpoint = "/api/asset/groupings/entity" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-entity_projectId_entityId_baselineId
	// AssetGroupingsDynamicEndpoint retrieves dynamically generated groupings.
	AssetGroupingsDynamicEndpoint = "/api/asset/groupings/dynamic" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-dynamic_projectId_summaryDate_groupingName_baselineId_entityId_summaryLevelID
	// AssetEndpoint submits asset data to the ICP API.
	AssetEndpoint = "/api/asset" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-asset

	// BaselineViewsProjectEndpoint lists baseline views for a project.
	BaselineViewsProjectEndpoint = "/api/baselineviews/project" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-project-projectId
	// BaselineViewsEndpoint retrieves a baseline view by ID.
	BaselineViewsEndpoint = "/api/baselineviews" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId

	// EntityMetaDataEndpoint submits entity metadata to the ICP API.
	EntityMetaDataEndpoint = "/api/entitymetadata" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entitymetadata
	// EntityMetaDataExportEndpoint exports project entity metadata.
	EntityMetaDataExportEndpoint = "/api/metadataexport" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metadataexport-projectId_onlyInclude

	// MetricsEndpoint retrieves metrics for a project baseline.
	MetricsEndpoint = "/api/metrics" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metrics_projectId_baselineId
	// MetricsSummariesEndpoint retrieves metric summaries.
	MetricsSummariesEndpoint = "/api/metrics/summaries" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
	// MetricsSummariesDateRangeEndpoint retrieves metric summaries over a date range.
	MetricsSummariesDateRangeEndpoint = "/api/metrics/summariesdaterange" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summariesdaterange

	// EventsBaselineViewEndpoint queries events for a baseline view.
	EventsBaselineViewEndpoint = "/api/events/baselineview" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview

	// EventsEventFilterEndpoint creates, lists, or deletes project event filters.
	EventsEventFilterEndpoint = "/api/events/eventfilter"

	// UploadEndpoint uploads a task log or artifact to the ICP API.
	UploadEndpoint = "/api/Upload" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-Upload_version_taskname_machinename_selectedProjectId_testOnly_log

	// CreateStackEndpoint TODO: Set the API path when stack creation is implemented.
	CreateStackEndpoint = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createstack
	// CreateServiceAccountEndpoint TODO: Set the API path when account creation is implemented.
	CreateServiceAccountEndpoint = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createserviceaccount
	// DeleteServiceAccountEndpoint TODO: Set the API path when account deletion is implemented.
	DeleteServiceAccountEndpoint = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-deleteserviceaccount
	// CreatePipelineServiceAndDataFoldersEndpoint TODO: Set the API path when pipeline creation is implemented.
	CreatePipelineServiceAndDataFoldersEndpoint = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createpipelineserviceanddatafolders

	// RecommendationsCloudInstancesEndpoint requests cloud-instance recommendations.
	RecommendationsCloudInstancesEndpoint = "/api/recommendations/cloudinstances" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-recommendations-cloudinstances

	// DataMartEntityPerformanceEndpoint queries entity performance from the Data Mart.
	DataMartEntityPerformanceEndpoint = "/Api/DataMart/EntityPerformance" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
	// DataMartEntityRelationEndpoint queries entity relationships from the Data Mart.
	DataMartEntityRelationEndpoint = "/Api/DataMart/EntityRelation" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
	// DataMartPropertiesEntityEndpoint queries entity properties from the Data Mart.
	DataMartPropertiesEntityEndpoint = "/Api/DataMart/PropertiesEntity" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
	// DataMartMetricsEndpoint queries metrics from the Data Mart.
	DataMartMetricsEndpoint = "/Api/DataMart/Metrics" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
	// DataMartMetricTimeseriesEndpoint queries metric time series from the Data Mart.
	DataMartMetricTimeseriesEndpoint = "/Api/DataMart/MetricTimeseries" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
	// DataMartEntityPropertiesEndpoint queries entity properties from the Data Mart.
	DataMartEntityPropertiesEndpoint = "/Api/DataMart/EntityProperties" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
	// DataMartMetricCapacitiesEndpoint queries metric capacities from the Data Mart.
	DataMartMetricCapacitiesEndpoint = "/Api/DataMart/MetricCapacities" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
	// DataMartGetEntitiesEndpoint queries entities from the Data Mart.
	DataMartGetEntitiesEndpoint = "/Api/DataMart/GetEntities" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
	// DataMartStartProcessingEndpoint starts Data Mart processing.
	DataMartStartProcessingEndpoint = "/Api/DataMart/StartProcessing" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
)
