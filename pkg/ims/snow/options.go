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

package snow

import "strconv"

type reqOptions struct {
	limit   string
	fields  string
	offset  string
	query   string
	display string
	sysID   string
}

// Option sets a parameter of a request to the ServiceNow Table API
type Option func(*reqOptions)

func evalReqOptions(options ...Option) (opts *reqOptions) {
	opts = &reqOptions{}
	for _, r := range options {
		r(opts)
	}
	return
}

// Limit sets the `sysparm_limit` parameter
func Limit(limit int) Option {
	return func(ro *reqOptions) {
		ro.limit = strconv.Itoa(limit)
	}
}

// Fields sets the `sysparm_fields` parameter, a comma separated list of
// field names
func Fields(fields string) Option {
	return func(ro *reqOptions) {
		ro.fields = fields
	}
}

// Offset sets the `sysparm_offset` parameter
func Offset(offset string) Option {
	return func(ro *reqOptions) {
		ro.offset = offset
	}
}

// Query sets the `sysparm_query` parameter, an encoded query
func Query(query string) Option {
	return func(ro *reqOptions) {
		ro.query = query
	}
}

// Display sets the `sysparm_display_value` parameter
func Display(display string) Option {
	return func(ro *reqOptions) {
		ro.display = display
	}
}

// SysID selects a single record, by appending its sys_id to the path
func SysID(sysID string) Option {
	return func(ro *reqOptions) {
		ro.sysID = sysID
	}
}

// FieldsOf returns the value of the [Fields] option in options, which is
// empty if it is not set.
func FieldsOf(options ...Option) string {
	return evalReqOptions(options...).fields
}
