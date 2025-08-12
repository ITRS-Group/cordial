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
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
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
	Fields  []string     `json:"fields,omitempty"`
	Results snow.Results `json:"results,omitempty"`
}

// not a complete test, but just filter characters *allowed* according to the ServiceNow
// documentation. This is not a complete test, but it should catch most
// of the invalid queries.
var queryRE = regexp.MustCompile(`^[\w\.,@=^!%*<> ]+$`)

// This is to get a list of all records that match `query`, which is
// usually for a user or caller_id. `format` is the output format,
// either json (the default) or `csv`. `fields` is a comma separated
// list of fields to return.
func getRecords(c echo.Context) (err error) {
	var query string
	var fields string
	var raw bool

	cc := c.(*snow.Context)
	cf := cc.Conf

	defer c.Request().Body.Close()

	tc, err := snow.TableConfig(cf, cc.Param("table"))
	if err != nil {
		return
	}
	if !tc.Query.Enabled {
		return echo.ErrNotFound
	}

	qb := echo.QueryParamsBinder(c)

	err = qb.String("query", &query).BindError()
	if err != nil || query == "" {
		query = "name=" + cf.GetString("servicenow.username")
	}

	err = qb.String("fields", &fields).BindError()
	if err != nil || fields == "" {
		fields = strings.Join(tc.Query.ResponseFields, ",")
	}

	qb.Bool("raw", &raw) // defaults to false regardless

	// validate fields
	if !validateFields(strings.Split(fields, ",")) {
		return echo.NewHTTPError(http.StatusBadRequest, "requested field names are invalid or not unique: %q", fields)
	}

	// real basic validation of query
	if !queryRE.MatchString(query) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("query %q supplied is invalid", query))
	}

	results, err := snow.GetRecords(cc, tc.Name, snow.Fields(fields), snow.Query(query), snow.Display(fmt.Sprint(!raw)))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ResultsResponse{
		Fields:  strings.Split(fields, ","),
		Results: results,
	})
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
	l2 := len(slices.Compact(keys))
	if l1 != l2 {
		return false
	}
	return true
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
func acceptRecord(c echo.Context) (err error) {
	var incident snow.Record
	var ok bool

	req := c.(*snow.Context)
	cf := req.Conf

	// set up the response, anticipate failure until we know otherwise
	response := &Response{
		Timestamp: time.Now().Local().Format(time.RFC3339),
		Action:    "Failed",
	}

	// close any request body after we are done
	if c.Request().Body != nil {
		defer c.Request().Body.Close()
	}

	table, err := snow.TableConfig(cf, req.Param("table"))
	if err != nil {
		response.Result = cf.ExpandString(
			fmt.Sprintf("${_timestamp} Error retrieving table configuration for %q", req.Param("table")),
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		return c.JSON(http.StatusBadRequest, response)
	}

	// set up defaults
	incident = maps.Clone(table.Defaults)

	if err = json.NewDecoder(c.Request().Body).Decode(&incident); err != nil {
		response.Error = fmt.Sprintf("error decoding request body: %v", err)
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.TrimSpace(false),
		)
		return c.JSON(http.StatusBadRequest, response)
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
		return c.JSON(http.StatusBadRequest, response)
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
				return c.JSON(http.StatusNotFound, response)
			}
		} else {
			if _, ok = incident[CMDB_CI_DEFAULT]; !ok {
				incident[CMDB_CI_DEFAULT] = table.Defaults[CMDB_CI_DEFAULT]
			}
			if incident[SNOW_CMDB_CI], err = snow.LookupCmdbCI(req, incident[CMDB_TABLE], query, incident[CMDB_CI_DEFAULT]); err != nil {
				response.Error = "failed to look up cmdb_ci: " + err.Error()
				response.Result = cf.ExpandString(
					table.Response.Failed,
					config.LookupTable(incident, map[string]string{
						"_error":     response.Error,
						"_timestamp": response.Timestamp,
					}),
					config.TrimSpace(false),
				)
				return c.JSON(http.StatusBadRequest, response)
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
		// if cmdb_ci is empty, we cannot continue
		return c.JSON(http.StatusNotFound, response)
	}

	lookup_map := map[string]string{
		SNOW_CMDB_CI:        incident[SNOW_CMDB_CI],
		SNOW_CORRELATION_ID: incident[SNOW_CORRELATION_ID],
	}
	incident_id, state, err := snow.LookupRecord(req,
		config.LookupTable(lookup_map),
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
		return c.JSON(http.StatusNotFound, response)
	}

	// first apply defaults, then remove excluded fields, apply renaming, check must-includes
	s, ok := table.CurrentStates[state]

	// try to fall back to the default state if we do not have a known
	// current state for the incident. This is useful for example when
	// the incident is in a state that we do not know about, such as a
	// custom state. It is NOT the same as a current state of 0, which
	// means incident is new.
	if !ok {
		s, ok = table.CurrentStates[-1] // -1 is the unknown state
	}

	if ok {
		// add default fields if they are not already set (or empty)
		for k, v := range s.Defaults {
			// log.Debug().Msgf("checking field %q: value %q - would set to %q", k, incident[k], v)
			if i, ok := incident[k]; !ok || i == "" {
				incident[k] = cf.ExpandString(v)
			}
		}

		// delete excluded
		for _, e := range s.Remove {
			delete(incident, e)
		}

		// no clash checking
		for k, v := range s.Rename {
			if _, ok := incident[k]; ok {
				incident[v] = incident[k]
				if strings.HasPrefix(k, "_") && !strings.HasPrefix(v, "_") {
					// do not delete src is it starts with an underscore and the dst does not
					continue
				}
				delete(incident, k)
			}
		}

		// all include must exist, else error
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
				// if a must-include field is missing, we cannot continue
				return c.JSON(http.StatusBadRequest, response)
			}
		}

		// Filter, if defined, is a slice of regexps to filter incident
		// fields. Only fields matching the filters will be retained.
		if len(s.Filter) > 0 {
			for key := range incident {
				if !slices.ContainsFunc(s.Filter, func(f string) bool {
					match, _ := regexp.MatchString(f, key) // ignore error as this would be a failed match anyway
					return match
				}) {
					delete(incident, key)
				}
			}
		}
	}

	// now remove all fields prefixed with an underscore, but leaves
	// them in incidentFields for later
	incidentFields := maps.Clone(incident)
	maps.DeleteFunc(incident, func(e, _ string) bool { return strings.HasPrefix(e, "_") })

	if incident_id != "" {
		number, err := incident.UpdateRecord(req, incident_id)
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
			return c.JSON(http.StatusInternalServerError, response)
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
		return c.JSON(http.StatusOK, response)
	}

	if incident[UPDATE_ONLY] == "true" {
		response.Action = "Ignored"
		response.Result = "No Incident Created. 'update only' option set."
		return c.JSON(http.StatusOK, response)
	}

	number, err := incident.CreateRecord(req)
	if err != nil {
		response.Error = fmt.Sprintf("error creating incident: %v", err)
		response.Result = cf.ExpandString(
			table.Response.Failed,
			config.LookupTable(incident, map[string]string{
				"_error":     response.Error,
				"_timestamp": response.Timestamp,
			}),
			config.LookupTable(incidentFields),
			config.TrimSpace(false),
		)
		return c.JSON(http.StatusInternalServerError, response)
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
	return c.JSON(http.StatusCreated, response)
}
