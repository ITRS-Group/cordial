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
	"runtime"
	"strings"

	"github.com/DeRuina/timberjack"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, trace bool
var logFile string

var daemon bool

const (
	// SummaryPath TODO: Describe its purpose and intended use.
	SummaryPath = "licensing/licences.csv"
	// DetailsPath TODO: Describe its purpose and intended use.
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

// levelTrace is the log level enabled by the `--trace` flag, below
// debug, for the output of every SQL query
const levelTrace = slog.LevelDebug - 4

// logLevel returns the log level selected by the command line flags,
// where trace includes debug
func logLevel() slog.Level {
	switch {
	case trace:
		return levelTrace
	case debug:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

// logMaxAge returns the number of days to keep rotated log files, which
// is the documented `gdna.log.max-age`. The undocumented
// `gdna.log.stale-after` is only used, as it was before, if it is set.
func logMaxAge(cf *config.Config) int {
	if key := cf.Join("gdna", "log", "stale-after"); cf.IsSet(key) {
		return config.Get[int](cf, key)
	}
	return config.Get[int](cf, cf.Join("gdna", "log", "max-age"), config.DefaultValue(14))
}

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
			return daemonize(cmd)
		}

		initConfig(cmd)

		if err = validateLayerReportSelection(reportNames); err != nil {
			return err
		}

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

func validateLayerConfig(cf *config.Config) error {
	for _, rename := range []struct {
		old string
		new string
	}{
		{"plugins.level1-optional", "plugins.layer1-optional"},
		{"plugins.level1", "plugins.layer1"},
		{"plugins.level2", "plugins.layer2"},
		{"reports." + legacyCoverageReportName, "reports." + coverageReportName},
	} {
		if cf.IsSet(rename.old) {
			return fmt.Errorf("configuration key %q was renamed to %q; update your GDNA configuration", rename.old, rename.new)
		}
		prefix := "GDNA_" + strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(rename.old))
		for _, env := range os.Environ() {
			name, value, _ := strings.Cut(env, "=")
			if value != "" && (name == prefix || strings.HasPrefix(name, prefix+"_")) {
				return fmt.Errorf("environment variable %q uses configuration key %q, renamed to %q", name, rename.old, rename.new)
			}
		}
	}
	if err := validateLayerReportSelection(config.Get[string](cf, "email.body-reports")); err != nil {
		return fmt.Errorf("email.body-reports: %w", err)
	}
	return nil
}

// configOptions returns the options to read the configuration from the
// file, or the default locations if empty
func configOptions(file string) []config.FileOption {
	return []config.FileOption{
		config.AppName("geneos"),
		config.FilePath(file),
		config.Format("yaml"),
		config.WithDefaults(defaults, "yaml"),
		config.StopOnInternalDefaultsErrors(),
		// config.WatchConfig(configReloaded),
		// config.MergeSettings(), // this allows "defaults" slices to be merged with configs, which should not happen
		config.WithEnvs("GDNA", "_"),
	}
}

// checkConfigFile returns an error if a configuration file was given, on
// the command line, but cannot be read, as otherwise it would be silently
// ignored and the defaults used instead
func checkConfigFile(file string) error {
	if file == "" {
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("configuration file: %w", err)
	}
	return f.Close()
}

// loadConfig reads and checks the configuration from file, or the
// default locations if empty, without changing any global state, so
// that it can be used to check a configuration and to reload it
func loadConfig(file string) (*config.Config, error) {
	if err := checkConfigFile(file); err != nil {
		return nil, err
	}
	cf, err := config.Read(execname, configOptions(file)...)
	if err != nil {
		return nil, err
	}
	if err = validateLayerConfig(cf); err != nil {
		return nil, err
	}
	bindFilterSQL(cf)
	return cf, nil
}

// bindFilterSQL makes the `filters` expansion prefix, used in queries,
// available to every expansion using cf, and only cf. The prefix is
// bound to the configuration when it is loaded, so that a reload does not
// change the queries of runs that are using the previous one.
func bindFilterSQL(cf *config.Config) {
	cf.SetDefaultExpandOptions(append(cf.DefaultExpandOptions(), buildFilterSQL(cf))...)
}

func initConfig(cmd *cobra.Command) {
	var err error
	var deferredlog string

	if cf == nil {
		if err = checkConfigFile(cfgFile); err != nil {
			log.Error("loading config", slog.Any("error", err))
			os.Exit(1)
		}

		cf, err = config.Read(execname, configOptions(cfgFile)...)
		if err != nil {
			log.Error("loading config",
				slog.Any("error", err),
				slog.String("path", config.Path(execname, configOptions(cfgFile)...)),
			)
			os.Exit(1)
		}

		bindFilterSQL(cf)

		// save log for after log setup
		deferredlog = fmt.Sprintf("configuration loaded from %s",
			config.Path(execname, append(configOptions(cfgFile), config.MustExist())...),
		)
	}

	if err = validateLayerConfig(cf); err != nil {
		log.Error("loading config", slog.Any("error", err))
		os.Exit(1)
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

	log = logger.Init(execname,
		logger.SetLogfile(logFile),
		logger.SetWriter(&timberjack.Logger{
			Filename:   logFile,
			MaxSize:    config.Get[int](cf, cf.Join("gdna", "log", "max-size"), config.DefaultValue(10)),
			MaxBackups: config.Get[int](cf, cf.Join("gdna", "log", "max-backups"), config.DefaultValue(5)),
			MaxAge:     logMaxAge(cf),
			Compress:   config.Get[bool](cf, cf.Join("gdna", "log", "compress"), config.DefaultValue(true)),
		}),
		logger.RotateOnStart(config.Get[bool](cf, cf.Join("gdna", "log", "rotate-on-start"), config.DefaultValue(false))),
		logger.SetLogLevel(logLevel()),
	)

	log.Info(fmt.Sprintf("command %q version %s built with %s", cmd.Name(), cordial.VERSION, runtime.Version()))
	log.Info(deferredlog)
}
