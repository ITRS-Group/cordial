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
	"path/filepath"
	"strings"

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

	rootCmd.PersistentFlags().StringVarP(&conffile, "conf", "c", "", "override config file")

	// how to remove the help flag help text from the help output! Sigh...
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkHidden("help")

	rootCmd.Flags().SortFlags = false

	execname = filepath.Base(os.Args[0])
	cordial.LogInit(execname)
}

var rootCmd = &cobra.Command{
	Use:   "servicenow",
	Short: "Geneos to ServiceNow integration",
	Long: strings.ReplaceAll(`
`, "|", "`"),
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
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func RootCmd() *cobra.Command {
	return rootCmd
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error

	cf, err = config.LoadConfig(execname,
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
