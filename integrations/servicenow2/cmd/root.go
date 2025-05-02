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
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

var cf *config.Config

var conffile, execname string
var debug bool

func init() {
	RootCmd.PersistentFlags().StringVarP(&conffile, "conf", "c", "", "override config file")

	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	RootCmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	RootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	RootCmd.PersistentFlags().MarkHidden("help")

	RootCmd.Flags().SortFlags = false

	execname = path.Base(os.Args[0])
	cordial.LogInit(execname)
}

var RootCmd = &cobra.Command{
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
	cordial.RenderHelpAsMD(RootCmd)
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// loadConfigFile reads in config file and ENV variables if set.
func loadConfigFile(cmd *cobra.Command) {
	var err error

	configBasename := strings.Join([]string{execname, cmd.Name()}, ".")

	cf, err = config.Load(configBasename,
		config.SetAppName("geneos"),
		config.UseGlobal(),
		config.SetFileExtension("yaml"),
		config.SetConfigFile(conffile),
		config.MustExist(),
	)
	if err != nil {
		log.Fatal().Msgf("failed to load a configuration file %q from any expected location", configBasename+".yaml")
	}
	log.Debug().Msgf("loaded config file %s",
		config.Path(configBasename,
			config.SetAppName("geneos"),
			config.UseGlobal(),
			config.SetFileExtension("yaml"),
			config.SetConfigFile(conffile)),
	)
}
