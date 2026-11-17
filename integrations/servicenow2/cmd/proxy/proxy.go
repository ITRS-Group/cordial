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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DeRuina/timberjack"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/process/control"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/pkg/ims/snow"
)

var daemon bool
var logFile, pidFile string

// background is the control of the proxy when it is running in the
// background, with the commands `proxy stop`, `restart`, `reload` and
// `status`
var background = &control.Daemon{
	Name:        "servicenow2 proxy",
	PIDFileName: "servicenow2.proxy.pid",
	PIDFile:     &pidFile,
	ConfigFile:  cmd.ConfigFile,
	Load:        func(file string) (*config.Config, error) { return cmd.LoadConfigFileFrom("proxy", file) },
	Check:       checkConfig,
}

func init() {
	cmd.Cmd.AddCommand(routerCmd)

	routerCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the proxy process")
	routerCmd.Flags().StringVar(&pidFile, "pidfile", "", control.FlagUsage)
	routerCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "-", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	routerCmd.AddCommand(background.Commands()...)

	routerCmd.Flags().SortFlags = false
}

// checkConfig checks that the configuration can be used to run the
// proxy, which is the server settings and a usable connection to
// ServiceNow
func checkConfig(cf *config.Config) error {
	if err := ims.CheckServer(cf); err != nil {
		return err
	}
	_, err := snow.New(cf.Sub("servicenow"))
	return err
}

var log = logger.Logger

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
|--daemon| flag it will background itself, writing its log to a file, and
only return once it is listening or has failed to start. It holds a PID
file while it runs, which is the one given with |--pidfile|, or set as
|server.pidfile| in the configuration, or if neither a file in the
user's cache directory when it is a background process.

A proxy running in the background is controlled by the |stop|,
|restart|, |reload| and |status| sub-commands. |reload| makes the proxy
read its configuration again, if it is valid, without interrupting any
requests that are in progress. This is also done by sending the process
the |SIGUSR1| signal. Changes to the logging settings need a restart.

The proxy reads it's configuration from a YAML file, which can be
shared with the submission client function, and uses this to look-up,
map and submit incidents.

`, "|", "`"),
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		// load the configuration before daemonising so that a missing or
		// invalid file is reported to the user and not lost with the
		// background process
		cf, err := cmd.LoadConfigFile("proxy")
		if err != nil {
			log.Error("cannot start proxy", slog.Any("error", err))
			os.Exit(1)
		}

		pidfile := control.ConfiguredPIDFile(cf, pidFile)

		if daemon {
			if pidfile == "" {
				pidfile = control.DefaultPIDFile(background.PIDFileName)
			}

			var logArgs []string
			logName := logFile
			if logFile == "-" {
				logName = "servicenow2.proxy.log"
				logArgs = append(logArgs, "--logfile", logName)
			}

			pid, err := control.Daemonize(pidfile, logArgs...)
			if err != nil {
				log.Error("failed to start proxy in the background", slog.Any("error", err), slog.String("logfile", logName))
				os.Exit(1)
			}
			fmt.Println(pid)
			return
		}

		var l slog.Level = slog.LevelInfo

		if cmd.Debug {
			l = slog.LevelDebug
		}

		// update logging for long running proxy
		logger.Init(cmd.Execname,
			logger.SetLogLevel(l),
			logger.SetLogfile(logFile),
			logger.SetWriter(&timberjack.Logger{
				Filename:   logFile,
				MaxSize:    config.Get[int](cf, cf.Join("server", "log", "max-size")),
				MaxBackups: config.Get[int](cf, cf.Join("server", "log", "max-backups")),
				MaxAge:     config.Get[int](cf, cf.Join("server", "log", "stale-after")),
				Compress:   config.Get[bool](cf, cf.Join("server", "log", "compress")),
			}),
			logger.RotateOnStart(config.Get[bool](cf, cf.Join("server", "log", "rotate-on-start"))),
		)
		proxy(cf, pidfile)
	},
}

// endpoints returns the endpoints served by the proxy, relative to the
// configured `server.path`
func endpoints() []ims.Endpoint {
	return []ims.Endpoint{
		{Method: http.MethodGet, Path: "/{table}", Handler: withConfig(getRecords)},
		{Method: http.MethodPost, Path: "/{table}", Handler: withConfig(acceptRecord)},
	}
}

// withConfig adapts a handler that needs the configuration to the
// context set up by the shared server
func withConfig(h func(cf *config.Config, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cf, _, ok := ims.RequestContext(w, r)
		if !ok {
			return
		}
		h(cf, w, r)
	}
}

func proxy(cf *config.Config, pidfile string) {
	// init connection or fail early
	if _, err := snow.New(cf.Sub("servicenow")); err != nil {
		log.Error("cannot set up connection to ServiceNow", slog.Any("error", err))
		os.Exit(1)
	}

	// the configuration file is recorded in the PID file, as an absolute
	// path as it is read from other directories by the control commands
	configFile := cmd.ConfigFile()
	if configFile != "" {
		if abs, err := filepath.Abs(configFile); err == nil {
			configFile = abs
		}
	}

	err := ims.ServeWith(cf, endpoints(), ims.ServeOptions{
		PIDFile:    pidfile,
		ConfigFile: configFile,
		Load:       func() (*config.Config, error) { return cmd.LoadConfigFileFrom("proxy", configFile) },
		Check:      checkConfig,
	})
	if err != nil {
		log.Error("proxy failed", slog.Any("error", err))
		os.Exit(1)
	}
}
