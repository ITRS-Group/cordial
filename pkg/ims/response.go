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
	"time"
)

type results map[string]string
type Results []results

// Response is the standard response from the IMS gateway, which may
// include the status code and message from the gateway itself, as well as
// any data returned by the gateway and the ID of the created or updated
// incident if applicable. The ProxyResponse field can be used to
// include the raw response from the proxy for debugging or logging
// purposes.
type Response struct {
	StartTime    time.Time  `json:"start_time,omitzero"`     // time the request was received by the gateway
	EndTime      time.Time  `json:"end_time,omitzero"`       // time the response is sent by the gateway
	Duration     float64    `json:"duration,omitzero"`       // duration of processing the request in seconds
	Status       string     `json:"status,omitempty"`        // as per http.Response.Status from remote IMS, empty if request failed before reaching IMS
	StatusCode   int        `json:"status_code,omitempty"`   // as per http.Response.StatusCode from remote IMS, empty if request failed before reaching IMS
	Error        string     `json:"error,omitempty"`         // error message if applicable
	ResultDetail string     `json:"result_detail,omitempty"` // error or success detail if applicable
	Action       string     `json:"action,omitempty"`        // action taken by the gateway, e.g. "Created", "Updated", "Ignored", etc.
	ID           string     `json:"id,omitempty"`            // ID of created or updated incident if applicable
	Data         []string   `json:"data,omitempty"`          // any additional data returned by the gateway, e.g. for query results
	DataTable    [][]string `json:"data_table,omitempty"`    // table of data, if applicable. first row is column names, subsequent rows are values
	RawResponse  any        `json:"raw_response,omitempty"`
}

// WriteJSONResponse writes the given value as JSON to the http.ResponseWriter
// with the specified status code. If there is an error encoding the
// value to JSON, it returns the error but does not write an error
// response to the client, as this function is intended to be used for
// writing successful responses. The caller should handle writing error
// responses separately if needed.
func WriteJSONResponse(w http.ResponseWriter, r *http.Request, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := r.Context().Value(ContextKeyResponse)
	if resp == nil {
		return nil // no response to write, but not an error
	}
	response, ok := resp.(*Response)
	if !ok {
		return nil // response in context is not of expected type, but not an error
	}
	response.EndTime = time.Now()
	response.Duration = response.EndTime.Sub(response.StartTime).Seconds()
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return err
	}
	return nil
}
