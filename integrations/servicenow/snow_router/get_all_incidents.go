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

package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// This is to get a list of all OPEN incidents opened by the service user
func GetAllIncidents(c echo.Context) (err error) {
	defer c.Request().Body.Close()

	var user string
	err = echo.QueryParamsBinder(c).String("user", &user).BindError()
	if err != nil || user == "" {
		user = cf.ServiceNow.Username
	}
	// real basic validation of user
	if !userRE.MatchString(user) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("username %q supplied is invalid", user))
	}

	s := InitializeConnection()
	u, err := s.GET("1", "sys_id", "", "user_name="+user, "").QueryTableDetail("sys_user")
	if err != nil || len(u) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("user %q not found in sys_user (and needed for lookup)", user))
	}

	// q := fmt.Sprintf(`stateNOT IN%s^opened_by=%s`, cf.ServiceNow.IncidentStates.Closed, u["sys_id"])
	q := fmt.Sprintf(`active=true^opened_by=%s`, u["sys_id"])

	l, _ := s.GET("", cf.ServiceNow.QueryResponseFields, "", q, "").QueryTable(cf.ServiceNow.IncidentTable)

	return c.JSON(200, l)
}
