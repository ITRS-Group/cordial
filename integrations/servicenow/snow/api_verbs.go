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

func (c Connection) POST(payload []byte, limit string, fields string, offset string, query string, sysID string) (t RequestTransitive) {
	parameters := url.Values{}

	if limit != "" {
		parameters.Add("sysparm_limit", limit)
	}
	if fields != "" {
		parameters.Add("sysparm_fields", fields)
	}
	if offset != "" {
		parameters.Add("sysparm_offset", offset)
	}
	if query != "" {
		parameters.Add("sysparm_query", query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	t = RequestTransitive{
		Connection: c,
		Method:     "POST",
		Payload:    payload,
		Params:     parameters,
		SysID:      sysID,
	}
	return
}

func (c Connection) GET(limit string, fields string, offset string, query string, sysID string) (t RequestTransitive) {
	parameters := url.Values{}

	if limit != "" {
		parameters.Add("sysparm_limit", limit)
	}
	if fields != "" {
		parameters.Add("sysparm_fields", fields)
	}
	if offset != "" {
		parameters.Add("sysparm_offset", offset)
	}
	if query != "" {
		parameters.Add("sysparm_query", query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	t = RequestTransitive{
		Connection: c,
		Method:     "GET",
		Params:     parameters,
		SysID:      sysID,
	}
	return
}

func (c Connection) PUT(payload []byte, limit string, fields string, offset string, query string, sysID string) (t RequestTransitive) {
	parameters := url.Values{}

	if limit != "" {
		parameters.Add("sysparm_limit", limit)
	}
	if fields != "" {
		parameters.Add("sysparm_fields", fields)
	}
	if offset != "" {
		parameters.Add("sysparm_offset", offset)
	}
	if query != "" {
		parameters.Add("sysparm_query", query)
	}

	parameters.Add("sysparm_exclude_reference_link", "true")

	t = RequestTransitive{
		Connection: c,
		Method:     "PUT",
		Payload:    payload,
		Params:     parameters,
		SysID:      sysID,
	}
	return

}
