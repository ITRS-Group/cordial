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

	"github.com/labstack/echo/v4"

	"github.com/itrs-group/cordial/pkg/config"
)

func LookupIncident(ctx *Context, cmdb_ci string, correlation_id string) (incident_id string, state int, err error) {
	cf := ctx.Conf
	s := InitializeConnection(cf)

	lookup_map := map[string]string{
		"cmdb_ci":        cmdb_ci,
		"correlation_id": correlation_id,
	}

	table, err := getTableConfig(cf, ctx.Param("table"))
	if err != nil {
		return
	}
	q := cf.ExpandString(table.Search,
		config.LookupTable(lookup_map),
		config.Default(fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s^ORDERBYDESCnumber", cmdb_ci, correlation_id)),
	)
	results, err := s.GET(Fields("sys_id,state"), Query(q)).QueryTableSingle(ctx.Param("table"))
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("lookup incident: %s", err))
		return
	}
	if len(results) == 0 {
		return
	}
	incident_id = results["sys_id"]
	state, _ = strconv.Atoi(results["state"])
	return
}

func LookupCmdbCI(vc *config.Config, table, query, cmdb_ci_default string) (cmdb_ci string, err error) {
	var ok bool
	var r results

	s := InitializeConnection(vc)

	if r, err = s.GET(Fields("sys_id"), Query(query)).QueryTableSingle(table); err != nil {
		err = echo.NewHTTPError(http.StatusNotFound, err)
		return
	}

	if cmdb_ci, ok = r["sys_id"]; !ok {
		if cmdb_ci_default == "" {
			err = echo.NewHTTPError(http.StatusNotFound, nil)
			return
		}
		cmdb_ci = cmdb_ci_default
	}

	return
}
