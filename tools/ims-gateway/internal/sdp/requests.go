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
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
)

// handle request endpoints

// getRequests retrieves a list of requests from the ServiceDesk Plus
// API based on the provided search criteria.
//
// The searchCriteria parameter can be provided in one of the following
// formats:
//
//   - A JSON string containing the search criteria (e.g. `{"list_info":{"start_index":"1"}}`)
//     This is passed through config.Expand to allow for dynamic values from the configuration file. opts is passed to config.Expand to allow for additional options such as custom functions.
//
//   - A url.Values object with the search criteria as form values (e.g. url.Values{"input_data": []string{`{"list_info":{"start_index":"1"}}`}})
//
//   - Any Go struct or map that can be marshalled to JSON, which will be automatically converted to the appropriate format
//     The resulting JSON is passed through config.Expand to allow for dynamic values from the configuration file. opts is passed to config.Expand to allow for additional options such as custom functions.
//
// The function will handle the conversion of the search criteria to the
// appropriate format for the API request, and will return a structured
// response containing the list of requests.
func (c *client) getRequests(ctx context.Context, cf *config.Config, listInfo any, opts ...config.ExpandOptions) (response *RequestGetListResponse, err error) {
	v := url.Values{}

	switch s := listInfo.(type) {
	case string:
		v.Add("input_data", string(cf.Expand(s, opts...)))
	case url.Values:
		v = s
	default:
		b, err2 := json.Marshal(listInfo)
		if err2 != nil {
			return nil, err2
		}
		v.Add("input_data", string(cf.Expand(string(b), opts...)))
	}

	endpoint, err := url.JoinPath("app", c.sdpCf.GetString("portal"), "/api/v3/requests")
	if err != nil {
		return
	}

	_, err = c.Get(ctx, endpoint, v.Encode(), &response)
	return
}

func (c *client) createRequest(ctx context.Context, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var b bytes.Buffer
	var resp string

	if err = sdpCf.SaveTo("sdp", &b,
		config.SetFileExtension("json"),
		config.ExpandOnSave(
			config.LookupTable(lookup),
		),
	); err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", c.sdpCf.GetString("portal"), "/api/v3/requests")
	if err != nil {
		return
	}

	_, err = c.Post(ctx, endpoint, "input_data="+b.String(), &resp)

	response = config.New(config.WithDefaults([]byte(resp), "json"))
	return
}

func (c *client) editRequest(ctx context.Context, id int64, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var b bytes.Buffer
	var resp *json.RawMessage

	sdpReqCf := sdpCf.Sub("edit_request")

	if err = sdpReqCf.SaveTo("sdp", &b,
		config.SetFileExtension("json"),
		config.ExpandOnSave(
			config.LookupTable(lookup),
		),
	); err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", c.sdpCf.GetString("portal"), fmt.Sprintf("/api/v3/requests/%d", id))
	if err != nil {
		return
	}

	_, err = c.Put(ctx, endpoint, "input_data="+b.String(), &resp)

	if resp == nil {
		err = fmt.Errorf("empty response from API")
		return
	}

	response = config.New(config.WithDefaults(*resp, "json"))
	return
}

func (c *client) addNote(ctx context.Context, id int64, sdpCf *config.Config, lookup map[string]string) (response *config.Config, err error) {
	var b bytes.Buffer
	var resp *json.RawMessage

	// TODO: this is currently the same as updateRequest, but in future
	// we may want to allow for different fields or processing for notes
	// vs requests, so keeping it separate for now

	sdpReqnoteCf := sdpCf.Sub("add_note")
	if err = sdpReqnoteCf.SaveTo("sdp", &b,
		config.SetFileExtension("json"),
		config.ExpandOnSave(
			config.LookupTable(lookup),
		),
	); err != nil {
		return
	}

	endpoint, err := url.JoinPath("app", c.sdpCf.GetString("portal"), fmt.Sprintf("/api/v3/requests/%d/notes", id))
	if err != nil {
		return
	}

	_, err = c.Post(ctx, endpoint, "input_data="+b.String(), &resp)

	if resp == nil {
		err = fmt.Errorf("empty response from API")
		return
	}

	response = config.New(config.WithDefaults(*resp, "json"))
	return
}
