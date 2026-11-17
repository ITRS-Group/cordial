/*
Copyright © 2025 ITRS Group

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

package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/ims/snow"
)

const (
	// ServiceNow fields
	SNOW_CMDB_CI = "cmdb_ci"
	// SNOW_CORRELATION_ID TODO: Describe its purpose and intended use.
	SNOW_CORRELATION_ID = "correlation_id"
	// SNOW_SYS_ID TODO: Describe its purpose and intended use.
	SNOW_SYS_ID = "sys_id"
	// SNOW_USER_NAME TODO: Describe its purpose and intended use.
	SNOW_USER_NAME = "user_name"

	// ServiceNow tables
	SNOW_SYS_USER_TABLE = "sys_user"
	// SNOW_INCIDENT_TABLE TODO: Describe its purpose and intended use.
	SNOW_INCIDENT_TABLE = "incident"
	// SNOW_CMDB_TABLE TODO: Describe its purpose and intended use.
	SNOW_CMDB_TABLE = "cmdb_ci"

	// internal fields
	CMDB_CI_DEFAULT = "_cmdb_ci_default"
	// CMDB_SEARCH TODO: Describe its purpose and intended use.
	CMDB_SEARCH = "_cmdb_search"
	// CMDB_TABLE TODO: Describe its purpose and intended use.
	CMDB_TABLE = "_cmdb_table"
	// INCIDENT_TABLE TODO: Describe its purpose and intended use.
	INCIDENT_TABLE = "_table"
	// PROFILE TODO: Describe its purpose and intended use.
	PROFILE = "_profile"
	// RAW_CORRELATION_ID TODO: Describe its purpose and intended use.
	RAW_CORRELATION_ID = "_correlation_id"
	// UPDATE_ONLY TODO: Describe its purpose and intended use.
	UPDATE_ONLY = "_update_only"
)

// ResultsResponse TODO: Describe its purpose and intended use.
type ResultsResponse struct {
	Fields  []string     `json:"fields,omitempty"`
	Results snow.Results `json:"results,omitempty"`
}

// getRecords is the handler to get a list of all records that match
// `query`, which is usually for a user or caller_id. The `fields` query
// parameter is a comma separated list of fields to return and `raw`
// turns off display values.
func getRecords(cf *config.Config, w http.ResponseWriter, r *http.Request) {
	tc, err := snow.TableConfig(cf.Sub("servicenow"), r.PathValue("table"))
	if err != nil {
		if errors.Is(err, snow.ErrTableNotFound) {
			writeError(w, r, &ims.HTTPError{Code: http.StatusNotFound, Err: err})
			return
		}
		writeError(w, r, err)
		return
	}
	if !tc.Query.Enabled {
		writeError(w, r, &ims.HTTPError{Code: http.StatusNotFound})
		return
	}

	params := r.URL.Query()

	query := params.Get("query")
	if query == "" {
		query = "name=" + config.Get[string](cf, "servicenow.username")
	}

	fields := params.Get("fields")
	if fields == "" {
		fields = strings.Join(tc.Query.ResponseFields, ",")
	}

	// raw defaults to false, including if it cannot be parsed
	raw, _ := strconv.ParseBool(params.Get("raw"))

	// validate fields
	if !snow.ValidateFields(strings.Split(fields, ",")) {
		writeError(w, r, &ims.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("requested field names are invalid or not unique: %q", fields)})
		return
	}

	// real basic validation of query
	if !snow.QueryRE.MatchString(query) {
		writeError(w, r, &ims.HTTPError{Code: http.StatusBadRequest, Message: fmt.Sprintf("query %q supplied is invalid", query)})
		return
	}

	c, err := snow.New(cf.Sub("servicenow"))
	if err != nil {
		writeError(w, r, err)
		return
	}

	results, err := c.Records(r.Context(), tc.Name, snow.Fields(fields), snow.Query(query), snow.Display(fmt.Sprint(!raw)))
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, r, http.StatusOK, ResultsResponse{
		Fields:  strings.Split(fields, ","),
		Results: results,
	})
}

const (
	// Failed TODO: Describe its purpose and intended use.
	Failed = "Failed"
	// Created TODO: Describe its purpose and intended use.
	Created = "Created"
	// Updated TODO: Describe its purpose and intended use.
	Updated = "Updated"
	// Ignored is the action when the request was valid but deliberately
	// not acted on
	Ignored = "Ignored"
)

// Response TODO: Describe its purpose and intended use.
type Response struct {
	Timestamp string `json:"_timestamp"`
	Action    string `json:"_action"`
	Error     string `json:"_error,omitempty"`
	Number    string `json:"_number,omitempty"`
	Result    string `json:"result"`
}

// acceptRecord is the main entry point for accepting a record from
// the ServiceNow proxy. It expects a JSON body with the record to
// create or update. It will look up the record based on the cmdb_ci
// and correlation_id fields, and if it exists, it will update it.
func acceptRecord(cf *config.Config, w http.ResponseWriter, r *http.Request) {
	var incident ims.Values
	var ok bool
	var err error

	tableName := r.PathValue("table")

	// set up the response, anticipate failure until we know otherwise
	response := &Response{
		Timestamp: time.Now().Local().Format(time.RFC3339),
		Action:    Failed,
	}

	table, err := snow.TableConfig(cf.Sub("servicenow"), tableName)
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, snow.ErrTableNotFound) {
			code = http.StatusBadRequest
		}
		response.Error = fmt.Sprintf("error retrieving table configuration for %q: %v", tableName, err)
		response.Result = fmt.Sprintf("%s Error retrieving table configuration for %q", response.Timestamp, tableName)
		respond(w, r, code, response)
		return
	}

	// fail sets the error and expands the configured failure response
	// using the current incident fields
	fail := func(code int, msg string) {
		response.Error = msg
		response.Result = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}, incident),
			config.TrimSpace(false),
		)
		respond(w, r, code, response)
	}

	c, err := snow.New(cf.Sub("servicenow"))
	if err != nil {
		fail(http.StatusInternalServerError, err.Error())
		return
	}

	// set up defaults
	incident = maps.Clone(table.Defaults)
	if incident == nil {
		incident = make(ims.Values)
	}

	if err = json.NewDecoder(r.Body).Decode(&incident); err != nil {
		code := http.StatusBadRequest
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			code = http.StatusRequestEntityTooLarge
		}
		fail(code, fmt.Sprintf("error decoding request body: %v", err))
		return
	}

	// validate incident field names
	if !snow.ValidateFields(slices.Collect(maps.Keys(incident))) {
		fail(http.StatusBadRequest, "field names are invalid or not unique")
		return
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident[SNOW_CMDB_CI]; !ok {
		if query, ok := incident[CMDB_SEARCH]; !ok {
			if incident[SNOW_CMDB_CI], ok = incident[CMDB_CI_DEFAULT]; !ok {
				fail(http.StatusNotFound, "must supply either a _cmdb_ci_default or a _cmdb_search parameter")
				return
			}
		} else {
			if _, ok = incident[CMDB_CI_DEFAULT]; !ok {
				incident[CMDB_CI_DEFAULT] = table.Defaults[CMDB_CI_DEFAULT]
			}
			if incident[SNOW_CMDB_CI], err = c.LookupCmdbCI(r.Context(), incident[CMDB_TABLE], query, incident[CMDB_CI_DEFAULT]); err != nil {
				fail(ims.StatusFor(err, http.StatusBadRequest), "failed to look up cmdb_ci: "+err.Error())
				return
			}
		}
	}

	if incident[SNOW_CMDB_CI] == "" {
		// if cmdb_ci is empty, we cannot continue
		fail(http.StatusNotFound, "cmdb_ci is empty or search resulted in no matches")
		return
	}

	// without a correlation ID the search matches any active incident
	// for the CI, so an unrelated incident would be updated
	if incident[SNOW_CORRELATION_ID] == "" {
		fail(http.StatusBadRequest, "correlation_id is required (set correlation_id or _correlation_id)")
		return
	}

	lookup_map := map[string]string{
		SNOW_CMDB_CI:        incident[SNOW_CMDB_CI],
		SNOW_CORRELATION_ID: incident[SNOW_CORRELATION_ID],
	}
	search := config.Expand[string](cf, table.Search,
		config.LookupTable(lookup_map),
		config.DefaultValue(fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber", incident[SNOW_CMDB_CI], incident[SNOW_CORRELATION_ID])),
	)
	incident_id, state, err := c.LookupRecord(r.Context(), tableName, search)
	if err != nil {
		fail(ims.StatusFor(err, http.StatusInternalServerError), "error looking up incident: "+err.Error())
		return
	}

	// apply the transformation for the state of the existing incident,
	// or the default for any state that is not otherwise known. This is
	// not the same as a state of 0, which means the incident is new.
	//
	// fields has all the fields, including the internal ones with a
	// leading underscore, for use in the messages returned, and the
	// incident to send to ServiceNow has the internal ones removed
	fields := maps.Clone(incident)
	if transform, ok := table.State(state); ok {
		transformed, err := transform.TransformKeepInternal(cf, "snow", incident)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ims.ErrInvalidPattern) {
				// a configuration error, not a problem with the request
				status = http.StatusInternalServerError
			}
			fail(status, err.Error())
			return
		}
		incident = transformed
		maps.Copy(fields, transformed)
	}
	ims.RemoveInternal(incident, "_")
	incidentFields := fields

	// from here on failures can refer to all of the fields, including
	// the internal ones
	fail = func(code int, msg string) {
		response.Error = msg
		response.Result = config.Expand[string](cf,
			table.Response.Failed,
			config.LookupTable(map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}, incidentFields),
			config.TrimSpace(false),
		)
		respond(w, r, code, response)
	}

	if incident_id != "" {
		number, err := c.Update(r.Context(), tableName, incident_id, incident)
		if err != nil {
			fail(ims.StatusFor(err, http.StatusInternalServerError), fmt.Sprintf("error updating incident: %v", err))
			return
		}
		response.Action = Updated
		response.Number = number
		response.Result = config.Expand[string](cf,
			table.Response.Updated,
			config.LookupTable(incidentFields, map[string]string{
				"_number":    number,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		respond(w, r, http.StatusOK, response)
		return
	}

	if updateOnly, err := strconv.ParseBool(incidentFields[UPDATE_ONLY]); err == nil && updateOnly {
		response.Action = Ignored
		response.Result = "No Incident Created. '" + UPDATE_ONLY + "' set."
		respond(w, r, http.StatusOK, response)
		return
	}

	number, err := c.Create(r.Context(), tableName, incident)
	if err != nil {
		fail(ims.StatusFor(err, http.StatusInternalServerError), fmt.Sprintf("error creating incident: %v", err))
		return
	}
	response.Action = Created
	response.Number = number
	response.Result = config.Expand[string](cf,
		table.Response.Created,
		config.LookupTable(incidentFields, map[string]string{
			"_number":    number,
			"_timestamp": response.Timestamp,
		}),
		config.TrimSpace(false),
	)
	respond(w, r, http.StatusCreated, response)
}
