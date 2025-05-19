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
type resultsSlice []results

// Record is a map of key/value pairs
type Record map[string]string

type snowError struct {
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status"`
}

type snowResult struct {
	Result resultsSlice `json:"result,omitempty"`
	Error  struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

func LookupRecord(ctx *Context, options ...config.ExpandOptions) (sys_id string, state int, err error) {
	cf := ctx.Conf

	table, err := getTableConfig(cf, ctx.Param("table"))
	if err != nil {
		return
	}

	results, err := GetRecord(ctx, makeURLPath(ctx.Param("table"), Fields("sys_id,state"), Query(cf.ExpandString(table.Search, options...))))
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("lookup record: %s", err))
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
	r, err := GetRecord(ctx, makeURLPath(table, Fields("sys_id"), Query(query)))
	if err != nil {
		err = echo.NewHTTPError(http.StatusNotFound, err)
		return
	}

	var ok bool
	if cmdb_ci, ok = r["sys_id"]; !ok {
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
	result, err := PostRecord(ctx, makeURLPath(ctx.Param("table"), Fields("number")), record)
	number = result["number"]
	return
}

func (record Record) UpdateRecord(ctx *Context, sys_id string) (number string, err error) {
	result, err := PutRecord(ctx, makeURLPath(ctx.Param("table"), Fields("number"), SysID(sys_id)), record)
	number = result["number"]
	return
}

func (x *resultsSlice) UnmarshalJSON(b []byte) error {
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

func GetRecords(ctx *Context, endpoint *url.URL) (result resultsSlice, err error) {
	var r snowResult
	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	_, err = rc.GetURL(ctx.Request().Context(), endpoint, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	result = r.Result
	return
}

func GetRecord(ctx *Context, endpoint *url.URL) (result results, err error) {
	var r snowResult
	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	_, err = rc.GetURL(ctx.Request().Context(), endpoint, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(r.Result) > 0 {
		result = r.Result[0]
	}
	return
}

func PostRecord(ctx *Context, endpoint *url.URL, record Record) (result results, err error) {
	var r snowResult

	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	_, err = rc.PostURL(ctx.Request().Context(), endpoint, record, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(r.Result) > 0 {
		result = r.Result[0]
	}
	return
}

func PutRecord(ctx *Context, endpoint *url.URL, record Record) (result results, err error) {
	var r snowResult

	rc := ServiceNow(ctx.Conf.Sub("servicenow"))
	_, err = rc.PutURL(ctx.Request().Context(), endpoint, record, &r)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	if len(r.Result) > 0 {
		result = r.Result[0]
	}
	return
}

func makeURLPath(table string, options ...Options) *url.URL {
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

type TableQuery struct {
	Enabled        bool     `mapstructure:"enabled"`
	Search         string   `mapstructure:"search"`
	ResponseFields []string `mapstructure:"response-fields"`
}

type TableStates struct {
	Defaults    map[string]string `mapstructure:"defaults,omitempty"`
	Rename      map[string]string `mapstructure:"rename,omitempty"`
	MustInclude []string          `mapstructure:"must-include,omitempty"`
	Remove      []string          `mapstructure:"remove,omitempty"`
}

type TableData struct {
	Name          string              `mapstructure:"name,omitempty"`
	Search        string              `mapstructure:"search,omitempty"`
	Query         TableQuery          `mapstructure:"query,omitempty"`
	Defaults      map[string]string   `mapstructure:"defaults,omitempty"`
	User          map[string]string   `mapstructure:"user,omitempty"`
	CurrentStates map[int]TableStates `mapstructure:"current-state,omitempty"`
}

func getTableConfig(cf *config.Config, tableName string) (tableData TableData, err error) {
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
