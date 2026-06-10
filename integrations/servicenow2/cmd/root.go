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
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

// var cf *config.Config

var configFile, Execname, logFile string
var Debug bool

var log = cordial.Logger

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
		slog.SetDefault(cordial.LogInit(Execname, cordial.SetLogLevel(l)))
		log.Debug("cordial 'servicenow2' running", slog.String("executable", cordial.ExecutableName()), slog.String("version", cordial.VERSION))
	})
}

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

// LoadConfigFile reads in config file and ENV variables if set.
func LoadConfigFile(cmdName string) (cf *config.Config) {
	var err error

	configBasename := strings.Join([]string{Execname, cmdName}, ".")

	cf, err = config.Read(configBasename,
		config.AppName("geneos"),
		config.UseGlobal(),
		config.Format("yaml"),
		config.FilePath(configFile),
		config.MustExist(),
	)
	if err != nil {
		log.Error("failed to load configuration", slog.Any("error", err))
	}
	log.Debug("loaded config file",
		slog.String("path",
			config.Path(configBasename,
				config.AppName("geneos"),
				config.UseGlobal(),
				config.Format("yaml"),
				config.FilePath(configFile),
			),
		),
	)

	return
}
