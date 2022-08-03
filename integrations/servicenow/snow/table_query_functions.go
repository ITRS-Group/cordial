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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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
	if len(i) != 0 {
		r = i[0]
	}
	return
}

func (t RequestTransitive) QueryTable(table string) (i ResultsArray, err error) {
	req := AssembleRequest(t, table)

	resp, err := t.Client.Do(req)
	if err != nil {
		fmt.Printf("Error %s\n", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error %s\n", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var r map[string]ResultsArray
		err = json.Unmarshal(body, &r)
		if err != nil {
			fmt.Printf("Error %s %s\n", err, string(body))
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
	req := AssembleRequest(t, table)

	resp, err := t.Client.Do(req)
	if err != nil {
		err = echo.NewHTTPError(http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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
