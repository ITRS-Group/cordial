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
	"net/url"
)

// POST TODO: Document its behavior, inputs, and effects.
func (snow Connection) POST(payload []byte, options ...ReqOptions) TransitiveConnection {
	opts := evalReqOptions(options...)

	parameters := url.Values{}

	if opts.limit != "" {
		parameters.Add("sysparm_limit", opts.limit)
	}
	if opts.fields != "" {
		parameters.Add("sysparm_fields", opts.fields)
	}
	if opts.offset != "" {
		parameters.Add("sysparm_offset", opts.offset)
	}
	if opts.query != "" {
		parameters.Add("sysparm_query", opts.query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	return TransitiveConnection{
		Connection: snow,
		Method:     "POST",
		Payload:    payload,
		Params:     parameters,
		SysID:      opts.sysID,
	}
}

// GET TODO: Document its behavior, inputs, and effects.
func (snow Connection) GET(options ...ReqOptions) TransitiveConnection {
	opts := evalReqOptions(options...)

	parameters := url.Values{}

	if opts.limit != "" {
		parameters.Add("sysparm_limit", opts.limit)
	}
	if opts.fields != "" {
		parameters.Add("sysparm_fields", opts.fields)
	}
	if opts.offset != "" {
		parameters.Add("sysparm_offset", opts.offset)
	}
	if opts.query != "" {
		parameters.Add("sysparm_query", opts.query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	return TransitiveConnection{
		Connection: snow,
		Method:     "GET",
		Params:     parameters,
		SysID:      opts.sysID,
	}
}

// limit string, fields string, offset string, query string, sysID string
func (snow Connection) PUT(payload []byte, options ...ReqOptions) TransitiveConnection {
	opts := evalReqOptions(options...)

	parameters := url.Values{}

	if opts.limit != "" {
		parameters.Add("sysparm_limit", opts.limit)
	}
	if opts.fields != "" {
		parameters.Add("sysparm_fields", opts.fields)
	}
	if opts.offset != "" {
		parameters.Add("sysparm_offset", opts.offset)
	}
	if opts.query != "" {
		parameters.Add("sysparm_query", opts.query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	return TransitiveConnection{
		Connection: snow,
		Method:     "PUT",
		Payload:    payload,
		Params:     parameters,
		SysID:      opts.sysID,
	}
}

type reqOptions struct {
	limit  string
	fields string
	offset string
	query  string
	sysID  string
}

// ReqOptions TODO: Describe its purpose and intended use.
type ReqOptions func(*reqOptions)

func evalReqOptions(options ...ReqOptions) (opts *reqOptions) {
	opts = &reqOptions{}
	for _, r := range options {
		r(opts)
	}
	return
}

// Limit TODO: Document its behavior, inputs, and effects.
func Limit(limit string) ReqOptions {
	return func(ro *reqOptions) {
		ro.limit = limit
	}
}

// Fields TODO: Document its behavior, inputs, and effects.
func Fields(fields string) ReqOptions {
	return func(ro *reqOptions) {
		ro.fields = fields
	}
}

// Offset TODO: Document its behavior, inputs, and effects.
func Offset(offset string) ReqOptions {
	return func(ro *reqOptions) {
		ro.offset = offset
	}
}

// Query TODO: Document its behavior, inputs, and effects.
func Query(query string) ReqOptions {
	return func(ro *reqOptions) {
		ro.query = query
	}
}

// SysID TODO: Document its behavior, inputs, and effects.
func SysID(sysID string) ReqOptions {
	return func(ro *reqOptions) {
		ro.sysID = sysID
	}
}
