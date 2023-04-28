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
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const pkgname = "cordial"

var (
	ErrInvalidArgs  error = errors.New("invalid arguments")
	ErrNotSupported error = errors.New("not supported")
)

var cfgFile string

var debug, quiet bool

func init() {
	cordial.LogInit(pkgname)

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are $HOME/.config/geneos.json, "+geneos.GlobalConfigPath+")")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	rootCmd.PersistentFlags().MarkHidden("debug")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")
	rootCmd.PersistentFlags().MarkHidden("quiet")

	// how to remove the help flag help text from the help output! Sigh...
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkHidden("help")

	// catch common abbreviations and typos
	rootCmd.PersistentFlags().SetNormalizeFunc(cmdNormalizeFunc)

	// this doesn't work as expected, define sort = false in each command
	// rootCmd.PersistentFlags().SortFlags = false
	rootCmd.Flags().SortFlags = false
}

const cmdName = "geneos"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   cmdName,
	Short: "Control your Geneos environment",
	Long: strings.ReplaceAll(`
Manage and control your Geneos environment. With |geneos| you can
initialise a new installation, add and remove components, control
processes and build template based configuration files for SANs and
more.
`, "|", "`"),
	Example: strings.ReplaceAll(`
$ geneos start
$ geneos ps
`, "|", "`"),
	SilenceUsage: true,
	Annotations:  make(map[string]string),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:            cordial.VERSION,
	DisableAutoGenTag:  true,
	DisableSuggestions: true,
	// SilenceErrors:      true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// check initialisation
		geneosdir := host.Geneos()
		if geneosdir == "" {
			// commands that do not require geneos home to be set - make this a pair of slices to test
			if !(cmd == versionCmd ||
				cmd == initCmd ||
				cmd == setUserCmd ||
				cmd == setGlobalCmd ||
				cmd == addHostCmd ||
				cmd.Parent() == initCmd ||
				cmd.Parent() == aesCmd ||
				len(host.RemoteHosts()) > 0) {
				cmd.SetUsageTemplate(" ")
				return fmt.Errorf("%s", strings.ReplaceAll(`
Geneos installation directory not set.

Use one of the following to fix this:

For an existing installation:
	$ geneos set user geneos=/path/to/geneos

To initialise a new installation:
	$ geneos init /path/to/geneos

For temporary usage:
	$ export ITRS_HOME=/path/to/geneos
`, "|", "`"))
			}
		}
		return parseArgs(cmd, args)
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

// catch misspelling and abbreviations of common flags
func cmdNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case "license":
		name = "licence"
	case "attr":
		name = "attribute"
	case "var":
		name = "variable"
	}
	return pflag.NormalizedName(name)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// `oldConfDir` is the original path to the user configuration,
	// typically directly in `~/geneos`. The LoadConfig() function
	// already looks in standardised user and global directories.
	oldConfDir, _ := config.UserConfigDir()

	cf, err := config.LoadConfig("geneos",
		config.SetConfigFile(cfgFile),
		config.Global(),
		config.AddConfigDirs(oldConfDir),
		config.MergeSettings(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	// support old set-ups
	cf.BindEnv("geneos", "ITRS_HOME")

	// auto env variables must be prefixed "ITRS_"
	cf.SetEnvPrefix("ITRS")
	replacer := strings.NewReplacer(".", "_")
	cf.SetEnvKeyReplacer(replacer)
	cf.AutomaticEnv()

	u, err := user.Current()
	username := "nobody"
	if err != nil {
		log.Error().Err(err).Msg("cannot get user details")
	} else {
		username = u.Username
	}
	// strip domain in case we are running on windows
	i := strings.Index(username, "\\")
	if i != -1 && len(username) >= i {
		username = username[i+1:]
	}
	cf.SetDefault("defaultuser", username)

	// manual alias+remove as the viper.RegisterAlias doesn't work as expected
	if cf.IsSet("itrshome") {
		if !cf.IsSet("geneos") {
			cf.Set("geneos", cf.GetString("itrshome"))
		}
		cf.Set("itrshome", nil)
	}

	// initialise after config loaded
	host.Init()
}
