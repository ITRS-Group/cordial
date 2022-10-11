/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package snow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
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
		fmt.Printf("Failed reading the request body %s", err)

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

	user := vc.GetString("servicenow.username")
	if _, ok := incident["caller_id"]; ok {
		user = incident["caller_id"]
	}

	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, "username supplied is invalid")
	}

	// only lookup user after all defaults applied
	u, err := s.GET("1", "sys_id", "", "user_name="+user, "").QueryTableDetail("sys_user")
	if err != nil || len(u) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}
	incident["caller_id"] = u["sys_id"]

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
