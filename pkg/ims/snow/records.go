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

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/ims"
)

// Result is a single record, as a map of field names to values
type Result map[string]string

// Results is a list of records. When decoded from JSON a single object
// is accepted as a list of one.
type Results []Result

// UnmarshalJSON satisfies the json Unmarshal interface for *Results
func (x *Results) UnmarshalJSON(b []byte) error {
	for _, c := range b {
		switch c {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return json.Unmarshal(b, (*[]Result)(x))
		case '{':
			var y Result
			if err := json.Unmarshal(b, &y); err != nil {
				return err
			}
			*x = []Result{y}
			return nil
		default:
			return fmt.Errorf("cannot unmarshal result: %s", string(b))
		}
	}
	return nil
}

// Table returns the results as rows of values, in the order of fields,
// preceded by a row of the field names.
func (x Results) Table(fields []string) (table [][]string) {
	table = append(table, fields)
	for _, v := range x {
		values := make([]string, 0, len(fields))
		for _, f := range fields {
			values = append(values, v[f])
		}
		table = append(table, values)
	}
	return
}

// response is the body of a response from ServiceNow. It contains the
// results of the request, which is a slice of results, and an error
// message and detail if the request failed. The status field is used to
// indicate the status of the request, and is `failure` if it failed.
type response struct {
	Results Results `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status,omitempty"`
}

// check returns an error if the response body reports a failure, even
// if the HTTP status code was successful.
func (r *response) check() error {
	if r.Error.Message == "" && !strings.EqualFold(r.Status, "failure") {
		return nil
	}
	msg := "ServiceNow reported an error"
	if r.Error.Message != "" {
		msg += ": " + r.Error.Message
	}
	if r.Error.Detail != "" {
		msg += " (" + r.Error.Detail + ")"
	}
	return &ims.HTTPError{Code: http.StatusBadGateway, Message: msg}
}

// Records returns all the records in table that match the options.
func (c *Client) Records(ctx context.Context, table string, options ...Option) (results Results, err error) {
	var r response

	if _, err = c.Get(ctx, AssembleURL(table, options...), nil, &r); err != nil {
		return nil, ims.UpstreamError("ServiceNow GET request failed", err)
	}
	if err = r.check(); err != nil {
		return nil, err
	}
	return r.Results, nil
}

// Record returns the first record in table that matches the options, or
// an empty result if there is none.
func (c *Client) Record(ctx context.Context, table string, options ...Option) (result Result, err error) {
	// ensure limit is set to 1
	results, err := c.Records(ctx, table, append(slices.Clone(options), Limit(1))...)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		result = results[0]
	}
	log.Debug("record found", slog.Any("result", result))
	return
}

// LookupRecord searches table using the encoded query and returns the
// sys_id and state of the first match. If there is no match then sys_id
// is empty and the state is 0, which is also the state if it cannot be
// parsed.
func (c *Client) LookupRecord(ctx context.Context, table, query string) (sysID string, state int, err error) {
	result, err := c.Record(ctx, table, Fields("sys_id,state"), Query(query))
	if err != nil {
		return
	}
	sysID = result["sys_id"]
	state, _ = strconv.Atoi(result["state"])
	return
}

// LookupCmdbCI searches table using the encoded query and returns the
// sys_id of the first match. If there is no match then cmdbCIDefault
// is returned, unless it is empty, in which case the error has a 404
// status.
func (c *Client) LookupCmdbCI(ctx context.Context, table, query, cmdbCIDefault string) (cmdbCI string, err error) {
	result, err := c.Record(ctx, table, Fields("sys_id"), Query(query))
	if err != nil {
		return
	}

	var ok bool
	if cmdbCI, ok = result["sys_id"]; !ok {
		if cmdbCIDefault == "" {
			return "", &ims.HTTPError{Code: http.StatusNotFound, Message: fmt.Sprintf("sys_id not found for query %q", query)}
		}
		cmdbCI = cmdbCIDefault
	}
	return
}

// Create creates a record in table and returns its number, which is
// empty if the table does not have one. It is an error if ServiceNow
// does not return the record that was created.
func (c *Client) Create(ctx context.Context, table string, record map[string]string) (number string, err error) {
	result, err := c.write(ctx, http.MethodPost, "POST (Create)", table, record, Fields("sys_id,number"))
	return result["number"], err
}

// Update updates the record in table with the sys_id and returns its
// number, which is empty if the table does not have one. It is an error
// if ServiceNow does not return the record that was updated.
func (c *Client) Update(ctx context.Context, table, sysID string, record map[string]string) (number string, err error) {
	result, err := c.write(ctx, http.MethodPut, "PUT (Update)", table, record, Fields("sys_id,number"), SysID(sysID))
	return result["number"], err
}

func (c *Client) write(ctx context.Context, method, what, table string, record map[string]string, options ...Option) (result Result, err error) {
	var r response

	if c.trace {
		if js, err := json.MarshalIndent(record, "", "    "); err != nil {
			log.Debug("failed to marshal trace record", slog.String("method", method), slog.Any("error", err))
		} else {
			log.Debug(what+" record", slog.String("table", table), slog.String("record", string(js)))
		}
	}

	if _, err = c.Do(ctx, method, AssembleURL(table, options...), record, &r); err != nil {
		return nil, ims.UpstreamError("ServiceNow "+method+" request failed", err)
	}
	if err = r.check(); err != nil {
		return nil, err
	}

	// a successful create or update must return the record
	if len(r.Results) == 0 || len(r.Results[0]) == 0 {
		return nil, &ims.HTTPError{Code: http.StatusBadGateway, Message: "ServiceNow accepted the request but returned no record"}
	}
	return r.Results[0], nil
}

// AssembleURL returns the URL, relative to the base URL of the client,
// for a request on table with the options.
func AssembleURL(table string, options ...Option) *url.URL {
	opts := evalReqOptions(options...)

	v := url.Values{}
	if opts.limit != "" {
		v.Add("sysparm_limit", opts.limit)
	}
	if opts.fields != "" {
		v.Add("sysparm_fields", opts.fields)
	}
	if opts.offset != "" {
		v.Add("sysparm_offset", opts.offset)
	}
	if opts.query != "" {
		v.Add("sysparm_query", opts.query)
	}
	if opts.display != "" {
		v.Add("sysparm_display_value", opts.display)
	}
	v.Add("sysparm_exclude_reference_link", "true")

	return &url.URL{
		Path:     path.Join("table", table, opts.sysID),
		RawQuery: v.Encode(),
	}
}
