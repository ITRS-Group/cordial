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

package sdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// handle request endpoints

// getRequests retrieves a list of requests from the ServiceDesk Plus
// API based on the search criteria from the configuration, listInfo.
//
// The listInfo parameter can be provided in one of the following
// formats:
//
//   - A JSON string containing the search criteria (e.g. `{"list_info":{"start_index":"1"}}`)
//     This is passed through config.Expand to allow for dynamic values from the configuration file.
//
//   - A url.Values object with the search criteria as form values (e.g. url.Values{"input_data": []string{`{"list_info":{"start_index":"1"}}`}})
//
//   - Any Go struct or map that can be marshalled to JSON, which will be automatically converted to the appropriate format
//     The resulting JSON is passed through config.Expand to allow for dynamic values from the configuration file.
//
// If values is not nil then the values it protects are substituted
// into the JSON, escaped as JSON strings, and are not expanded.
//
// The function will handle the conversion of the search criteria to the
// appropriate format for the API request, and will return a structured
// response containing the list of requests.
func (c *client) getRequests(ctx context.Context, cf *config.Config, listInfo any, values *ims.Literals) (response *RequestGetListResponse, err error) {
	var input string

	switch s := listInfo.(type) {
	case string:
		input = expandJSON(cf, s, values)
	case url.Values:
		return c.listRequests(ctx, s.Encode())
	default:
		b, err2 := json.Marshal(listInfo)
		if err2 != nil {
			return nil, err2
		}
		input = expandJSON(cf, string(b), values)
	}

	return c.listRequests(ctx, url.Values{"input_data": {input}}.Encode())
}

// getRawRequests is like [client.getRequests] but sends the JSON in
// input, which is not expanded as it is from the caller and not the
// configuration
func (c *client) getRawRequests(ctx context.Context, input string) (response *RequestGetListResponse, err error) {
	return c.listRequests(ctx, url.Values{"input_data": {input}}.Encode())
}

// listRequests sends the encoded query to get a list of requests
func (c *client) listRequests(ctx context.Context, query string) (response *RequestGetListResponse, err error) {
	endpoint, err := url.JoinPath("app", config.Get[string](c.sdpCf, "portal"), "/api/v3/requests")
	if err != nil {
		return
	}

	if _, err = c.Get(ctx, endpoint, query, &response); err != nil {
		return nil, ims.UpstreamError("ServiceDesk Plus GET requests failed", err)
	}
	return
}

// expandJSON expands the JSON in input, which is from the
// configuration, with the values from values, if any, substituted as
// they are
func expandJSON(cf *config.Config, input string, values *ims.Literals) string {
	if values == nil {
		return config.Expand[string](cf, input)
	}
	return values.Restore(config.Expand[string](cf, input, values.Option()))
}

// jsonValues returns the values in the lookup tables to be substituted
// into JSON documents, which escapes them as JSON strings
func jsonValues(tables ...map[string]string) *ims.Literals {
	return ims.NewLiterals(true, ims.JSONString, tables...)
}

// inputData returns the form encoded body for a request to ServiceDesk
// Plus, which is the JSON written from sdpCf with the values
// substituted into it
func inputData(sdpCf *config.Config, module string, values *ims.Literals) (body string, err error) {
	var b bytes.Buffer

	if err = sdpCf.Write(module,
		config.Writer(&b),
		config.Format("json"),
		config.ExpandOnSave(values.Option()),
	); err != nil {
		return
	}

	return url.Values{"input_data": {values.Restore(b.String())}}.Encode(), nil
}

func (c *client) createRequest(ctx context.Context, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var resp *json.RawMessage

	body, err := inputData(sdpCf, "sdp", jsonValues(lookup))
	if err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", config.Get[string](c.sdpCf, "portal"), "/api/v3/requests")
	if err != nil {
		return
	}

	if _, err = c.Post(ctx, endpoint, body, &resp); err != nil {
		return nil, ims.UpstreamError("ServiceDesk Plus create request failed", err)
	}

	return decodeStatus(resp)
}

// decodeStatus converts the body of a response from ServiceDesk Plus
// to a config and checks that it does not report a failure
func decodeStatus(resp *json.RawMessage) (response *config.Config, err error) {
	if resp == nil {
		return nil, &ims.HTTPError{Code: http.StatusBadGateway, Message: "empty response from ServiceDesk Plus"}
	}
	response = config.New(config.WithDefaults(*resp, "json"))
	if err = checkStatus(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *client) editRequest(ctx context.Context, id int64, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var resp *json.RawMessage

	body, err := inputData(sdpCf.Sub("edit_request"), "sdp", jsonValues(lookup))
	if err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", config.Get[string](c.sdpCf, "portal"), fmt.Sprintf("/api/v3/requests/%d", id))
	if err != nil {
		return
	}

	if _, err = c.Put(ctx, endpoint, body, &resp); err != nil {
		return nil, ims.UpstreamError("ServiceDesk Plus edit request failed", err)
	}

	return decodeStatus(resp)
}

func (c *client) addNote(ctx context.Context, id int64, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var resp *json.RawMessage

	// TODO: this is currently the same as updateRequest, but in future
	// we may want to allow for different fields or processing for notes
	// vs requests, so keeping it separate for now

	body, err := inputData(sdpCf.Sub("add_note"), "sdp", jsonValues(lookup))
	if err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", config.Get[string](c.sdpCf, "portal"), fmt.Sprintf("/api/v3/requests/%d/notes", id))
	if err != nil {
		return
	}

	if _, err = c.Post(ctx, endpoint, body, &resp); err != nil {
		return nil, ims.UpstreamError("ServiceDesk Plus add note failed", err)
	}

	return decodeStatus(resp)
}
