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
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/labstack/echo/v4"
)

func LookupIncident(cmdb_ci_id string, correlation_id string) (incident_id string, short_description string, state int, err error) {
	s := InitializeConnection()

	q := fmt.Sprintf("active=true^cmdb_ci=%s^correlation_id=%s", cmdb_ci_id, correlation_id)
	results, err := s.GET("", "sys_id,short_description,state", "", q, "").QueryTableDetail(cf.ServiceNow.IncidentTable)
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

func LookupSysIDSimple(table, search, cmdb_ci_default string) (sys_id, sys_class_name string, err error) {
	var field, value string
	var ok bool

	s := InitializeConnection()

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

	var r snow.ResultDetail

	if r, err = s.GET("", "name,sys_id,sys_class_name", "", field+"="+value, "").QueryTableDetail(table); err != nil {
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

func LookupSysID(table, search, cmdb_ci_default string) (sys_id, sys_class_name string, err error) {
	var ok bool

	s := InitializeConnection()

	s1 := strings.SplitN(search, ":", 2)
	if len(s1) == 2 {
		table = s1[0]
		search = s1[1]
	}

	var r snow.ResultDetail

	if r, err = s.GET("", "name,sys_id,sys_class_name", "", search, "").QueryTableDetail(table); err != nil {
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
