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

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var daemon bool

func init() {
	rootCmd.AddCommand(routerCmd)

	routerCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the router process")
	routerCmd.Flags().SortFlags = false

}

// routerCmd represents the router command
var routerCmd = &cobra.Command{
	Use:   "router",
	Short: "Run a ServiceNow integration router",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		router()
	},
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

func router() {
	var err error

	vc, err = config.LoadConfig(execname, config.SetAppName("itrs"), config.SetConfigFile(conffile))
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

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
			cc := &snow.RouterContext{Context: c, Conf: vc}
			return next(cc)
		}
	})

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
	v1route.GET("/incident", snow.GetAllIncidents)

	// Put Endpoints
	v1route.POST("/incident", snow.AcceptEvent)

	i := fmt.Sprintf("%s:%d", vc.GetString("api.host"), vc.GetInt("api.port"))

	InitializeConnection()

	// firing up the server
	if !vc.GetBool("api.tls.enabled") {
		e.Logger.Fatal(e.Start(i))
	} else if vc.GetBool("api.tls.enabled") {
		e.Logger.Fatal(e.StartTLS(i, vc.GetString("api.tls.certificate"), vc.GetString("api.tls.key")))
	}
}
