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
	"os"
	dbg "runtime/debug"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/process"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, trace, quiet bool
var logFile string

var daemon bool

const (
	SummaryPath = "licensing/licences.csv"
	DetailsPath = "licensing/all_licences.csv"
)

func init() {
	GDNACmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	GDNACmd.PersistentFlags().MarkHidden("debug")

	GDNACmd.PersistentFlags().BoolVarP(&trace, "trace", "t", false, "enable trace output (SQL queries etc.) - implies debug")
	GDNACmd.PersistentFlags().MarkHidden("trace")

	// how to remove the help flag help text from the help output! Sigh...
	GDNACmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	GDNACmd.PersistentFlags().MarkHidden("help")

	GDNACmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "Use configuration `FILE`")
	GDNACmd.PersistentFlags().StringVarP(&logFile, "logfile", "l", execname+".log", "Write logs to `file`. Use '-' for console or "+os.DevNull+" for none")

	GDNACmd.Flags().SortFlags = false

	// cobra.OnInitialize(initConfig)
}

var cf *config.Config

//go:embed gdna.defaults.yaml
var defaults []byte

//go:embed _docs/gdna.md
var rootCmdDescription string

// GDNACmd represents the base command when called without any subcommands
var GDNACmd = &cobra.Command{
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
			return process.Daemon(nil, process.RemoveArgs, "-D", "--daemon")
		}

		initConfig(cmd)

		if !cf.IsSet("gdna.version") {
			cf.Set("gdna.version", cordial.VERSION)
		}

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cordial.RenderHelpAsMD(GDNACmd)

	err := GDNACmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command) {
	var err error
	var deferredlog string

	if cf == nil {
		opts := []config.FileOptions{
			config.SetAppName("geneos"),
			config.SetConfigFile(cfgFile),
			config.SetFileExtension("yaml"),
			config.WithDefaults(defaults, "yaml"),
			config.StopOnInternalDefaultsErrors(),
			// config.WatchConfig(configReloaded),
			// config.MergeSettings(), // this allows "defaults" slices to be merged with configs, which should not happen
			config.WithEnvs("GDNA", "_"),
		}

		cf, err = config.Load(execname, opts...)
		if err != nil {
			log.Fatal().Err(err).Msgf("loading from %s", config.Path(execname, opts...))
		}

		// use MustExists() to check for actual files
		opts = append(opts, config.MustExist())
		deferredlog = fmt.Sprintf("configuration loaded from %s", config.Path(execname, opts...))
	}

	// check if logfile is set on the command line, which overrides config
	if cmd != nil {
		if f := cmd.Flag("logfile"); f != nil && !f.Changed {
			logFile = cf.GetString("gdna.log.filename")
			if cmd.Annotations["defaultlog"] != "" {
				logFile = cmd.Annotations["defaultlog"]
			}
		}
	}
	cordial.LogInit(execname,
		cordial.SetLogfile(logFile),
		cordial.LumberjackOptions(&lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    cf.GetInt("gdna.log.max-size"),
			MaxBackups: cf.GetInt("gdna.log.max-backups"),
			MaxAge:     cf.GetInt("gdna.log.stale-after"),
			Compress:   cf.GetBool("gdna.log.compress"),
		}),
		cordial.RotateOnStart(cf.GetBool("gdna.log.rotate-on-start")),
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

// func configReloaded(_ fsnotify.Event) {
// 	// XXX protect this
// 	cf = nil
// 	initConfig(nil)
// 	log.Info().Msg("config reloaded")
// 	updateJobs()
// }
