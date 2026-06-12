/*
Copyright © 2024 ITRS Group

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
	dbg "runtime/debug"

	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, trace bool
var logFile string

var daemon bool

const (
	SummaryPath = "licensing/licences.csv"
	DetailsPath = "licensing/all_licences.csv"
)

func init() {
	Cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")

	Cmd.PersistentFlags().BoolVarP(&trace, "trace", "t", false, "enable trace output (SQL queries etc.) - implies debug")
	Cmd.PersistentFlags().MarkHidden("trace")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	Cmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "Use configuration `FILE`")
	Cmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", execname+".log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	Cmd.Flags().SortFlags = false

	// cobra.OnInitialize(initConfig)
}

var cf *config.Config

//go:embed gdna.defaults.yaml
var defaults []byte

//go:embed _docs/gdna.md
var rootCmdDescription string

var log *slog.Logger = slog.Default()

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
	Use:   "gdna [FLAGS...]",
	Short: "Process Geneos License Usage Data",
	Long:  rootCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if daemon {
			return process.Daemon(nil, nil, process.RemoveArgs, "-D", "--daemon")
		}

		initConfig(cmd)

		if v, ok := config.Lookup[string](cf, cf.Join("gdna", "version")); !ok || v == "" {
			config.Set(cf, cf.Join("gdna", "version"), cordial.VERSION)
		}

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cordial.RenderHelpAsMD(Cmd)

	err := Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command) {
	var err error
	var deferredlog string
	var loglevel slog.Level

	if cf == nil {
		cf, err = config.Read(execname,
			config.AppName("geneos"),
			config.FilePath(cfgFile),
			config.Format("yaml"),
			config.WithDefaults(defaults, "yaml"),
			config.StopOnInternalDefaultsErrors(),
			// config.WatchConfig(configReloaded),
			// config.MergeSettings(), // this allows "defaults" slices to be merged with configs, which should not happen
			config.WithEnvs("GDNA", "_"),
		)
		if err != nil {
			log.Error("loading config",
				slog.Any("error", err),
				slog.String("path",
					config.Path(execname,
						config.AppName("geneos"),
						config.FilePath(cfgFile),
						config.Format("yaml"),
						config.WithDefaults(defaults, "yaml"),
						config.StopOnInternalDefaultsErrors(),
						// config.WatchConfig(configReloaded),
						// config.MergeSettings(), // this allows "defaults" slices to be merged with configs, which should not happen
						config.WithEnvs("GDNA", "_"),
					),
				),
			)
			os.Exit(1)
		}

		// save log for after log setup
		deferredlog = fmt.Sprintf("configuration loaded from %s",
			config.Path(execname,
				config.AppName("geneos"),
				config.FilePath(cfgFile),
				config.Format("yaml"),
				config.WithDefaults(defaults, "yaml"),
				config.StopOnInternalDefaultsErrors(),
				// config.WatchConfig(configReloaded),
				// config.MergeSettings(), // this allows "defaults" slices to be merged with configs, which should not happen
				config.WithEnvs("GDNA", "_"),
				config.MustExist(),
			),
		)
	}

	// check if logfile is set on the command line, which overrides config
	if cmd != nil {
		if f := cmd.Flag("logfile"); f != nil && !f.Changed {
			logFile = config.Get[string](cf, "gdna.log.filename")
			if cmd.Annotations["defaultlog"] != "" {
				logFile = cmd.Annotations["defaultlog"]
			}
		}
	}

	log = cordial.LogInit(execname,
		cordial.SetLogfile(logFile),
		cordial.LumberjackOptions(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    config.Get[int](cf, cf.Join("gdna", "log", "max-size"), config.DefaultValue(10)),
			MaxBackups: config.Get[int](cf, cf.Join("gdna", "log", "max-backups"), config.DefaultValue(5)),
			MaxAge:     config.Get[int](cf, cf.Join("gdna", "log", "stale-after"), config.DefaultValue(30)),
			Compress:   config.Get[bool](cf, cf.Join("gdna", "log", "compress"), config.DefaultValue(true)),
		}),
		cordial.RotateOnStart(config.Get[bool](cf, cf.Join("gdna", "log", "rotate-on-start"), config.DefaultValue(false))),
		cordial.SetLogLevel(loglevel),
	)

	info, _ := dbg.ReadBuildInfo()
	log.Info(fmt.Sprintf("command %q version %s built with %s", cmd.Name(), cordial.VERSION, info.GoVersion))
	log.Info(deferredlog)
}
