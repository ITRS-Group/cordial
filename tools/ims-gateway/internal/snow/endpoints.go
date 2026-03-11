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

package snow

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/rs/zerolog/log"
)

const (
	// ServiceNow fields
	SNOW_CMDB_CI        = "cmdb_ci"
	SNOW_CORRELATION_ID = "correlation_id"
	SNOW_SYS_ID         = "sys_id"
	SNOW_USER_NAME      = "user_name"

	// ServiceNow tables
	SNOW_SYS_USER_TABLE = "sys_user"
	SNOW_INCIDENT_TABLE = "incident"
	SNOW_CMDB_TABLE     = "cmdb_ci"

	// internal fields
	CMDB_CI_DEFAULT    = "_cmdb_ci_default"
	CMDB_SEARCH        = "_cmdb_search"
	CMDB_TABLE         = "_cmdb_table"
	INCIDENT_TABLE     = "_table"
	PROFILE            = "_profile"
	RAW_CORRELATION_ID = "_correlation_id"
	UPDATE_ONLY        = "_update_only"
)

type ResultsResponse struct {
	Fields  []string `json:"fields,omitempty"`
	Results results  `json:"results,omitempty"`
}

var Get, Post ims.Endpoint

func init() {
	// register endpoint for accepting records

	ims.RegisterEndpoint(
		http.MethodGet,
		"/snow/{table}",
		func(w http.ResponseWriter, r *http.Request) {
			getRecordsHTTP(w, r)
		},
	)

	ims.RegisterEndpoint(
		http.MethodPost,
		"/snow/{table}",
		func(w http.ResponseWriter, r *http.Request) {
			acceptRecordHTTP(w, r)
		},
	)
}

// not a complete test, but just filter characters *allowed* according
// to the ServiceNow documentation. This is not a complete test, but it
// should catch most of the invalid queries.
var queryRE = regexp.MustCompile(`^[\w\.,@=^!%*<> ]+$`)

