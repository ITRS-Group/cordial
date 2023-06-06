package icp

const (
	LoginEndpoint = "/api/login" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-login

	EntityPropertiesEndpoint = "/api/entityproperties" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entityproperties

	ProjectsEndpoint           = "/api/projects"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects
	ProjectsModelsEndpoint     = "/api/projects/models"     // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
	ProjectsIgnoreListEndpoint = "/api/projects/ignorelist" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId

	AssetServersEndpoint           = "/api/asset/servers"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-servers_projectId_baselineId
	AssetStorageEndpoint           = "/api/asset/storage"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-storage_projectId_baselineId
	AssetGroupingsEndpoint         = "/api/asset/groupings"          // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings_projectId_baselineId
	AssetGroupingsGroupingEndpoint = "/api/asset/groupings/grouping" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-grouping_projectId_groupingName_baselineId
	AssetGroupingsEntityEndpoint   = "/api/asset/groupings/entity"   // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-entity_projectId_entityId_baselineId
	AssetGroupingsDynamicEndpoint  = "/api/asset/groupings/dynamic"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-dynamic_projectId_summaryDate_groupingName_baselineId_entityId_summaryLevelID
	AssetEndpoint                  = "/api/asset"                    // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-asset

	BaselineViewsProjectEndpoint = "/api/baselineviews/project" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-project-projectId
	BaselineViewsEndpoint        = "/api/baselineviews"         // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId

	EntityMetaDataEndpoint       = "/api/entitymetadata" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entitymetadata
	EntityMetaDataExportEndpoint = "/api/metadataexport" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metadataexport-projectId_onlyInclude

	MetricsEndpoint                   = "/api/metrics"                    // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-metrics_projectId_baselineId
	MetricsSummariesEndpoint          = "/api/metrics/summaries"          // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summaries
	MetricsSummariesDateRangeEndpoint = "/api/metrics/summariesdaterange" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-metrics-summariesdaterange

	EventsBaselineViewEndpoint = "/api/events/baselineview" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-baselineview

	// EventsEventFilterEndpoint endpoint
	//
	// https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-events-eventfilter
	//
	// https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-events-eventfilter_projectId
	//
	// https://icp-api.itrsgroup.com/v2.0/Help/Api/DELETE-api-events-eventfilter_projectId
	EventsEventFilterEndpoint = "/api/events/eventfilter"

	UploadEndpoint = "/api/Upload" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-Upload_version_taskname_machinename_selectedProjectId_testOnly_log

	CreateStackEndpoint                         = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createstack
	CreateServiceAccountEndpoint                = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createserviceaccount
	DeleteServiceAccountEndpoint                = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-deleteserviceaccount
	CreatePipelineServiceAndDataFoldersEndpoint = "" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-createpipelineserviceanddatafolders

	RecommendationsCloudInstancesEndpoint = "/api/recommendations/cloudinstances" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-recommendations-cloudinstances

	DataMartEntityPerformanceEndpoint = "/Api/DataMart/EntityPerformance" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
	DataMartEntityRelationEndpoint    = "/Api/DataMart/EntityRelation"    // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityRelation
	DataMartPropertiesEntityEndpoint  = "/Api/DataMart/PropertiesEntity"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-PropertiesEntity
	DataMartMetricsEndpoint           = "/Api/DataMart/Metrics"           // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-Metrics
	DataMartMetricTimeseriesEndpoint  = "/Api/DataMart/MetricTimeseries"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricTimeseries
	DataMartEntityPropertiesEndpoint  = "/Api/DataMart/EntityProperties"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityProperties
	DataMartMetricCapacitiesEndpoint  = "/Api/DataMart/MetricCapacities"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-MetricCapacities
	DataMartGetEntitiesEndpoint       = "/Api/DataMart/GetEntities"       // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-GetEntities
	DataMartStartProcessingEndpoint   = "/Api/DataMart/StartProcessing"   // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-StartProcessing
)
