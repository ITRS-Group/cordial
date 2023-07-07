/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVarP(&conffile, "conf", "c", "", "override config file")

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
