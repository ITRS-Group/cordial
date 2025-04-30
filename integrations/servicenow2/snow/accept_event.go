/*
Copyright © 2022 ITRS Group

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
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

func AcceptEvent(c echo.Context) (err error) {
	var incident IncidentFields
	var ok bool
	var cmdb_ci string

	// close any request body after we are done
	if c.Request().Body != nil {
		defer c.Request().Body.Close()
	}

	rc := c.(*Context)
	cf := rc.Conf

	table, err := getTableConfig(cf, rc.Param("table"))
	if err != nil {
		return
	}

	err = json.NewDecoder(c.Request().Body).Decode(&incident)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	snow := InitializeConnection(cf)

	// if not given an explicit cmdb_ci then search based on config
	if cmdb_ci, ok = incident["cmdb_ci"]; !ok {
		if query, ok := incident["_cmdb_search"]; !ok {
			if cmdb_ci, ok = incident["_cmdb_ci_default"]; !ok {
				return echo.NewHTTPError(http.StatusNotFound, "must supply either a _cmdb_ci_default or a _cmdb_search parameter")
			}
		} else {
			if cmdb_ci, err = LookupCmdbCI(cf, incident["_cmdb_table"], query, incident["_cmdb_ci_default"]); err != nil {
				return
			}
		}
	}

	if cmdb_ci == "" {
		// incident["action"] = "Failed"
		return echo.NewHTTPError(http.StatusNotFound, "cmdb_ci is empty or search resulted in no matches")
	}

	correlation_id := incident["correlation_id"]

	response := make(map[string]string)

	incident_id, state, err := LookupIncident(rc, cmdb_ci, correlation_id)
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
		u, err := snow.GET(Limit("1"), Fields("sys_id"), Query("user_name="+user)).QueryTableSingle("sys_user")
		if err != nil || len(u) == 0 {
			log.Error().Err(err).Msgf("user not found")
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		incident[userfield] = u["sys_id"]
	}

	if incident_id != "" {
		incidentID, err := UpdateIncident(rc, incident_id, incident)
		if err != nil {
			return err
		}
		response["host"] = incident["host"]
		response["action"] = "Updated"
		response["number"] = incidentID
		response["event_type"] = "Incident"
	} else if incident["update_only"] == "true" {
		response["host"] = incident["host"]
		response["action"] = "Ignored"
		response["event_type"] = "Incident"
		return echo.NewHTTPError(http.StatusAccepted, "No Incident Created. 'update only' option set.")
	} else {
		incident["correlation_id"] = correlation_id
		incidentID, err := CreateIncident(rc, cmdb_ci, incident)
		if err != nil {
			return err
		}
		response["host"] = incident["host"]
		response["action"] = "Created"
		response["number"] = incidentID
		response["event_type"] = "Incident"
	}

	return c.JSON(201, response)
}

// an empty value means delete any value passed - e.g. short_description in an update
func configDefaults(incident IncidentFields, defaults map[string]string) {
	for k, v := range defaults {
		if _, ok := incident[k]; !ok {
			// trim spaces and surrounding quotes before un-quoting embedded escapes
			str, err := strconv.Unquote(`"` + strings.Trim(v, `"`) + `"`)
			if err == nil {
				v = str
			}
			incident[k] = v
		} else if v == "" {
			delete(incident, k)
		}
	}
}
