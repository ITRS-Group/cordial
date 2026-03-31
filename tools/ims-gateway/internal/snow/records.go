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
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/tools/ims-gateway/internal/common"
)

type result map[string]string
type results []result

// snowResult is the response from ServiceNow. It contains the results
// of the request, which is a slice of results. It also contains an
// error message if the request failed. The status field is used to
// indicate the status of the request. If the request was successful,
// the status will be "success". If the request failed, the status will
// be "error". The error field contains the error message and detail if
// the request failed. The results field contains the results of the
// request.
type snowResult struct {
	Results results `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status,omitempty"`
}

type TableQuery struct {
	Enabled        bool     `mapstructure:"enabled"`
	ResponseFields []string `mapstructure:"fields"`
}

type TableStates struct {
	Defaults    map[string]string `mapstructure:"defaults,omitempty"`
	Remove      []string          `mapstructure:"remove,omitempty"`
	Rename      map[string]string `mapstructure:"rename,omitempty"`
	MustInclude []string          `mapstructure:"must-include,omitempty"`
	Filter      []string          `mapstructure:"filter,omitempty"`
}

type TableResponses struct {
	Created string `mapstructure:"created,omitempty"`
	Updated string `mapstructure:"updated,omitempty"`
	Failed  string `mapstructure:"failed,omitempty"`
}

type TableData struct {
	Name          string                   `mapstructure:"name,omitempty"`
	Search        string                   `mapstructure:"search,omitempty"`
	Query         TableQuery               `mapstructure:"query,omitempty"`
	Defaults      map[string]string        `mapstructure:"defaults,omitempty"`
	CurrentStates map[int]common.Transform `mapstructure:"current-state,omitempty"`
	Response      TableResponses           `mapstructure:"response,omitempty"`
}

func (c *client) lookupRecord(ctx context.Context, cf *config.Config, tableName string, options ...config.ExpandOptions) (sysID string, state int, err error) {
	table, err := tableConfig(cf, tableName)
	if err != nil {
		return
	}

	result, err := c.doRequest(
		ctx,
		http.MethodGet,
		cf.Sub("snow"),
		tableName,
		nil,
		Fields("sys_id,state"),
		Query(cf.ExpandString(table.Search, options...)),
	)
	if err != nil {
		return
	}
	if len(result) == 0 {
		return
	}

	sysID = result["sys_id"]
	state, _ = strconv.Atoi(result["state"]) // parse failure -> 0
	return
}

func (c *client) lookupCmdbCI(ctx context.Context, cf *config.Config, table, query, cmdbCIDefault string) (cmdbCI string, err error) {
	result, err := c.doRequest(
		ctx,
		http.MethodGet,
		cf.Sub("snow"),
		table,
		nil,
		Fields("sys_id"),
		Query(query),
	)
	if err != nil {
		return
	}

	var ok bool
	if cmdbCI, ok = result["sys_id"]; !ok {
		if cmdbCIDefault == "" {
			return "", fmt.Errorf("sys_id not found for query %q", query)
		}
		cmdbCI = cmdbCIDefault
	}

	return
}

// createRecord uses POST to create a ServiceNow record in "table".
func (c *client) createRecord(ctx context.Context, cf *config.Config, record ims.Values, table string) (number string, err error) {
	result, err := c.doRequest(ctx, http.MethodPost, cf.Sub("snow"), table, record, Fields("number"))
	number = result["number"]
	return
}

// updateRecord uses PUT to update a ServiceNow record in "table" with
// the given sysID. The record should contain the fields to update, and
// the number of the updated record is returned.
func (c *client) updateRecord(ctx context.Context, cf *config.Config, record ims.Values, table, sysID string) (number string, err error) {
	result, err := c.doRequest(ctx, http.MethodPut, cf.Sub("snow"), table, record, Fields("number"), SysID(sysID))
	number = result["number"]
	return
}

// UnmarshalJSON satisfies the json Unmarshal interface for *Results
func (x *results) UnmarshalJSON(b []byte) error {
	for _, c := range b {
		switch c {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return json.Unmarshal(b, (*[]result)(x))
		case '{':
			var y result
			if err := json.Unmarshal(b, &y); err != nil {
				return err
			}
			*x = []result{y}
			return nil
		default:
			return fmt.Errorf("cannot unmarshal result: %s", string(b))
		}
	}
	return nil
}

func (c *client) getRecords(ctx context.Context, snowCfg *config.Config, table string, options ...Options) (response *ims.Response, err error) {
	var result snowResult

	response = &ims.Response{}

	opts := evalReqOptions(options...)

	_, err = c.Get(ctx, assembleURL(table, options...), nil, &result)
	if err != nil {
		return response, fmt.Errorf("snow GET records failed: %w", err)
	}

	response.Status = result.Status
	response.Error = result.Error.Message
	response.ResultDetail = result.Error.Detail

	fields := strings.Split(opts.fields, ",")
	response.DataTable = append(response.DataTable, fields)
	for _, v := range result.Results {
		var values []string

		for _, c := range fields {
			values = append(values, v[c])
		}
		response.DataTable = append(response.DataTable, values)
	}
	return response, nil
}

func (c *client) doRequest(ctx context.Context, method string, snowCfg *config.Config, table string, record any, options ...Options) (result result, err error) {
	var r snowResult

	// rc := newClient(snowCfg)
	options = append(options, Limit(1))

	_, err = c.Do(ctx, method, assembleURL(table, options...), record, &r)
	if err != nil {
		return nil, fmt.Errorf("snow GET record failed: %w", err)
	}

	if len(r.Results) > 0 {
		result = r.Results[0]
	}
	return
}

func assembleURL(table string, options ...Options) *url.URL {
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

	p, err := url.JoinPath("table", table, opts.sysID)
	if err != nil {
		log.Error().Err(err).Msg("failed to join snow URL path")
		p = "/table/" + table
	}

	u, err := url.Parse(p)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse snow URL")
		u = &url.URL{Path: p}
	}
	u.RawQuery = v.Encode()

	return u
}

func tableConfig(cf *config.Config, tableName string) (tableData TableData, err error) {
	var tables []TableData

	if err = cf.UnmarshalKey(cf.Join("snow", "tables"), &tables); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}
	for _, t := range tables {
		if t.Name == tableName {
			return t, nil
		}
	}
	err = fmt.Errorf("table details not found")
	return
}
