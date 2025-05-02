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

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/integrations/servicenow2/snow"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

var daemon bool

func init() {
	RootCmd.AddCommand(routerCmd)

	routerCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the router process")
	routerCmd.Flags().SortFlags = false

}

// routerCmd represents the router command
var routerCmd = &cobra.Command{
	Use:   "router",
	Short: "Run a ServiceNow integration router",
	Long: strings.ReplaceAll(`
Run an ITRS Geneos to ServiceNow router.

The router acts as a proxy between Geneos Gateways, each running an
incident submission client, and the ServiceNow instance API. The
router can run on a different network endpoint, such as a DMZ, and
can also help limit the number of IP endpoints connecting to a
ServiceNow instance that may have a limit on source connections. The
router can also act on data fetched from ServiceNow as part of the
incident submission or update flow.

In normal operation the router starts and runs in the foreground,
logging actions and results to stdout/stderr. If started with the
|--daemon| flag it will background itself and no logging will be
available. (Logging to an external file will be added in a future
release)

The router reads it's configuration from a YAML file, which can be
shared with the submission client function, and uses this to look-up,
map and submit incidents.

`, "|", "`"),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		loadConfigFile(cmd)
		router()
	},
}

// timestamp the start of the request
func Timestamp() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("starttime", time.Now())
			return next(c)
		}
	}
}

func router() {
	if daemon {
		process.Daemon(nil, process.RemoveArgs, "-D", "--daemon")
	}

	// Initialization of go-echo server
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	// pass configuration into handlers
	// as per https://echo.labstack.com/guide/context/
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &snow.Context{Context: c, Conf: cf}
			return next(cc)
		}
	})
	e.Use(Timestamp())
	e.Use(middleware.BodyDump(bodyDumpLog))
	e.Use(middleware.KeyAuth(func(key string, c echo.Context) (bool, error) {
		return key == cf.GetString(cf.Join("router", "authentication", "token")), nil
	}))

	v2route := e.Group(cf.GetString(cf.Join("router", "path")))

	// GET Endpoint
	v2route.GET("/:table", snow.GetAllRecords)

	// POST Endpoint
	v2route.POST("/:table", snow.AcceptEvent)

	listen := cf.GetString(cf.Join("router", "listen"))

	// init connection or fail early
	snow.ServiceNow(cf.Sub("servicenow"))

	if cf.GetBool(cf.Join("router", "tls", "enabled")) {
		e.Logger.Fatal(e.StartTLS(
			listen,
			config.GetBytes(cf.Join("router", "tls", "certificate")),
			config.GetBytes(cf.Join("router", "tls", "key")),
		))
	}

	e.Logger.Fatal(e.Start(listen))
}

func bodyDumpLog(c echo.Context, reqBody, resBody []byte) {
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

	if err := json.Unmarshal(resBody, &result); err == nil {
		if result["message"] != "" {
			message = result["message"]
		} else if result["action"] == "Failed" {
			message = fmt.Sprintf("Failed to create event for %s", result["host"])
		} else {
			message = fmt.Sprintf("%s %s %s", result["event_type"], result["number"], result["action"])
		}
	}

	bytes_in := req.Header.Get(echo.HeaderContentLength)
	if bytes_in == "" {
		bytes_in = "0"
	}
	starttime := c.Get("starttime").(time.Time)
	latency := time.Since(starttime)
	latency = latency.Round(time.Millisecond)

	fmt.Printf("%v %s %s %3d %s/%d %v %s %s %s %q\n",
		time.Now().Format(time.RFC3339),     // TIMESTAMP for route access
		cf.GetString("servicenow.instance"), // name of server (APP) with the environment
		req.Proto,                           // protocol
		resStatus,                           // response status
		// stats here
		bytes_in,
		res.Size,
		latency,
		c.RealIP(), // client IP
		reqMethod,  // request method
		req.URL,    // request URI (path)
		message,
	)
}
