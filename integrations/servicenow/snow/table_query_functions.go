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

	"github.com/labstack/echo/v4"
)

type ResultDetail map[string]string
type ResultsArray []ResultDetail

type SNOWError struct {
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
	Status string `json:"status"`
}

func (t RequestTransitive) QueryTableDetail(table string) (r ResultDetail, err error) {
	var i ResultsArray
	i, err = t.QueryTable(table)
	if len(i) > 0 {
		r = i[0]
	}
	return
}

func (t RequestTransitive) QueryTable(table string) (i ResultsArray, err error) {
	req, err := AssembleRequest(t, table)
	if err != nil {
		return
	}

	if t.Trace {
		if reqdump, err := httputil.DumpRequest(req, t.Method != "GET"); err == nil {
			fmt.Fprintf(os.Stderr, "-- Request --\n\n%s\n\n", string(reqdump))
		}
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if t.Trace {
		if respdump, err := httputil.DumpResponse(resp, true); err == nil {
			fmt.Fprintf(os.Stderr, "-- Response --\n\n%s\n\n", string(respdump))
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var r map[string]ResultsArray
		err = json.Unmarshal(body, &r)
		if err != nil {
			return
		}
		i = r["result"]
	} else {
		if json.Valid(body) {
			var msg SNOWError
			json.Unmarshal(body, &msg)
			err = echo.NewHTTPError(resp.StatusCode, fmt.Sprintf("%s: %s", msg.Error.Message, msg.Error.Detail))
		} else {
			err = echo.NewHTTPError(resp.StatusCode, string(body))
		}
	}

	return
}

func (t RequestTransitive) QueryTableSingle(table string) (i ResultDetail, err error) {
	req, err := AssembleRequest(t, table)
	if err != nil {
		return
	}

	if t.Trace {
		if reqdump, err := httputil.DumpRequest(req, t.Method != "GET"); err == nil {
			fmt.Fprintf(os.Stderr, "-- Request --\n\n%s\n\n", string(reqdump))
		}
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if t.Trace {
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
		var r map[string]ResultDetail
		err = json.Unmarshal(body, &r)
		if err != nil {
			err = echo.NewHTTPError(http.StatusInternalServerError, err)
			return
		}
		i = r["result"]
	} else {
		if json.Valid(body) {
			var msg SNOWError
			json.Unmarshal(body, &msg)
			err = echo.NewHTTPError(resp.StatusCode, fmt.Sprintf("%s: %s", msg.Error.Message, msg.Error.Detail))
		} else {
			err = echo.NewHTTPError(resp.StatusCode, string(body))
		}
	}

	return
}
