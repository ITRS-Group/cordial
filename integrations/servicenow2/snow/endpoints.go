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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

func AcceptEvent(c echo.Context) (err error) {
	var incident Record
	var ok bool

	// close any request body after we are done
	if c.Request().Body != nil {
		defer c.Request().Body.Close()
	}

	req := c.(*Context)
	cf := req.Conf

	table, err := getTableConfig(cf, req.Param("table"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid table %q", req.Param("table")))
	}

	if err = json.NewDecoder(c.Request().Body).Decode(&incident); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// if not given an explicit cmdb_ci then search based on config
	if _, ok = incident["cmdb_ci"]; !ok {
		if query, ok := incident["_cmdb_search"]; !ok {
			if incident["cmdb_ci"], ok = incident["_cmdb_ci_default"]; !ok {
				return echo.NewHTTPError(http.StatusNotFound, "must supply either a _cmdb_ci_default or a _cmdb_search parameter")
			}
		} else {
			// if incident["cmdb_ci"], err = LookupCmdbCI(cf, incident["_cmdb_table"], query, incident["_cmdb_ci_default"]); err != nil {
			if incident["cmdb_ci"], err = LookupCmdbCI(req, incident["_cmdb_table"], query, incident["_cmdb_ci_default"]); err != nil {
				return
			}
		}
	}

	if incident["cmdb_ci"] == "" {
		return echo.NewHTTPError(http.StatusNotFound, "cmdb_ci is empty or search resulted in no matches")
	}

	// correlation_id := incident["correlation_id"]

	lookup_map := map[string]string{
		"cmdb_ci":        incident["cmdb_ci"],
		"correlation_id": incident["correlation_id"],
	}
	incident_id, state, err := LookupRecord(req,
		config.LookupTable(lookup_map),
		config.Default(fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber", incident["cmdb_ci"], incident["correlation_id"])),
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
				delete(incident, k)
			}
		}

		// now remove all fields prefixed with an underscore
		maps.DeleteFunc(incident, func(e, _ string) bool { return strings.HasPrefix(e, "_") })

		// all include must exist, else error
		for _, i := range s.MustInclude {
			if _, ok := incident[i]; !ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("missing field %q", i))
			}
		}

	}

	if cf.GetBool(cf.Join("servicenow", "incident-user", "lookup")) {
		user := cf.GetString(cf.Join("servicenow", "username"))
		userfield := cf.GetString(cf.Join("servicenow", "incident-user", "field"), config.Default("caller_id"))
		if _, ok := incident[userfield]; ok {
			user = incident[userfield]
		}

		// basic validation of user
		if !userRE.MatchString(user) {
			return echo.NewHTTPError(http.StatusBadRequest, "username supplied is invalid")
		}

		// only lookup user after all defaults applied
		u, err := GetRecord(req, makeURLPath("sys_user", Fields("sys_id"), Query("user_name="+user), Limit(1)))
		if err != nil || len(u) == 0 {
			log.Error().Err(err).Msgf("user not found")
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		incident[userfield] = u["sys_id"]
	}

	// response := make(map[string]string)
	response := map[string]string{
		"host":       incident["host"],
		"event_type": "Incident",
	}

	if incident_id != "" {
		number, err := incident.UpdateRecord(req, incident_id)
		if err != nil {
			return err
		}
		response["action"] = "Updated"
		response["number"] = number
	} else if incident["update_only"] == "true" {
		response["action"] = "Ignored"
		return echo.NewHTTPError(http.StatusAccepted, "No Incident Created. 'update only' option set.")
	} else {
		// incident["correlation_id"] = correlation_id
		number, err := incident.CreateRecord(req)
		if err != nil {
			return err
		}
		response["action"] = "Created"
		response["number"] = number
	}

	return c.JSON(201, response)
}

// This is to get a list of all records opened by the service
// user. `format` is the output format, either json (the default) or
// `csv`.
func GetAllRecords(c echo.Context) (err error) {
	var user, format string

	cc := c.(*Context)
	cf := cc.Conf

	defer c.Request().Body.Close()

	qb := echo.QueryParamsBinder(c)

	err = qb.String("user", &user).BindError()
	if err != nil || user == "" {
		user = cf.GetString("servicenow.username")
	}

	err = qb.String("format", &format).BindError()
	if err != nil || format == "" {
		format = "json"
	}

	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("username %q supplied is invalid", user))
	}

	u, err := GetRecord(cc, makeURLPath("sys_user", Fields("sys_id"), Query("user_name="+user), Limit(1)))
	if err != nil || len(u) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("user %q not found in sys_user (and needed for lookup)", user))
	}

	q := fmt.Sprintf(`active=true^opened_by=%s`, u["sys_id"])

	tc, err := getTableConfig(cf, cc.Param("table"))
	if err != nil {
		return
	}
	if !tc.Query.Enabled {
		return echo.ErrNotFound
	}

	l, _ := GetRecords(cc, makeURLPath(tc.Name, Fields(strings.Join(tc.Query.ResponseFields, ",")), Query(q)))

	switch format {
	case "json":
		return c.JSON(200, l)
	case "csv":
		columns := tc.Query.ResponseFields
		csv := csv.NewWriter(c.Response())
		csv.Write(columns)
		c.Response().Header().Set(echo.HeaderContentType, "text/csv")
		c.Response().WriteHeader(http.StatusOK)
		for _, line := range l {
			var fields []string
			for _, col := range columns {
				fields = append(fields, line[col])
			}
			csv.Write(fields)
		}
		csv.Flush()
		c.Response().Flush()
		return nil
	default:
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unknown output format %q. Use `csv` or `json`", format))
	}
}
