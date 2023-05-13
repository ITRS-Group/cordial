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
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

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
var UserKeyFile config.KeyFile

var debug, quiet bool

var DefaultUserKeyfile = config.KeyFile(config.Path("keyfile", config.SetAppName(Execname), config.SetFileFormat("aes"), config.IgnoreWorkingDir()))

func init() {
	cordial.LogInit(pkgname)

	cobra.OnInitialize(initConfig)

	config.DefaultKeyDelimiter("::")
	config.ResetConfig(config.KeyDelimiter("::"))

	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are $HOME/.config/geneos.json, "+
		config.Path(Execname,
			config.IgnoreUserConfDir(),
			config.IgnoreWorkingDir())+
		")")
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	RootCmd.PersistentFlags().MarkHidden("debug")
	RootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")
	RootCmd.PersistentFlags().MarkHidden("quiet")

	// how to remove the help flag help text from the help output! Sigh...
	RootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	RootCmd.PersistentFlags().MarkHidden("help")

	// catch common abbreviations and typos
	RootCmd.PersistentFlags().SetNormalizeFunc(cmdNormalizeFunc)

	// this doesn't work as expected, define sort = false in each command
	// RootCmd.PersistentFlags().SortFlags = false
	RootCmd.Flags().SortFlags = false

	// run initialisers on internal packages, set the executable name
	geneos.Initialise(Execname)
	instance.Initialise(Execname)
}

var Execname = filepath.Base(os.Args[0])

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   Execname,
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
	Annotations: map[string]string{
		"needshomedir": "true",
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:            cordial.VERSION,
	DisableAutoGenTag:  true,
	DisableSuggestions: true,
	// SilenceErrors:      true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// check initialisation
		geneosdir := geneos.Root()
		if geneosdir == "" {
			// commands that do not require geneos home to be set - use
			// a const/var to iterate over to test this
			log.Debug().Msgf("parent? %v parent name %s name %s needshomedir %s", cmd.HasParent(), cmd.Parent().Name(), cmd.Name(), cmd.Annotations["needshomedir"])
			if cmd.Annotations["needshomedir"] == "true" {
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
	// RunE: lsCmd.RunE,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
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

	cf, err := config.Load(Execname,
		config.SetConfigFile(cfgFile),
		config.SetGlobal(),
		config.AddConfigDirs(oldConfDir),
		config.MergeSettings(),
		config.IgnoreWorkingDir(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	// support old set-ups
	cf.BindEnv(Execname, "ITRS_HOME")

	// auto env variables must be prefixed "ITRS_"
	cf.SetEnvPrefix("ITRS")
	replacer := strings.NewReplacer(".", "_")
	cf.SetEnvKeyReplacer(replacer)
	cf.AutomaticEnv()

	// manual alias+remove as the viper.RegisterAlias doesn't work as expected
	if cf.IsSet("itrshome") {
		if !cf.IsSet(Execname) {
			cf.Set(Execname, cf.GetString("itrshome"))
		}
		cf.Set("itrshome", nil)
	}

	// initialise after config loaded
	geneos.InitHosts(Execname)
}

// RunE runs a command in a sub-package to avoid import loops. It is
// named to align with the cobra struct member of the same name.
//
// The caller must have:
//
//	DisableFlagParsing: true,
//
// set in their command struct for flags to work. Then hook this
// function like this in the command struct:
//
//	RunE: func(command *cobra.Command, args []string) (err error) {
//	     return RunE(command.Root(), []string{"host", "ls"}, args)
//	},
func RunE(root *cobra.Command, path []string, args []string) (err error) {
	alias, newargs, err := root.Find(append(path, args...))
	if err != nil {
		return
	}
	alias.ParseFlags(newargs)
	// we have to explicitly test for the help flag for some reason
	if t, _ := alias.Flags().GetBool("help"); t {
		return alias.Help()
	}
	return alias.RunE(alias, alias.Flags().Args())
}
