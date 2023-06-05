package icp

const (
	LoginEndpoint = "/api/login" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-login

	EntityPropertiesEndpoint = "/api/entityproperties" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-entityproperties

	ProjectsEndpoint           = "/api/projects"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects
	ProjectsModelsEndpoint     = "/api/projects/models"     // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-models_projectId_baselineId
	ProjectsIgnoreListEndpoint = "/api/projects/ignorelist" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-projects-ignorelist_projectId_baselineId

	BaselineViewsProjectEndpoint = "/api/baselineviews/project" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-project-projectId
	BaselineViewsEndpoint        = "/api/baselineviews"         // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-baselineviews-baselineViewId

	AssetServersEndpoint           = "/api/asset/servers"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-servers_projectId_baselineId
	AssetStorageEndpoint           = "/api/asset/storage"            // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-storage_projectId_baselineId
	AssetGroupingsEndpoint         = "/api/asset/groupings"          // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings_projectId_baselineId
	AssetGroupingsGroupingEndpoint = "/api/asset/groupings/grouping" // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-grouping_projectId_groupingName_baselineId
	AssetGroupingsEntityEndpoint   = "/api/asset/groupings/entity"   // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-entity_projectId_entityId_baselineId
	AssetGroupingsDynamicEndpoint  = "/api/asset/groupings/dynamic"  // https://icp-api.itrsgroup.com/v2.0/Help/Api/GET-api-asset-groupings-dynamic_projectId_summaryDate_groupingName_baselineId_entityId_summaryLevelID
	AssetEndpoint                  = "/api/asset"                    // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-api-asset

	DataMartEntityPerformanceEndpoint = "/api/DataMart/EntityPerformance" // https://icp-api.itrsgroup.com/v2.0/Help/Api/POST-Api-DataMart-EntityPerformance
)
