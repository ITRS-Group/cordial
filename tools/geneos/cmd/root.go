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
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

// give these more convenient names and also shadow the std log
// package for normal logging
var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
)

var (
	ErrInvalidArgs  error = errors.New("invalid arguments")
	ErrNotSupported error = errors.New("not supported")
)

//go:embed VERSION
var VERSION string

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "geneos",
	Short: "Control your Geneos environment",
	Long: `Control your Geneos environment. With 'geneos' you can initialise
a new installation, add and remove components, control processes and build
template based configuration files for SANs and new gateways.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations:           make(map[string]string),
	Version:               VERSION,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// check initialisation
		geneosdir := host.Geneos()
		if geneosdir == "" {
			// only allow init through
			if cmd != initCmd && cmd != setUserCmd && cmd != setGlobalCmd {
				cmd.SetUsageTemplate(" ")
				return fmt.Errorf("%s", `Installation directory is not set.

You can fix this by doing one of the following:

1. Create a new Geneos environment:

	$ geneos init

	or, if not in your home directory:

	$ geneos init /path/to/geneos

2. Set the ITRS_HOME environment:

	$ export ITRS_HOME=/path/to/geneos

3. Set the Geneos path in your user's configuration file:

	$ geneos set user geneos=/path/to/geneos

3. Set the Geneos path in the global configuration file (usually as root):

	# echo '{ "Geneos": "/path/to/geneos" }' > `+geneos.GlobalConfigPath)
			}
		}

		parseArgs(cmd, args)
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

func RootCmd() *cobra.Command {
	return rootCmd
}

var debug, quiet bool

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are $HOME/.config/geneos.json, "+geneos.GlobalConfigPath+")")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	rootCmd.PersistentFlags().MarkHidden("debug")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")

	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "username for downloads")
	// rootCmd.PersistentFlags().BoolVarP(&passwordPrompt, "password", "p", false, "prompt for a password, only valid for downloads and in conjunction with -u")
	rootCmd.PersistentFlags().StringVarP(&passwordFile, "pwfile", "P", "", "path to password file, only valid for downloads and in conjunction with -u")

	rootCmd.PersistentFlags().SortFlags = false
	rootCmd.Flags().SortFlags = false
}

var username, passwordFile string

// var passwordPrompt bool

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if quiet {
		log.SetOutput(ioutil.Discard)
	} else if debug {
		logger.EnableDebugLog()
	}

	oldConfDir, _ := os.UserConfigDir()

	cf, err := config.LoadConfig("geneos", config.SetConfigFile(cfgFile), config.UseGlobal(), config.AddConfigDirs(oldConfDir))
	if err != nil {
		log.Fatalln(err)
	}
	// support old set-ups
	cf.BindEnv("geneos", "ITRS_HOME")

	// auto env variables must be prefixed "ITRS_"
	cf.SetEnvPrefix("ITRS")
	replacer := strings.NewReplacer(".", "_")
	cf.SetEnvKeyReplacer(replacer)
	cf.AutomaticEnv()

	u, _ := user.Current()
	cf.SetDefault("defaultuser", u.Username)

	// manual alias+remove as the viper.RegisterAlias doesn't work as expected
	if cf.IsSet("itrshome") {
		if !cf.IsSet("geneos") {
			cf.Set("geneos", cf.GetString("itrshome"))
		}
		cf.Set("itrshome", nil)
	}

	if username != "" {
		cf.Set("download.username", username)
	}

	if passwordFile != "" {
		cf.Set("download.password", utils.ReadPasswordFile(passwordFile))
	} else if cf.GetString("download.username") != "" {
		cf.Set("download.password", utils.ReadPasswordPrompt())
		// only ask once
		// passwordPrompt = false
	}

	// initialise after config loaded
	host.Init()
}
