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
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// This is to get a list of all OPEN incidents opened by the service
// user. `format` is the output format, either json (the default) or
// `csv`.
func GetAllIncidents(c echo.Context) (err error) {
	var user, format string

	cc := c.(*RouterContext)
	vc := cc.Conf

	defer c.Request().Body.Close()

	qb := echo.QueryParamsBinder(c)

	err = qb.String("user", &user).BindError()
	if err != nil || user == "" {
		user = vc.GetString("servicenow.username")
	}

	err = qb.String("format", &format).BindError()
	if err != nil || format == "" {
		format = "json"
	}

	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("username %q supplied is invalid", user))
	}

	s := InitializeConnection(vc)

	u, err := s.GET(Limit("1"), Fields("sys_id"), Query("user_name="+user)).QueryTableSingle("sys_user")
	if err != nil || len(u) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("user %q not found in sys_user (and needed for lookup)", user))
	}

	q := fmt.Sprintf(`active=true^opened_by=%s`, u["sys_id"])

	tc, err := getTableConfig(vc, cc.Param("table"))
	if err != nil {
		return
	}
	if !tc.Query.Enabled {
		return echo.ErrNotFound
	}

	l, _ := s.GET(Fields(strings.Join(tc.Query.ResponseFields, ",")), Query(q)).QueryTable(tc.Name)

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
