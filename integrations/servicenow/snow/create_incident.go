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

	"github.com/itrs-group/cordial/pkg/config"
)

type IncidentFields map[string]string

func CreateIncident(vc *config.Config, sys_id string, incident IncidentFields) (incident_number string, err error) {
	var postbytes []byte
	var result resultDetail

	// Initialize ServiceNow Connection
	s := InitializeConnection(vc)

	incident["cmdb_ci"] = sys_id

	// this has to bypass default settings in caller
	if incident["text"] != "" {
		incident["description"] = incident["text"]
		delete(incident, "text")
	}

	postbytes, err = json.Marshal(incident)
	if err != nil {
		return
	}
	result, err = s.POST(postbytes, Fields("number")).QueryTableSingle(vc.GetString("servicenow.incidenttable"))
	if err != nil {
		return
	} else {
		incident_number = result["number"]
	}

	return
}
