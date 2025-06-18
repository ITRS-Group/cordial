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

// not a complete test, but just filter characters *allowed*
var userRE = regexp.MustCompile(`^[\w\.@ ]+$`)

// This is to get a list of all records opened by the service
// user. `format` is the output format, either json (the default) or
// `csv`.
func getRecords(c echo.Context) (err error) {
	var user string
	var fields string

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

	err = qb.String("user", &user).BindError()
	if err != nil || user == "" {
		user = cf.GetString("servicenow.username")
	}

	err = qb.String("fields", &fields).BindError()
	if err != nil || fields == "" {
		fields = strings.Join(tc.Query.ResponseFields, ",")
	}

	// validate fields
	if !validateFields(strings.Split(fields, ",")) {
		return echo.NewHTTPError(http.StatusBadRequest, "one or more field names are invalid or not unique")
	}

	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("username %q supplied is invalid", user))
	}

	u, err := snow.GetRecord(cc, SNOW_SYS_USER_TABLE, snow.Limit(1), snow.Fields(SNOW_SYS_ID), snow.Query(SNOW_USER_NAME+"="+user))
	if err != nil || len(u) == 0 {
		return
	}

	q := fmt.Sprintf(`active=true^opened_by=%s`, u[SNOW_SYS_ID])

	results, err := snow.GetRecords(cc, tc.Name, snow.Fields(fields), snow.Query(q), snow.Display("true"))
	if err != nil {
		return err
	}

	return c.JSON(200, ResultsResponse{
		Fields:  strings.Split(fields, ","),
		Results: results,
	})
}

var snowField1 = regexp.MustCompile(`^[\w-]+$`)

// validateFields checks all the keys in the snow.Record incident.
// ServiceNow fields can consist of letters, numbers, underscored and
// hyphens and cannot begin or end with a hyphen. They cannot be an
// empty string either.
//
// it also lowercases all fields names and if there is a clash it
// returns false.
//
// if there are no invalid fields, the function returns true.
func validateFields(keys []string) bool {
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

func acceptRecord(c echo.Context) (err error) {
	var incident snow.Record
	var ok bool

	// close any request body after we are done
	if c.Request().Body != nil {
		defer c.Request().Body.Close()
	}

	req := c.(*snow.Context)
	cf := req.Conf

	table, err := snow.TableConfig(cf, req.Param("table"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid table %q", req.Param("table")))
	}

	if err = json.NewDecoder(c.Request().Body).Decode(&incident); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// validate incident field names
	if !validateFields(slices.Collect(maps.Keys(incident))) {
		return echo.NewHTTPError(http.StatusBadRequest, "one or more field names are invalid or not unique")
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident[SNOW_CMDB_CI]; !ok {
		if query, ok := incident[CMDB_SEARCH]; !ok {
			if incident[SNOW_CMDB_CI], ok = incident[CMDB_CI_DEFAULT]; !ok {
				return echo.NewHTTPError(http.StatusNotFound, "must supply either a _cmdb_ci_default or a _cmdb_search parameter")
			}
		} else {
			if _, ok = incident[CMDB_CI_DEFAULT]; !ok {
				incident[CMDB_CI_DEFAULT] = table.Defaults[CMDB_CI_DEFAULT]
			}
			if incident[SNOW_CMDB_CI], err = snow.LookupCmdbCI(req, incident[CMDB_TABLE], query, incident[CMDB_CI_DEFAULT]); err != nil {
				return
			}
		}
	}

	if incident[SNOW_CMDB_CI] == "" {
		return echo.NewHTTPError(http.StatusNotFound, "cmdb_ci is empty or search resulted in no matches")
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
		return
	}

	// first apply defaults, then remove excluded fields, apply renaming, check must-includes
	s, ok := table.CurrentStates[state]
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
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("missing field %q", i))
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

	response := map[string]string{}

	if incident_id != "" {
		number, err := incident.UpdateRecord(req, incident_id)
		if err != nil {
			return err
		}
		response["_action"] = "Updated"
		response["_number"] = number
		response["result"] = cf.ExpandString(
			table.Response.Updated,
			config.LookupTable(incidentFields, response),
			config.TrimSpace(false),
		)
		return c.JSON(http.StatusOK, response)
	}

	if incident[UPDATE_ONLY] == "true" {
		response["result"] = "No Incident Created. 'update only' option set."
		response["_action"] = "Ignored"
		return c.JSON(http.StatusOK, response)
	}

	number, err := incident.CreateRecord(req)
	if err != nil {
		return err
	}
	response["_action"] = "Created"
	response["_number"] = number
	response["result"] = cf.ExpandString(
		table.Response.Created,
		config.LookupTable(incidentFields, response),
		config.TrimSpace(false),
	)
	return c.JSON(http.StatusCreated, response)
}
