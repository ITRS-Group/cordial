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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

type results map[string]string
type Results []results

// Record is a map of key/value pairs
type Record map[string]string

// snowResult is the response from ServiceNow. It contains the results
// of the request, which is a slice of results. It also contains an
// error message if the request failed. The status field is used to
// indicate the status of the request. If the request was successful,
// the status will be "success". If the request failed, the status will
// be "error". The error field contains the error message and detail if
// the request failed. The results field contains the results of the
// request.
type snowResult struct {
	Results Results `json:"result,omitempty"`
	Error   struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error,omitempty"`
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
	Name          string              `mapstructure:"name,omitempty"`
	Search        string              `mapstructure:"search,omitempty"`
	Query         TableQuery          `mapstructure:"query,omitempty"`
	Defaults      map[string]string   `mapstructure:"defaults,omitempty"`
	CurrentStates map[int]TableStates `mapstructure:"current-state,omitempty"`
	Response      TableResponses      `mapstructure:"response,omitempty"`
}

func LookupRecord(ctx *Context, options ...config.ExpandOptions) (sys_id string, state int, err error) {
	cf := ctx.Conf

	table, err := TableConfig(cf, ctx.Param("table"))
	if err != nil {
		return
	}

	results, err := GetRecord(ctx, ctx.Param("table"), Fields("sys_id,state"), Query(cf.ExpandString(table.Search, options...)))
	if err != nil {
		return
	}
	if len(results) == 0 {
		return
	}
	sys_id = results["sys_id"]
	state, _ = strconv.Atoi(results["state"]) // error results in 0, which is valid
	return
}

func LookupCmdbCI(ctx *Context, table, query, cmdb_ci_default string) (cmdb_ci string, err error) {
	result, err := GetRecord(ctx, table, Fields("sys_id"), Query(query))
	if err != nil {
		return
	}

	var ok bool
	if cmdb_ci, ok = result["sys_id"]; !ok {
		if cmdb_ci_default == "" {
			err = echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("sys_id not found for query %q", query))
			return
		}
		cmdb_ci = cmdb_ci_default
	}

	return
}

// CreateRecord uses the POST method to send create a new ServiceNow record in the table named as a parameter
func (record Record) CreateRecord(ctx *Context) (number string, err error) {
	result, err := PostRecord(ctx, ctx.Param("table"), record, Fields("number"))
	number = result["number"]
	return
}

func (record Record) UpdateRecord(ctx *Context, sys_id string) (number string, err error) {
	result, err := PutRecord(ctx, ctx.Param("table"), record, Fields("number"), SysID(sys_id))
	number = result["number"]
	return
}

// UnmarshalJSON satisfies the json Unmarshal interface for *Results
func (x *Results) UnmarshalJSON(b []byte) error {
	for _, c := range b {
		switch c {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return json.Unmarshal(b, (*[]results)(x))
		case '{':
			var y results
			if err := json.Unmarshal(b, &y); err != nil {
				return err
			}
			*x = []results{y}
			return nil
		default:
			return fmt.Errorf("cannot unmarshal result: %s", string(b))
		}
	}
	return nil
}

func GetRecords(ctx *Context, table string, options ...Options) (results Results, err error) {
	var result snowResult
	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	_, err = rc.Get(ctx.Request().Context(), AssembleURL(table, options...), nil, &result)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	results = result.Results
	return
}

func GetRecord(ctx *Context, table string, options ...Options) (results results, err error) {
	var result snowResult
	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	// ensure limit is set to 1
	options = append(options, Limit(1))
	_, err = rc.Get(ctx.Request().Context(), AssembleURL(table, options...), nil, &result)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(result.Results) > 0 {
		results = result.Results[0]
	}
	log.Debug().Msgf("GetRecord results: %+v", results)
	return
}

func PostRecord(ctx *Context, table string, record Record, options ...Options) (result results, err error) {
	var r snowResult

	cf := ctx.Conf.Sub("servicenow")
	rc := ServiceNow(ctx.Conf.Sub("servicenow"))

	if cf.GetBool("trace") {
		js, err := json.MarshalIndent(record, "", "    ")
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal trace record for POST request")
		} else {
			log.Debug().Msgf("POST (Create) %s with record:\n%s", table, js)
		}
	}
	_, err = rc.Post(ctx.Request().Context(), AssembleURL(table, options...), record, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(r.Results) > 0 {
		result = r.Results[0]
	}
	return
}

func PutRecord(ctx *Context, table string, record Record, options ...Options) (result results, err error) {
	var r snowResult

	cf := ctx.Conf.Sub("servicenow")
	rc := ServiceNow(cf)

	if cf.GetBool("trace") {
		js, err := json.MarshalIndent(record, "", "    ")
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshal trace record for PUT request")
		} else {
			log.Debug().Msgf("PUT (Update) %s with record:\n%s", table, js)
		}
	}

	_, err = rc.Put(ctx.Request().Context(), AssembleURL(table, options...), record, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(r.Results) > 0 {
		result = r.Results[0]
	}
	return
}

func AssembleURL(table string, options ...Options) *url.URL {
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
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	u, err := url.Parse(p)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	u.RawQuery = v.Encode()

	return u
}

func TableConfig(cf *config.Config, tableName string) (tableData TableData, err error) {
	var tables []TableData

	if err = cf.UnmarshalKey("servicenow.tables", &tables); err != nil {
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
