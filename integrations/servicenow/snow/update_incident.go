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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/labstack/echo/v4"
)

func UpdateIncident(vc *config.Config, incident_id string, incident Incident) (incident_number string, err error) {
	var postbytes []byte
	var result ResultDetail

	// this has to bypass default settings in caller
	if incident["text"] != "" {
		incident["work_notes"] = incident["text"]
		delete(incident, "text")
	}

	postbytes, err = json.Marshal(incident)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	s := InitializeConnection(vc)
	result, err = s.PUT(postbytes, "", "number", "", "", incident_id).QueryTableSingle(vc.GetString("servicenow.incidenttable"))
	if err != nil {
		return
	} else {
		incident_number = result["number"]
	}

	return
}
