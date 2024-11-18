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

	"github.com/labstack/echo/v4"
)

// This is to get a list of all OPEN incidents opened by the service user
func GetAllIncidents(c echo.Context) (err error) {
	cc := c.(*RouterContext)
	vc := cc.Conf

	defer c.Request().Body.Close()

	var user string
	err = echo.QueryParamsBinder(c).String("user", &user).BindError()
	if err != nil || user == "" {
		user = vc.GetString("servicenow.username")
	}
	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("username %q supplied is invalid", user))
	}

	s := InitializeConnection(vc)
	u, err := s.GET(Limit("1"), Fields("sys_id"), Query("user_name="+user)).QueryTableDetail("sys_user")
	if err != nil || len(u) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("user %q not found in sys_user (and needed for lookup)", user))
	}

	q := fmt.Sprintf(`active=true^opened_by=%s`, u["sys_id"])

	l, _ := s.GET(Fields(vc.GetString("servicenow.queryresponsefields")), Query(q)).QueryTable(vc.GetString("servicenow.incidenttable"))

	return c.JSON(200, l)
}
