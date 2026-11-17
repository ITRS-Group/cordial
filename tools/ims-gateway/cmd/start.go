/*
Copyright © 2026 ITRS Group

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
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/DeRuina/timberjack"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims/control"
	"github.com/itrs-group/cordial/pkg/logger"

	_ "github.com/itrs-group/cordial/tools/ims-gateway/internal/sdp"
	_ "github.com/itrs-group/cordial/tools/ims-gateway/internal/snow"
)

var daemon bool
var pidFile string

// background is the control of the gateway when it is running in the
// background, with the commands `stop`, `restart`, `reload` and `status`
var background = &control.Daemon{
	Name:        "ims-gateway",
	PIDFileName: "ims-gateway.pid",
	PIDFile:     &pidFile,
	ConfigFile:  ConfigFile,
	Load:        LoadConfigFileFrom,
}

func init() {
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(background.Commands()...)

	startCmd.Flags().BoolVarP(&daemon, "daemon", "D", false, "Daemonise the proxy process")
	startCmd.Flags().StringVar(&pidFile, "pidfile", "", control.FlagUsage)
	startCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", "-", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	startCmd.Flags().SortFlags = false
}

//go:embed _docs/start.md
var startDescription string

// startCmd represents the proxy command
var startCmd = &cobra.Command{
	Use:          "start",
	Short:        "Run an ims-gateway",
	Long:         startDescription,
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		// load the configuration before daemonising so that a missing or
		// invalid file is reported to the user and not lost with the
		// background process
		cf, err := LoadConfigFile()
		if err != nil {
			log.Error("cannot start ims-gateway", slog.Any("error", err))
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
				logName = cordial.ExecutableName() + ".proxy.log"
				logArgs = append(logArgs, "--logfile", logName)
			}

			pid, err := control.Daemonize(pidfile, logArgs...)
			if err != nil {
				log.Error("failed to start ims-gateway in the background", slog.Any("error", err), slog.String("logfile", logName))
				os.Exit(1)
			}
			fmt.Println(pid)
			return
		}

		var l slog.Level = slog.LevelInfo
		if Debug {
			l = slog.LevelDebug
		}

		log = logger.Init(cordial.ExecutableName(),
			logger.SetLogLevel(l),
			logger.SetLogfile(logFile),
			logger.SetWriter(&timberjack.Logger{
				Filename:   logFile,
				MaxSize:    config.Get[int](cf, cf.Join("server", "logs", "max-size")),
				MaxBackups: config.Get[int](cf, cf.Join("server", "logs", "max-backups")),
				MaxAge:     config.Get[int](cf, cf.Join("server", "logs", "stale-after")),
				Compress:   config.Get[bool](cf, cf.Join("server", "logs", "compress")),
			}),
			logger.RotateOnStart(config.Get[bool](cf, cf.Join("server", "logs", "rotate-on-start"))),
		)
		startGateway(cf, pidfile)
	},
}
