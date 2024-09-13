/*
Copyright © 2022 ITRS Group

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

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/integrations/servicenow/snow"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

var cf *config.Config

var conffile, execname string
var debug bool

func init() {
	cobra.OnInitialize(initConfig)

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
	Use:   "servicenow",
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error

	cf, err = config.Load(execname,
		config.SetAppName("geneos"),
		config.UseGlobal(),
		config.SetFileExtension("yaml"),
		config.SetConfigFile(conffile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

}

var snowConnection *snow.Connection

func InitializeConnection() *snow.Connection {
	if snowConnection != nil {
		return snowConnection
	}

	snowConnection = snow.InitializeConnection(cf)
	return snowConnection
}
