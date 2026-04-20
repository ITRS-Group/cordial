/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ims

// ServiceNow specific code, which is currently only used for the "snow"
// profile in imscmd raise, but may be extended to other areas in
// future.

const (
	// ServiceNow fields
	SNOW_CMDB_CI_FIELD     = "cmdb_ci"
	SNOW_CORRELATION_FIELD = "correlation_id"
	SNOW_SYS_ID_FIELD      = "sys_id"
	SNOW_USER_NAME_FIELD   = "user_name"

	// ServiceNow tables
	SNOW_SYS_USER_TABLE_DEFAULT = "sys_user"
	SNOW_INCIDENT_TABLE_DEFAULT = "incident"
	SNOW_CMDB_TABLE_DEFAULT     = "cmdb_ci"

	// internal fields
	SNOW_CORRELATION     = "__snow_correlation"
	SNOW_CMDB_CI_DEFAULT = "__snow_cmdb_ci_default"
	SNOW_CMDB_SEARCH     = "__snow_cmdb_search"
	SNOW_CMDB_TABLE      = "__snow_cmdb_table"
	SNOW_INCIDENT_TABLE  = "__snow_incident_table"
)

// snowResult is the response from ServiceNow. It contains the results
// of the request, which is a slice of results. It also contains an
// error message if the request failed. The status field is used to
// indicate the status of the request. If the request was successful,
// the status will be "success". If the request failed, the status will
// be "error". The error field contains the error message and detail if
// the request failed. The results field contains the results of the
// request.
type SnowResult struct {
	Results Results `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status,omitempty"`
}

type SnowResultsResponse struct {
	Fields  []string `json:"fields,omitempty"`
	Results Results  `json:"results,omitempty"`
}
