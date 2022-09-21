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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
)

var vc *config.Config

type Incident map[string]string

// not a complete test, but just filter characters *allowed*
var userRE = regexp.MustCompile(`^[\w\.@ ]+$`)

func main() {
	var conffile string
	var err error

	pflag.StringVarP(&conffile, "conf", "c", "", "Optional path to configuration file")
	pflag.Parse()

	execname := filepath.Base(os.Args[0])
	vc, err = config.LoadConfig(execname, config.SetAppName("itrs"), config.SetConfigFile(conffile))
	if err != nil {
		log.Fatalln(err)
	}
	// Initialization of go-echo server
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.Use(Timestamp())

	name := vc.GetString("servicenow.instance")
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
		return key == vc.GetString("api.apikey"), nil
	}))

	// list of endpoint routes
	APIRoute := e.Group("/api")
	// grouping routes for version 1.0 API
	v1route := APIRoute.Group("/v1")

	// Get Endpoints
	v1route.GET("/incident", GetAllIncidents)

	// Put Endpoints
	v1route.POST("/incident", AcceptEvent)

	i := fmt.Sprintf("%s:%d", vc.GetString("api.host"), vc.GetInt("api.port"))

	InitializeConnection()

	// firing up the server
	if !vc.GetBool("api.tls.enabled") {
		e.Logger.Fatal(e.Start(i))
	} else if vc.GetBool("api.tls.enabled") {
		e.Logger.Fatal(e.StartTLS(i, vc.GetString("api.tls.certificate"), vc.GetString("api.tls.key")))
	}
}

var snowConnection *snow.Connection

func InitializeConnection() *snow.Connection {
	if snowConnection != nil {
		return snowConnection
	}

	snowConnection = snow.InitializeConnection(vc)
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
