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
	"context"
	"encoding/json"
	"net/url"
)

// handle request endpoints

// GetRequests retrieves a list of requests from the ServiceDesk Plus
// API based on the provided search criteria.
//
// The searchCriteria parameter can be provided in one of the following
// formats:
//
//   - A JSON string containing the search criteria (e.g. `{"list_info":{"start_index":"1"}}`)
//
//   - A url.Values object with the search criteria as form values (e.g. url.Values{"input_data": []string{`{"list_info":{"start_index":"1"}}`}})
//
//   - Any Go struct or map that can be marshalled to JSON, which will be automatically converted to the appropriate format
//
// The function will handle the conversion of the search criteria to the
// appropriate format for the API request, and will return a structured
// response containing the list of requests.
func (c *Client) GetRequests(ctx context.Context, listInfo any) (response *RequestGetListResponse, err error) {
	v := url.Values{}

	switch s := listInfo.(type) {
	case string:
		v.Add("input_data", s)
	case url.Values:
		v = s
	default:
		b, err2 := json.Marshal(listInfo)
		if err2 != nil {
			return nil, err2
		}
		v.Add("input_data", string(b))
	}

	endpoint, err := url.JoinPath("app", c.cf.GetString("portal"), "/api/v3/requests")
	if err != nil {
		return
	}

	_, err = c.Get(ctx, endpoint, v.Encode(), &response)
	return
}

func CreateRequest(ctx context.Context, request *RequestAttributes) (response *RequestAttributes, err error) {

	return
}
