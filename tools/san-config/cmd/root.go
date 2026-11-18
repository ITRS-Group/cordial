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
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"os"
	dbg "runtime/debug"

	"github.com/DeRuina/timberjack"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var cf *config.Config
var debug, trace, quiet bool
var conffile string
var nowatchconfig bool
var hostname, hosttype, output string

var log = logger.Logger

func init() {
	Cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")

	Cmd.PersistentFlags().BoolVarP(&trace, "trace", "t", false, "enable trace output (SQL queries etc.) - implies debug")
	Cmd.PersistentFlags().MarkHidden("trace")

	Cmd.PersistentFlags().StringVar(&conffile, "config", "", "path to configuration file")
	Cmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", cordial.ExecutableName()+".log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	Cmd.PersistentFlags().BoolVarP(&nowatchconfig, "nowatch", "N", false, "Do not watch configuration file for changes")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	Cmd.Flags().StringVarP(&hostname, "host", "H", "", "create configuration for `HOSTNAME`")
	Cmd.Flags().StringVarP(&hosttype, "type", "T", "", "override hosttype")
	Cmd.Flags().StringVarP(&output, "output", "o", "", "output to `FILE`")

	Cmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

//go:embed san-config.defaults.yaml
var defaults []byte

var uuidNS uuid.UUID

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
	Use:   cordial.ExecutableName(),
	Short: "Generate SAN configurations",
	Long:  ``,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		initConfig(cmd)

		uuidNS, err = uuid.Parse(config.Get[string](cf, cf.Join("server", "namespace-uuid")))
		if err != nil {
			return fmt.Errorf("invalid server.namespace-uuid: %w", err)
		}
		return
	},
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		if hostname == "" {
			return errors.New("no hostname given (use `-H hostname`)")
		}

		liveGateways := CheckGateways(cf)
		hosts, err := LoadHosts(cf)
		if err != nil {
			return
		}

		cs := &ConfigServer{conf: cf, hosts: hosts, gateways: liveGateways}

		np, finalHosttype := cs.NetprobeConfig(hostname, hosttype)
		if np == nil {
			return fmt.Errorf("no configuration could be generated for host %q (see log for details)", hostname)
		}
		log.Debug("generated config", slog.String("hostname", hostname), slog.String("type", finalHosttype))

		b, err := xml.MarshalIndent(np, "", "    ")
		if err != nil {
			return
		}

		out := os.Stdout
		if output != "" {
			if out, err = os.Create(output); err != nil {
				return
			}
		}
		fmt.Fprint(out, xml.Header)
		fmt.Fprintln(out, string(b))
		fmt.Fprintln(out)
		if output != "" {
			return out.Close()
		}
		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := Cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// var cf *config.Config
var logFile string

func initConfig(cmd *cobra.Command) {
	var err error
	var deferredlog string

	if cf == nil {
		opts := []config.FileOption{
			config.AppName("geneos"),
			config.FilePath(conffile),
			config.Format("yaml"),
			config.WithDefaults(defaults, "yaml"),
			config.StopOnInternalDefaultsErrors(),
		}

		if !nowatchconfig {
			opts = append(opts,
				config.WatchConfig(func(e fsnotify.Event) {
					log.Info("configuration changed, reloading", slog.String("file", e.Name))
					// clear the previous state, so that all inventories are
					// reloaded on the next check
					Inventories.Clear()
				}),
			)
		}

		cf, err = config.Read(cordial.ExecutableName(), opts...)
		if err != nil {
			log.Error("Failed to load configuration", slog.Any("error", err), slog.String("path", config.Path(cordial.ExecutableName(), opts...)))
			os.Exit(1)
		}

		// use MustExists() to check for actual files
		opts = append(opts, config.MustExist())
		deferredlog = fmt.Sprintf("configuration loaded from %s", config.Path(cordial.ExecutableName(), opts...))
	}

	// the log file from the command line, if given, overrides the
	// configuration. Both the root `--logfile/-l` and the `server
	// --logfile/-L` flags are named `logfile`
	logFile = config.Get[string](cf, "server.logs.path")
	if cmd != nil {
		if f := cmd.Flag("logfile"); f != nil && f.Changed {
			logFile = f.Value.String()
		}
	}

	log = logger.Init(cordial.ExecutableName(),
		logger.SetLogfile(logFile),
		logger.SetWriter(&timberjack.Logger{
			Filename:   logFile,
			MaxSize:    logSetting[int](cf, "size", "max-size"),
			MaxBackups: logSetting[int](cf, "backups", "max-backups"),
			MaxAge:     logSetting[int](cf, "age", "stale-after"),
			Compress:   config.Get[bool](cf, cf.Join("server", "logs", "compress")),
		}),
		logger.RotateOnStart(logSetting[bool](cf, "rotate-at-start", "rotate-on-start")),
	)

	switch {
	case quiet:
		logger.LogLevel.Set(slog.LevelError)
	case trace, debug:
		logger.LogLevel.Set(slog.LevelDebug)
	default:
		logger.LogLevel.Set(slog.LevelInfo)
	}

	goVersion := "unknown"
	if info, ok := dbg.ReadBuildInfo(); ok {
		goVersion = info.GoVersion
	}
	command := cordial.ExecutableName()
	if cmd != nil {
		command = cmd.Name()
	}
	log.Info("command version", slog.String("command", command), slog.String("version", cordial.VERSION), slog.String("go_version", goVersion))
	log.Debug(deferredlog)
}

// logSetting returns the `server.logs` setting for the documented name
// or, if that is not set, the older name that was previously read by the
// program but never documented
func logSetting[T any](cf *config.Config, name, legacy string) T {
	if cf.IsSet(cf.Join("server", "logs", legacy)) {
		return config.Get[T](cf, cf.Join("server", "logs", legacy))
	}
	return config.Get[T](cf, cf.Join("server", "logs", name))
}
