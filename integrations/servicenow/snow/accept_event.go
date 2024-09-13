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
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

func AcceptEvent(c echo.Context) (err error) {
	cc := c.(*RouterContext)
	vc := cc.Conf

	var incident Incident
	var ok bool
	var cmdb_ci_id, sys_class_name string

	defer c.Request().Body.Close()

	err = json.NewDecoder(c.Request().Body).Decode(&incident)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	s := InitializeConnection(vc)

	if cmdb_ci_id, ok = incident["sys_id"]; !ok {
		var search string
		if search, ok = incident["search"]; !ok {
			if cmdb_ci_id, ok = incident["default_cmdb_ci"]; !ok {
				return echo.NewHTTPError(http.StatusNotFound, "must supply either a default_cmdb_ci or a sys_id search parameter")
			}
		} else {
			if vc.GetString("servicenow.searchtype") == "simple" {
				if cmdb_ci_id, sys_class_name, err = LookupSysIDSimple(vc, "cmdb_ci", search, incident["default_cmdb_ci"]); err != nil {
					return
				}
			} else {
				if cmdb_ci_id, sys_class_name, err = LookupSysID(vc, "cmdb_ci", search, incident["default_cmdb_ci"]); err != nil {
					return
				}
			}
		}
	}

	_ = sys_class_name // unused for now

	if cmdb_ci_id == "" {
		incident["action"] = "Failed"
		return echo.NewHTTPError(http.StatusNotFound, "cmdb_ci sys_id is empty or search resulted in no matches")
	}

	// save for re-user after defaults
	correlation_id := incident["correlation_id"]

	response := make(map[string]string)

	incident_id, short_description, state, err := LookupIncident(vc, cmdb_ci_id, correlation_id)
	if err != nil {
		return
	}

	// always set a short description for an existing incident, even if it's just the one
	// we got back from the lookup. if user wants to remove it, they use the config
	if state > 0 {
		if _, ok := incident["short_description"]; !ok {
			incident["short_description"] = short_description
		}
	}

	// look up state mappings for defaults for 'state' - this is a list of strings
	for sk, sv := range vc.GetStringMapStringSlice("servicenow.incidentstates") {
		if fmt.Sprint(state) == sk {
			for _, si := range sv {
				// trim spaces as YAML comma lists leave spaces in
				si = strings.ToLower(strings.TrimSpace(si))
				configDefaults(incident, vc.GetStringMapString("servicenow.incidentstatedefaults."+si))
			}
		}
	}

	if vc.GetBool(config.Join("servicenow", "incident-user", "lookup")) {
		user := vc.GetString(config.Join("servicenow", "username"))
		userfield := vc.GetString(config.Join("servicenow", "incident-user", "field"), config.Default("caller_id"))
		if _, ok := incident[userfield]; ok {
			user = incident[userfield]
		}

		// basic validation of user
		if !userRE.MatchString(user) {
			return echo.NewHTTPError(http.StatusBadRequest, "username supplied is invalid")
		}

		// only lookup user after all defaults applied
		u, err := s.GET("1", "sys_id", "", "user_name="+user, "").QueryTableDetail("sys_user")
		if err != nil || len(u) == 0 {
			log.Error().Err(err).Msgf("user not found")
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		incident[userfield] = u["sys_id"]
	}

	if incident_id != "" {
		incidentID, err := UpdateIncident(vc, incident_id, incident)
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
		incidentID, err := CreateIncident(vc, cmdb_ci_id, incident)
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
func configDefaults(incident Incident, defaults map[string]string) {
	for k, v := range defaults {
		if _, ok := incident[k]; !ok {
			// trim spaces and surrounding quotes before unquoting embedded escapes
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
