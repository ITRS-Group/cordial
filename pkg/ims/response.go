/*
Copyright © 2026 ITRS Group

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

package ims

import (
	"encoding/json"
	"net/http"
)

type results map[string]string
type Results []results

// snowResult is the response from ServiceNow. It contains the results
// of the request, which is a slice of results. It also contains an
// error message if the request failed. The status field is used to
// indicate the status of the request. If the request was successful,
// the status will be "success". If the request failed, the status will
// be "error". The error field contains the error message and detail if
// the request failed. The results field contains the results of the
// request.
type SnowResult struct {
	Results Results `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status,omitempty"`
}

type SnowResultsResponse struct {
	Fields  []string `json:"fields,omitempty"`
	Results Results  `json:"results,omitempty"`
}

// Response is the standard response from the IMS gateway, which may
// include the status code and message from the gateway itself, as well as
// any data returned by the gateway and the ID of the created or updated
// incident if applicable. The ProxyResponse field can be used to
// include the raw response from the proxy for debugging or logging
// purposes.
type Response struct {
	Status      string     `json:"status,omitempty"`       // as per http.Response.Status from remote IMS, empty if request failed before reaching IMS
	StatusCode  int        `json:"status_code,omitempty"`  // as per http.Response.StatusCode from remote IMS, empty if request failed before reaching IMS
	Error       string     `json:"error,omitempty"`        // error message if applicable
	ErrorDetail string     `json:"error_detail,omitempty"` // error detail if applicable
	ID          string     `json:"id,omitempty"`           // ID of created or updated incident if applicable
	Data        []string   `json:"data,omitempty"`         // any additional data returned by the gateway, e.g. for query results
	DataTable   [][]string `json:"data_table,omitempty"`   // table of data, if applicable. first row is column names, subsequent rows are values
	RawResponse any        `json:"raw_response,omitempty"`
}

func ServiceNowToResponse(snowResponse SnowResultsResponse) Response {
	var response Response

	response.DataTable = append(response.DataTable, snowResponse.Fields)

	for _, v := range snowResponse.Results {
		var values []string

		for _, c := range snowResponse.Fields {
			values = append(values, v[c])
		}
		response.DataTable = append(response.DataTable, values)
	}
	return response
}

// WriteJSON writes the given value as JSON to the http.ResponseWriter
// with the specified status code. If there is an error encoding the
// value to JSON, it returns the error but does not write an error
// response to the client, as this function is intended to be used for
// writing successful responses. The caller should handle writing error
// responses separately if needed.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return err
	}
	return nil
}
