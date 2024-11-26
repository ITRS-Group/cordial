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
	"fmt"
	"os"
	dbg "runtime/debug"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

var cf *config.Config
var debug, trace, quiet bool
var conffile string
var nowatchconfig bool
var hostname, hosttype, output string

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	rootCmd.PersistentFlags().MarkHidden("debug")

	rootCmd.PersistentFlags().BoolVarP(&trace, "trace", "t", false, "enable trace output (SQL queries etc.) - implies debug")
	rootCmd.PersistentFlags().MarkHidden("trace")

	rootCmd.PersistentFlags().StringVar(&conffile, "config", "", "path to configuration file")
	rootCmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", cordial.ExecutableName()+".log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	rootCmd.PersistentFlags().BoolVarP(&nowatchconfig, "nowatch", "N", false, "Do not watch configuration file for changes")

	// how to remove the help flag help text from the help output! Sigh...
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkHidden("help")

	rootCmd.Flags().StringVarP(&hostname, "host", "H", "", "create configuration for `HOSTNAME`")
	rootCmd.Flags().StringVarP(&hosttype, "type", "T", "", "override hosttype")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "output to `FILE`")

	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

//go:embed san-config.defaults.yaml
var defaults []byte

var uuidNS uuid.UUID

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig(cmd)

		uuidNS = uuid.MustParse(cf.GetString("server.namespace-uuid"))
	},
	TraverseChildren: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		if hostname == "" {
			fmt.Println("no hostname given (use `-H hostname`)")
			return
		}

		liveGateways := CheckGateways(cf)
		hosts, err := LoadHosts(cf)
		if err != nil {
			return
		}

		cs := &ConfigServer{conf: cf, hosts: hosts, gateways: liveGateways}

		out := os.Stdout
		if output != "" {
			out, err = os.Create(output)
			if err != nil {
				return
			}
		}
		var b []byte
		np, _ := cs.NetprobeConfig(hostname, hosttype)
		fmt.Fprint(out, xml.Header)
		b, err = xml.MarshalIndent(np, "", "    ")
		fmt.Fprintln(out, string(b))
		fmt.Fprintln(out)
		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
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
		opts := []config.FileOptions{
			config.SetAppName("geneos"),
			config.SetConfigFile(conffile),
			config.SetFileExtension("yaml"),
			config.WithDefaults(defaults, "yaml"),
			config.StopOnInternalDefaultsErrors(),
		}

		if !nowatchconfig {
			opts = append(opts,
				config.WatchConfig(func(e fsnotify.Event) {
					log.Info().Msgf("configuration changed, reloading %s and inventories", e.Name)
					Inventories.Range(func(key, value any) bool {
						// zero out modification check values
						inv := value.(*Inventory)
						inv.size = 0
						inv.cksum = ""
						inv.lastModified = time.Time{}
						Inventories.Store(key, inv)
						return true
					})
				}),
			)
		}

		cf, err = config.Load(cordial.ExecutableName(), opts...)
		if err != nil {
			log.Fatal().Err(err).Msgf("loading from %s", config.Path(cordial.ExecutableName(), opts...))
		}

		// use MustExists() to check for actual files
		opts = append(opts, config.MustExist())
		deferredlog = fmt.Sprintf("configuration loaded from %s", config.Path(cordial.ExecutableName(), opts...))
	}

	// check if logfile is set on the command line, which overrides config
	if cmd != nil {
		f := cmd.Flag("logfile")
		if f != nil && !f.Changed {
			logFile = cf.GetString("server.logs.path")
		}
	}

	cordial.LogInit(cordial.ExecutableName(),
		cordial.SetLogfile(logFile),
		cordial.LumberjackOptions(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    cf.GetInt("server.logs.max-size"),
			MaxBackups: cf.GetInt("server.logs.max-backups"),
			MaxAge:     cf.GetInt("server.logs.stale-after"),
			Compress:   cf.GetBool("server.logs.compress"),
		}),
		cordial.RotateOnStart(cf.GetBool("server.logs.rotate-on-start")),
	)

	switch {
	case quiet:
		zerolog.SetGlobalLevel(zerolog.Disabled)
	case trace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case debug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	info, _ := dbg.ReadBuildInfo()
	log.Info().Msgf("command %q version %s built with %s", cmd.Name(), cordial.VERSION, info.GoVersion)
	log.Debug().Msg(deferredlog)
}
