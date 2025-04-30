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
)

type IncidentFields map[string]string

func CreateIncident(ctx *Context, sys_id string, incident IncidentFields) (incident_number string, err error) {
	var postbytes []byte
	var result results

	cf := ctx.Conf
	table, err := getTableConfig(cf, ctx.Param("table"))
	if err != nil {
		return
	}

	// Initialize ServiceNow Connection
	s := InitializeConnection(cf)

	incident["cmdb_ci"] = sys_id

	postbytes, err = json.Marshal(incident)
	if err != nil {
		return
	}
	result, err = s.POST(postbytes, Fields("number")).QueryTableSingle(table.Name)
	if err != nil {
		return
	} else {
		incident_number = result["number"]
	}

	return
}
