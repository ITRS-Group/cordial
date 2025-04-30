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
	"net/http"

	"github.com/labstack/echo/v4"
)

func UpdateIncident(cc *RouterContext, incident_id string, incident IncidentFields) (incident_number string, err error) {
	var postbytes []byte
	var result results

	cf := cc.Conf
	table, err := getTableConfig(cf, cc.Param("table"))
	if err != nil {
		return
	}

	postbytes, err = json.Marshal(incident)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	s := InitializeConnection(cf)
	result, err = s.PUT(postbytes, Fields("number"), SysID(incident_id)).QueryTableSingle(table.Name)
	if err != nil {
		return
	} else {
		incident_number = result["number"]
	}

	return
}
