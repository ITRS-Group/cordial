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

package proxy

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/internal/snow"
)

var daemon bool
var logFile string

func init() {
	cmd.RootCmd.AddCommand(routerCmd)

	routerCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the proxy process")
	routerCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "-", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	routerCmd.Flags().SortFlags = false
}

// routerCmd represents the proxy command
var routerCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run a ServiceNow integration proxy",
	Long: strings.ReplaceAll(`
Run an ITRS Geneos to ServiceNow proxy.

The proxy acts as a proxy between Geneos Gateways, each running an
incident submission client, and the ServiceNow instance API. The
proxy can run on a different network endpoint, such as a DMZ, and
can also help limit the number of IP endpoints connecting to a
ServiceNow instance that may have a limit on source connections. The
proxy can also act on data fetched from ServiceNow as part of the
incident submission or update flow.

In normal operation the proxy starts and runs in the foreground,
logging actions and results to stdout/stderr. If started with the
|--daemon| flag it will background itself and no logging will be
available.

The proxy reads it's configuration from a YAML file, which can be
shared with the submission client function, and uses this to look-up,
map and submit incidents.

`, "|", "`"),
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		if daemon {
			process.Daemon(nil, process.RemoveArgs, "-D", "--daemon")
		}

		var l slog.Level
		if cmd.Debug {
			l = slog.LevelDebug
		}
		cf := cmd.LoadConfigFile("proxy")
		// update logging for long running proxy
		cordial.LogInit(cmd.Execname,
			cordial.LogLevel(l),
			cordial.SetLogfile(logFile),
			cordial.LumberjackOptions(&lumberjack.Logger{
				Filename:   logFile,
				MaxSize:    cf.GetInt("server.log.max-size"),
				MaxBackups: cf.GetInt("server.log.max-backups"),
				MaxAge:     cf.GetInt("server.log.stale-after"),
				Compress:   cf.GetBool("server.log.compress"),
			}),
			cordial.RotateOnStart(cf.GetBool("server.log.rotate-on-start")),
		)
		proxy(cf)
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

func proxy(cf *config.Config) {
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
		return key == cf.GetString(cf.Join("server", "authentication", "token")), nil
	}))

	v2route := e.Group(cf.GetString(cf.Join("server", "path")))

	// GET Endpoint
	v2route.GET("/:table", getRecords)

	// POST Endpoint
	v2route.POST("/:table", acceptRecord)

	listen := cf.GetString(cf.Join("server", "listen"))

	// init connection or fail early
	snow.ServiceNow(cf.Sub("servicenow"))

	if cf.GetBool(cf.Join("server", "tls", "enabled")) {
		e.Logger.Fatal(e.StartTLS(
			listen,
			config.GetBytes(cf.Join("server", "tls", "certificate")),
			config.GetBytes(cf.Join("server", "tls", "private-key")),
		))
	}

	e.Logger.Fatal(e.Start(listen))
}

func bodyDumpLog(c echo.Context, reqBody, resBody []byte) {
	var reqMethod string
	var resStatus int

	cf := c.(*snow.Context).Conf

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
		message = result["result"]
	}

	bytes_in := req.Header.Get(echo.HeaderContentLength)
	if bytes_in == "" {
		bytes_in = "0"
	}
	starttime := c.Get("starttime").(time.Time)
	latency := time.Since(starttime)

	log.Info().Msgf("%s %s %3d %s/%d %.3fs %s %s %s %q",
		cf.GetString("servicenow.url"), // name of server (APP) with the environment
		req.Proto,                      // protocol
		resStatus,                      // response status
		// stats here
		bytes_in,
		res.Size,
		float64(latency.Milliseconds())/1000.0,
		c.RealIP(), // client IP
		reqMethod,  // request method
		req.URL,    // request URI (path)
		message,
	)
}