// This is to get a list of all records that match `query`, which is
// usually for a user or caller_id. `format` is the output format,
// either json (the default) or `csv`. `fields` is a comma separated
// list of fields to return.
func getRecordsHTTP(w http.ResponseWriter, r *http.Request) {
	var response = &ims.Response{}

	log.Debug().Msgf("handling request for %s", r.URL.Path)

	v := r.Context().Value(ims.ContextKeyConfig)
	if v == nil {
		log.Debug().Msgf("config not found in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	cf, ok := v.(*config.Config)
	if !ok {
		log.Debug().Msgf("config not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	table := r.PathValue("table")
	if table == "" {
		log.Debug().Msgf("table name missing from request path")
		http.NotFound(w, r)
		return
	}

	tc, err := tableConfig(cf, table)
	if err != nil {
		response.Error = fmt.Sprintf("error retrieving table configuration for %q: %v", table, err)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	log.Debug().Msgf("table config %+v", tc)

	if !tc.Query.Enabled {
		http.NotFound(w, r)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		query = "name=" + cf.GetString(cf.Join("snow", "username"))
	}

	fields := r.URL.Query().Get("fields")
	if fields == "" {
		fields = strings.Join(tc.Query.ResponseFields, ",")
	}

	raw, _ := strconv.ParseBool(r.URL.Query().Get("raw"))

	// validate fields
	if !validateFields(strings.Split(fields, ",")) {
		response.Error = fmt.Sprintf("field names are invalid or not unique: %q", fields)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	// real basic validation of query
	if !queryRE.MatchString(query) {
		response.Error = fmt.Sprintf("query %q supplied is invalid", query)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	response, err = getRecords(r.Context(), cf.Sub("snow"), tc.Name, Fields(fields), Query(query), Display(fmt.Sprint(!raw)))
	if err != nil {
		response.Error = err.Error()
		ims.WriteJSON(w, http.StatusBadGateway, response)
		return
	}

	response.Status = http.StatusText(http.StatusOK)
	response.StatusCode = http.StatusOK

	ims.WriteJSON(w, http.StatusOK, response)
}

var snowField1 = regexp.MustCompile(`^[\w\.-]+$`)

// validateFields checks all the keys in the snow.Record incident.
// ServiceNow fields can consist of letters, numbers, underscored and
// hyphens and cannot begin or end with a hyphen. They cannot be an
// empty string either.
//
// it also lowercases all fields names and if there is a clash it
// returns false.
//
// if there are no keys the function returns false.
//
// if there are no invalid fields, the function returns true.
func validateFields(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// check keys are valid (we cannot use a single regexp to check for
	// non leading hyphen on a single char string)
	for _, k := range keys {
		if k == "" || strings.HasPrefix(k, "-") || strings.HasSuffix(k, "-") {
			return false
		}
		if !snowField1.MatchString(k) {
			return false
		}
	}

	// check keys are unique when lowercased
	//
	// NOTE: this could be skipped as viper lowercases all parameters
	// names from yaml, but we change future YAML parsers
	slices.SortFunc(keys, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	l1 := len(keys)
	l2 := len(slices.Compact(keys)) // slices.Compact modifies the slice

	return l1 == l2
}

const (
	Failed  = "Failed"
	Created = "Created"
	Updated = "Updated"
)

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
func acceptRecordHTTP(w http.ResponseWriter, r *http.Request) {
	var incident record
	var ok bool

	v := r.Context().Value(ims.ContextKeyConfig)
	if v == nil {
		log.Debug().Msgf("config not found in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	cf, ok := v.(*config.Config)
	if !ok {
		log.Debug().Msgf("config not correct type in request context")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tableName := r.PathValue("table")
	if tableName == "" {
		http.NotFound(w, r)
		return
	}

	// set up the response, anticipate failure until we know otherwise
	response := &Response{
		Timestamp: time.Now().Local().Format(time.RFC3339),
		Action:    "Failed",
	}

	// close any request body after we are done
	if r.Body != nil {
		defer r.Body.Close()
	}

	table, err := tableConfig(cf, tableName)
	if err != nil {
		response.Result = cf.ExpandString(
			fmt.Sprintf("${_timestamp} Error retrieving table configuration for %q", tableName),
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	// set up defaults
	incident = maps.Clone(table.Defaults)

	if err = json.NewDecoder(r.Body).Decode(&incident); err != nil {
		response.Error = fmt.Sprintf("error decoding request body: %v", err)
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(incident))) {
		response.Error = "field names are invalid or not unique"
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusBadRequest, response)
		return
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident[SNOW_CMDB_CI]; !ok {
		if query, ok := incident[CMDB_SEARCH]; !ok {
			if incident[SNOW_CMDB_CI], ok = incident[CMDB_CI_DEFAULT]; !ok {
				response.Error = "must supply either a _cmdb_ci_default or a _cmdb_search parameter"
				response.Result = cf.ExpandString(
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"_error":     response.Error,
						"_timestamp": response.Timestamp,
					}),
					config.TrimSpace(false),
				)
				ims.WriteJSON(w, http.StatusNotFound, response)
				return
			}
		} else {
			if _, ok = incident[CMDB_CI_DEFAULT]; !ok {
				incident[CMDB_CI_DEFAULT] = table.Defaults[CMDB_CI_DEFAULT]
			}
			if incident[SNOW_CMDB_CI], err = lookupCmdbCI(r.Context(), cf, incident[CMDB_TABLE], query, incident[CMDB_CI_DEFAULT]); err != nil {
				response.Error = "failed to look up cmdb_ci: " + err.Error()
				response.Result = cf.ExpandString(
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"_error":     response.Error,
						"_timestamp": response.Timestamp,
					}),
					config.TrimSpace(false),
				)
				ims.WriteJSON(w, http.StatusBadRequest, response)
				return
			}
		}
	}

	if incident[SNOW_CMDB_CI] == "" {
		response.Error = "cmdb_ci is empty or search resulted in no matches"
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusNotFound, response)
		return
	}

	lookupMap := map[string]string{
		SNOW_CMDB_CI:        incident[SNOW_CMDB_CI],
		SNOW_CORRELATION_ID: incident[SNOW_CORRELATION_ID],
	}

	incidentID, state, err := lookupRecord(
		r.Context(),
		cf,
		table.Name,
		config.LookupTable(lookupMap),
		config.Default(fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber", incident[SNOW_CMDB_CI], incident[SNOW_CORRELATION_ID])),
	)
	if err != nil {
		response.Error = "error looking up incident: " + err.Error()
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusNotFound, response)
		return
	}

	s, ok := table.CurrentStates[state]
	if !ok {
		s, ok = table.CurrentStates[-1] // unknown state fallback
	}

	if ok {
		for k, v := range s.Defaults {
			if i, ok := incident[k]; !ok || i == "" {
				incident[k] = cf.ExpandString(v)
			}
		}

		for _, e := range s.Remove {
			delete(incident, e)
		}

		for k, v := range s.Rename {
			if _, ok := incident[k]; ok {
				incident[v] = incident[k]
				if strings.HasPrefix(k, "_") && !strings.HasPrefix(v, "_") {
					continue
				}
				delete(incident, k)
			}
		}

		for _, i := range s.MustInclude {
			if _, ok := incident[i]; !ok {
				response.Error = fmt.Sprintf("missing required field %q", i)
				response.Result = cf.ExpandString(
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"_error":     response.Error,
						"_timestamp": response.Timestamp,
					}),
					config.TrimSpace(false),
				)
				ims.WriteJSON(w, http.StatusBadRequest, response)
				return
			}
		}

		if len(s.Filter) > 0 {
			for key := range incident {
				if !slices.ContainsFunc(s.Filter, func(f string) bool {
					match, _ := regexp.MatchString(f, key)
					return match
				}) {
					delete(incident, key)
				}
			}
		}
	}

	incidentFields := maps.Clone(incident)
	maps.DeleteFunc(incident, func(e, _ string) bool { return strings.HasPrefix(e, "_") })

	if incidentID != "" {
		number, err := incident.updateRecord(r.Context(), cf, table.Name, incidentID)
		if err != nil {
			response.Error = fmt.Sprintf("error updating incident: %v", err)
			response.Result = cf.ExpandString(
				table.Response.Failed,
				config.LookupTable(incident, map[string]string{
					"_error":     response.Error,
					"_timestamp": response.Timestamp,
				}),
				config.TrimSpace(false),
			)
			ims.WriteJSON(w, http.StatusInternalServerError, response)
			return
		}
		response.Action = "Updated"
		response.Number = number
		response.Result = cf.ExpandString(
			table.Response.Updated,
			config.LookupTable(incidentFields, map[string]string{
				"_number":    number,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusOK, response)
		return
	}

	if updateOnly, err := strconv.ParseBool(incidentFields[UPDATE_ONLY]); err == nil && updateOnly {
		response.Action = "Ignored"
		response.Result = "No Incident Created. '" + UPDATE_ONLY + "' set."
		ims.WriteJSON(w, http.StatusOK, response)
		return
	}

	number, err := incident.createRecord(r.Context(), cf, table.Name)
	if err != nil {
		response.Error = fmt.Sprintf("error creating incident: %v", err)
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incidentFields, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.LookupTable(incidentFields),
			config.TrimSpace(false),
		)
		ims.WriteJSON(w, http.StatusInternalServerError, response)
		return
	}
	response.Action = "Created"
	response.Number = number
	response.Result = cf.ExpandString(
		table.Response.Created,
		config.LookupTable(incidentFields, map[string]string{
			"_number":    number,
			"_timestamp": response.Timestamp,
		}),
		config.TrimSpace(false),
	)
	ims.WriteJSON(w, http.StatusCreated, response)
}
