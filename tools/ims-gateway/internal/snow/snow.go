/*
Copyright © 2025 ITRS Group

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

// package snow provides the ServiceNow integration for the IMS Gateway.
// It provides a client for connecting to ServiceNow and endpoints for
// receiving records from the IMS Gateway and sending them to
// ServiceNow. It also provides a configuration struct for configuring
// the integration and a function for validating the configuration.
package snow

import (
	"regexp"
	"slices"
	"strings"
)

type ResultsResponse struct {
	Fields  []string `json:"fields,omitempty"`
	Results results  `json:"results,omitempty"`
}

var snowFieldRE = regexp.MustCompile(`^[\w\.-]+$`)

// validateFields checks all the keys in the incident. ServiceNow fields
// can consist of letters, numbers, underscored and hyphens and cannot
// begin or end with a hyphen. They cannot be an empty string either.
//
// The function also lowercases all fields names and if there is a clash
// it returns false.
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
		if !snowFieldRE.MatchString(k) {
			return false
		}
	}

	// check keys are unique when lowercased
	slices.SortFunc(keys, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	l1 := len(keys)
	l2 := len(slices.Compact(keys)) // slices.Compact modifies the slice

	return l1 == l2
}
