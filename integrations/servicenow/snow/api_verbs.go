/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
		Parms:      parameters,
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
		Parms:      parameters,
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
		Parms:      parameters,
		SysID:      sysID,
	}
	return

}
