package gwhub

const (
	// PingEndpoint is the GW Hub health-check endpoint.
	PingEndpoint = "/ping"
	// MetricsQueryEndpoint queries metrics for entities in a selected time range.
	MetricsQueryEndpoint = "/metrics/query"
	// MetricsAggregationsEndpoint lists aggregations accepted by metric queries.
	MetricsAggregationsEndpoint = "/metrics/aggregations"
	// QueryEventRecordsEndpoint queries event records from GW Hub.
	QueryEventRecordsEndpoint = "/events/recordsQuery"
	// EntitiesEndpoint queries entities.
	EntitiesEndpoint = "/entities"
	// EntitiesSummaryEndpoint queries entity summaries.
	EntitiesSummaryEndpoint = "/entities/summary"
	// EntitiesMetricsEndpoint queries the metrics available for entities.
	EntitiesMetricsEndpoint = "/entities/metrics"
	// EntitiesMetricsRowsEndpoint queries metric rows for entities.
	EntitiesMetricsRowsEndpoint = "/entities/metrics/rows"
	// EntitiesAttributesEndpoint queries entity attributes.
	EntitiesAttributesEndpoint = "/entities/attributes"
)
