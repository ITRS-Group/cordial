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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/labstack/echo/v4"
)

func LookupIncident(vc *config.Config, cmdb_ci_id string, correlation_id string) (incident_id string, short_description string, state int, err error) {
	s := InitializeConnection(vc)

	q := fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s", cmdb_ci_id, correlation_id)
	results, err := s.GET(Fields("sys_id,short_description,state"), Query(q)).QueryTableDetail(vc.GetString("servicenow.incidenttable"))
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("lookup incident: %s", err))
		return
	}
	if len(results) == 0 {
		return
	}
	incident_id = results["sys_id"]
	short_description = results["short_description"]
	state, _ = strconv.Atoi(results["state"])
	return
}

func LookupSysIDSimple(vc *config.Config, table, search, cmdb_ci_default string) (sys_id, sys_class_name string, err error) {
	var field, value string
	var ok bool

	s := InitializeConnection(vc)

	s1 := strings.SplitN(search, ":", 2)
	if len(s1) > 1 {
		table = s1[0]
		s1 = s1[1:]
	}
	s2 := strings.SplitN(s1[0], "=", 2)
	if len(s2) < 2 {
		err = echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("invalid simple search %q - must be KEY=VALUE format", s1[0]))
		return
	}
	field = s2[0]
	value = s2[1]

	var r resultDetail

	if r, err = s.GET(Fields("name,sys_id,sys_class_name"), Query(field+"="+value)).QueryTableDetail(table); err != nil {
		err = echo.NewHTTPError(http.StatusNotFound, err)
		return
	}

	if sys_id, ok = r["sys_id"]; !ok {
		if cmdb_ci_default == "" {
			err = echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("search %q returned no results", search))
			return
		}
		sys_id = cmdb_ci_default
	}
	sys_class_name = r["sys_class_name"]

	return
}

func LookupSysID(vc *config.Config, table, search, cmdb_ci_default string) (sys_id, sys_class_name string, err error) {
	var ok bool

	s := InitializeConnection(vc)

	s1 := strings.SplitN(search, ":", 2)
	if len(s1) == 2 {
		table = s1[0]
		search = s1[1]
	}

	var r resultDetail

	if r, err = s.GET(Fields("name,sys_id,sys_class_name"), Query(search)).QueryTableDetail(table); err != nil {
		err = echo.NewHTTPError(http.StatusNotFound, err)
		return
	}

	if sys_id, ok = r["sys_id"]; !ok {
		if cmdb_ci_default == "" {
			err = echo.NewHTTPError(http.StatusNotFound, nil)
			return
		}
		sys_id = cmdb_ci_default
	}
	sys_class_name = r["sys_class_name"]

	return
}
