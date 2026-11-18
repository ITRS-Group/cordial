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
	"encoding/json"
	"regexp"
	"strings"

	"github.com/itrs-group/cordial/pkg/logger"
)

// RequestGetListResponse represents the response for fetching a list of requests (tickets).
type RequestGetListResponse struct {
	Entity         string `json:"entity,omitempty"`
	ResponseStatus []struct {
		StatusCode int    `json:"status_code"`
		Status     string `json:"status"`
	} `json:"response_status"`
	ListInfo struct {
		HasMoreRows bool `json:"has_more_rows"`
		StartIndex  int  `json:"start_index"`
		RowCount    int  `json:"row_count"`
	} `json:"list_info"`
	Requests []*json.RawMessage `json:"requests"` // each response is just plain JSON, so we can unmarshal it into a config object for easier handling of dynamic fields
}

var log = logger.Logger

var sdpField1 = regexp.MustCompile(`^[\w\.-]+$`)

// validateFields checks all the keys in the snow.Record incident.
// SDP fields can consist of letters, numbers, underscored and
// hyphens and cannot begin or end with a hyphen. They cannot be an
// empty string either.
//
// it also returns false if any two field names are the same when
// compared ignoring case. The keys are not changed.
//
// if there are no keys the function returns false.
//
// if there are no invalid fields, the function returns true.
func validateFields(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	// check keys are valid (we cannot use a single regexp to check for
	// non leading hyphen on a single char string)
	for _, k := range keys {
		if k == "" || strings.HasPrefix(k, "-") || strings.HasSuffix(k, "-") {
			return false
		}
		if !sdpField1.MatchString(k) {
			return false
		}
	}

	// check keys are unique when lowercased
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		lower := strings.ToLower(k)
		if _, ok := seen[lower]; ok {
			return false
		}
		seen[lower] = struct{}{}
	}
	return true
}
