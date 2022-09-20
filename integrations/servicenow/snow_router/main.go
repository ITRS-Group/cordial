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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/itrs-group/cordial/integrations/servicenow/settings"
	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

var cf settings.Settings

type Incident map[string]string

// not a complete test, but just filter characters *allowed*
var userRE = regexp.MustCompile(`^[\w\.@ ]+$`)

func main() {
	var conffile string
	flag.StringVar(&conffile, "conf", "", "Optional path to configuration file")
	flag.Parse()

	execname := filepath.Base(os.Args[0])
	cf = settings.GetConfig(conffile, execname)

	// Initialization of go-echo server
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.Use(Timestamp())

	name := cf.ServiceNow.Instance
	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		var reqMethod string
		var resStatus int
		// request and response object
		req := c.Request()
		res := c.Response()
		// rendering variables for response status and request method
		resStatus = res.Status
		reqMethod = req.Method

		// print formatting the custom logger tailored for DEVELOPMENT environment
		var result map[string]string
		var message string
		json.Unmarshal(resBody, &result)
		if result["message"] != "" {
			message = result["message"]
		} else if result["action"] == "Failed" {
			message = fmt.Sprintf("Failed to create event for %s", result["host"])
		} else {
			message = fmt.Sprintf("%s %s %s", result["event_type"], result["number"], result["action"])
		}

		bytes_in := req.Header.Get(echo.HeaderContentLength)
		if bytes_in == "" {
			bytes_in = "0"
		}
		starttime := c.Get("starttime").(time.Time)
		latency := time.Since(starttime)
		latency = latency.Round(time.Millisecond)

		fmt.Printf("%v %s %s %3d %s/%d %v %s %s %s %q\n",
			time.Now().Format(time.RFC3339), // TIMESTAMP for route access
			name,                            // name of server (APP) with the environment
			req.Proto,                       // protocol
			resStatus,                       // response status
			// stats here
			bytes_in,
			res.Size,
			latency,
			c.RealIP(), // client IP
			reqMethod,  // request method
			req.URL,    // request URI (path)
			message,
		)
	}))

	e.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		return key == cf.API.APIKey, nil
	}))

	// list of endpoint routes
	APIRoute := e.Group("/api")
	// grouping routes for version 1.0 API
	v1route := APIRoute.Group("/v1")

	// Get Endpoints
	v1route.GET("/incident", GetAllIncidents)

	// Put Endpoints
	v1route.POST("/incident", AcceptEvent)

	i := fmt.Sprintf("%s:%d", cf.API.Host, cf.API.Port)

	_ = InitializeConnection()

	// firing up the server
	if !cf.API.TLS.Enabled {
		e.Logger.Fatal(e.Start(i))
	} else if cf.API.TLS.Enabled {
		e.Logger.Fatal(e.StartTLS(i, cf.API.TLS.Certificate, cf.API.TLS.Key))
	}
}

var snowConnection *snow.Connection

func InitializeConnection() *snow.Connection {
	// get password from a file
	var pw []byte
	var err error
	if snowConnection != nil {
		return snowConnection
	}

	pw = []byte(config.GetConfig().ExpandString(cf.ServiceNow.Password, nil))
	if len(pw) == 0 {
		if strings.HasPrefix(cf.ServiceNow.PasswordFile, "~/") {
			home, _ := os.UserHomeDir()
			cf.ServiceNow.PasswordFile = strings.Replace(cf.ServiceNow.PasswordFile, "~", home, 1)
		}
		if pw, err = os.ReadFile(cf.ServiceNow.PasswordFile); err != nil {
			log.Fatalf("cannot read password from file %q", cf.ServiceNow.PasswordFile)
		}
	}
	snowConnection = snow.InitializeConnection(cf, strings.TrimSpace(string(pw)))
	return snowConnection
}

// timestamp the start of the request
func Timestamp() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// do nothing
			c.Set("starttime", time.Now())
			return next(c)
		}
	}
}
