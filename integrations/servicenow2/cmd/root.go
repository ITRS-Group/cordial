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
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

// var cf *config.Config

// Execname TODO: Describe its purpose and intended use.
// TODO: Revisit unused declaration logFile; restore it only when needed.
var configFile, Execname string

// TODO: Revisit unused declaration logFile; restore it only when needed.
// var logFile string
// Debug TODO: Describe its purpose and intended use.
var Debug bool

var log = logger.Logger

func init() {
	Cmd.PersistentFlags().StringVarP(&configFile, "conf", "c", "", "override config file")

	Cmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	Cmd.Flags().SortFlags = false

	Execname = path.Base(os.Args[0])
	cobra.OnInitialize(func() {
		var l slog.Level
		if Debug {
			l = slog.LevelDebug
		}
		slog.SetDefault(logger.Init(Execname, logger.SetLogLevel(l)))
		log.Debug("cordial 'servicenow2' running", slog.String("executable", cordial.ExecutableName()), slog.String("version", cordial.VERSION))
	})
}

// Cmd TODO: Describe its purpose and intended use.
var Cmd = &cobra.Command{
	Use:   "servicenow2",
	Short: "Geneos to ServiceNow integration",
	Long:  ``,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:           cordial.VERSION,
	DisableAutoGenTag: true,
	SilenceUsage:      true,
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

// ConfigFile returns the configuration file given on the command line,
// which is empty if there was not one
func ConfigFile() string { return configFile }

// LoadConfigFile reads in the configuration file for the command, using
// the one given on the command line, if any, and ENV variables if set.
// An error is returned if the configuration cannot be read, in which case
// callers must not carry on with the (empty) configuration.
func LoadConfigFile(cmdName string) (cf *config.Config, err error) {
	return LoadConfigFileFrom(cmdName, configFile)
}

// LoadConfigFileFrom is like [LoadConfigFile] but for the file given, or
// if that is empty the default location. A new configuration is returned
// each time, so it can be used to reload a configuration without
// changing the one in use.
func LoadConfigFileFrom(cmdName, file string) (cf *config.Config, err error) {
	configBasename := strings.Join([]string{Execname, cmdName}, ".")

	cf, err = config.Read(configBasename,
		config.AppName("geneos"),
		config.Format("yaml"),
		config.FilePath(file),
		config.MustExist(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s configuration: %w", cmdName, err)
	}
	if cf == nil {
		return nil, fmt.Errorf("failed to load %s configuration", cmdName)
	}
	log.Debug("loaded config file",
		slog.String("path",
			config.Path(configBasename,
				config.AppName("geneos"),
				config.Format("yaml"),
				config.FilePath(file),
			),
		),
	)

	return cf, nil
}
