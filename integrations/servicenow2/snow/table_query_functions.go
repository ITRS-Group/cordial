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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
)

type results map[string]string
type resultsSlice []results

type snowResult struct {
	Result resultsSlice `json:"result,omitempty"`
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

type snowError struct {
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status"`
}

func (snow TransitiveConnection) QueryTableSingle(table string) (r results, err error) {
	i, err := snow.QueryTable(table)
	if len(i) > 0 {
		r = i[0]
	}
	return
}

func (snow TransitiveConnection) QueryTable(table string) (i resultsSlice, err error) {
	req, err := AssembleRequest(snow, table)
	if err != nil {
		return
	}

	if snow.Trace {
		if reqdump, err := httputil.DumpRequest(req, snow.Method != "GET"); err == nil {
			fmt.Fprintf(os.Stderr, "-- Request --\n\n%s\n\n", string(reqdump))
		}
	}

	resp, err := snow.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("content-type"); !strings.Contains(ct, "application/json") {
		err = echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unexpected content type %s", ct))
		return
	}

	if snow.Trace {
		if respdump, err := httputil.DumpResponse(resp, true); err == nil {
			fmt.Fprintf(os.Stderr, "-- Response --\n\n%s\n\n", string(respdump))
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var r snowResult
		err = json.Unmarshal(body, &r)
		if err != nil {
			err = echo.NewHTTPError(http.StatusInternalServerError, err)
			return
		}
		i = r.Result
	} else {
		if json.Valid(body) {
			var msg snowError
			json.Unmarshal(body, &msg)
			err = echo.NewHTTPError(resp.StatusCode, fmt.Sprintf("%s: %s", msg.Error.Message, msg.Error.Detail))
		} else {
			err = echo.NewHTTPError(resp.StatusCode, string(body))
		}
	}

	return
}

func (snow TransitiveConnection) QueryTableSingleX(table string) (i results, err error) {
	req, err := AssembleRequest(snow, table)
	if err != nil {
		return
	}

	if snow.Trace {
		if reqdump, err := httputil.DumpRequest(req, snow.Method != "GET"); err == nil {
			fmt.Fprintf(os.Stderr, "-- Request --\n\n%s\n\n", string(reqdump))
		}
	}

	resp, err := snow.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("content-type"); !strings.Contains(ct, "application/json") {
		err = echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unexpected content type %s", ct))
		return
	}

	if snow.Trace {
		if respdump, err := httputil.DumpResponse(resp, true); err == nil {
			fmt.Fprintf(os.Stderr, "-- Response --\n\n%s\n\n", string(respdump))
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var r snowResult
		err = json.Unmarshal(body, &r)
		if err != nil {
			err = echo.NewHTTPError(http.StatusInternalServerError, err)
			return
		}
		i = r.Result[0]
	} else {
		if json.Valid(body) {
			var msg snowError
			json.Unmarshal(body, &msg)
			err = echo.NewHTTPError(resp.StatusCode, fmt.Sprintf("%s: %s", msg.Error.Message, msg.Error.Detail))
		} else {
			err = echo.NewHTTPError(resp.StatusCode, string(body))
		}
	}

	return
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
