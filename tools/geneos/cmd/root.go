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

// Package cmd contains all the main commands for the `geneos` program
package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const pkgname = "cordial"

// default command name for pre-init
const execname = "geneos"

// Execname is the basename, without extension, of the underlying binary
// used to start the program. The initialising routines evaluate
// symlinks etc.
//
// initialise to sensible default
var Execname = execname

var cfgFile string

// Hostname is the cmd package global host selected ny the `--host`/`-H` option
var Hostname string

// UserKeyFile is the path to the user's key file. It starts as DefaultUserKeyFile but can be changed.
var UserKeyFile config.KeyFile

var debug, quiet bool

// DefaultUserKeyfile is the path to the user's key file as a
// config.Keyfile type
var DefaultUserKeyfile = config.KeyFile(
	config.Path("keyfile",
		config.SetAppName(Execname),
		config.SetFileExtension("aes"),
		config.IgnoreWorkingDir()),
)

var GeneosUnsetError = errors.New(strings.ReplaceAll(`Geneos location not set.

You can do one of the following:
* Run |geneos config set geneos=/PATH| (where |/PATH| is the location of the Geneos installation)
* Run |geneos init| or |geneos init /PATH| to initialise an installation
  * There are also variations on the |init| command, please see help for the command
* Set the |GENEOS_HOME| or |ITRS_HOME| environment variables, either once or in your |.profile|:
  * |export GENEOS_HOME=/PATH|

`, "|", "`"))

func init() {
	cordial.LogInit(pkgname)
	cobra.OnInitialize(initConfig)

	config.DefaultKeyDelimiter("::")
	config.ResetConfig(config.KeyDelimiter("::"))

	GeneosCmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are $HOME/.config/geneos.json, "+
		config.Path(Execname,
			config.IgnoreUserConfDir(),
			config.IgnoreWorkingDir())+
		")")
	GeneosCmd.PersistentFlags().StringVarP(&Hostname, "host", "H", "all", "Limit actions to `HOSTNAME` (not for commands given instance@host parameters)")
	GeneosCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	GeneosCmd.PersistentFlags().MarkHidden("debug")
	GeneosCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")
	GeneosCmd.PersistentFlags().MarkHidden("quiet")

	// how to remove the help flag help text from the help output! Sigh...
	GeneosCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	GeneosCmd.PersistentFlags().MarkHidden("help")

	// catch common abbreviations and typos
	GeneosCmd.PersistentFlags().SetNormalizeFunc(cmdNormalizeFunc)

	// this doesn't work as expected, define sort = false in each command
	// RootCmd.PersistentFlags().SortFlags = false
	GeneosCmd.Flags().SortFlags = false

	// save orig help func, check if this is a --help call or a help command
	helpfunc := GeneosCmd.HelpFunc()
	GeneosCmd.SetHelpFunc(func(c *cobra.Command, s []string) {
		if b, _ := c.Flags().GetBool("help"); b {
			c.Usage()
		} else {
			helpfunc(c, []string{})
		}
	})

	// run initialisers on internal packages, set the executable name
	geneos.Init(Execname)
}

//go:embed _docs/geneos.md
var geneosCmdDescription string

// GeneosCmd represents the base command when called without any subcommands
var GeneosCmd = &cobra.Command{
	Use:   Execname + " COMMAND [flags] [TYPE] [NAME...] [parameters...]",
	Short: "Take control of your Geneos environments",
	Long:  geneosCmdDescription,
	Example: strings.ReplaceAll(`
geneos init demo -u email@example.com -l
geneos ps
geneos restart
`, "|", "`"),
	// SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationNeedsHome: "true",
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// SilenceErrors:      true, // this blocks all child errors too...
	// don't uncomment it
	PersistentPreRunE: func(command *cobra.Command, args []string) (err error) {
		// "manually" parse root flags so that legacy commands get conf
		// file, debug etc.
		command.Root().ParseFlags(args)

		// if we have no args and have explicit set, then go bang. check component type
		if command.Annotations[AnnotationWildcard] == "explicit" {
			rootargs := command.Root().Flags().Args()
			if len(rootargs) == 0 {
				return fmt.Errorf("%w: %q requires at least TYPE or one or more NAME arguments", geneos.ErrInvalidArgs, command.Name())
			}
		}

		// check for AnnotationReplacedBy annotation, warn the user, run the new
		// command later (after prerun) but if the help flag is set
		// output the help for the new command and cleanly exit.
		var realcmd *cobra.Command

		if r, ok := command.Annotations[AnnotationReplacedBy]; ok {
			var newargs []string
			realcmd, newargs, err = command.Root().Find(append(strings.Split(r, " "), args...))
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			if realcmd != nil {
				fmt.Printf("*** Please note that the %q command has been replaced by %q\n\n", command.CommandPath(), realcmd.CommandPath())
				command.RunE = func(cmd *cobra.Command, args []string) error {
					realcmd.ParseFlags(newargs)
					ParseArgs(realcmd, newargs)
					return realcmd.RunE(realcmd, realcmd.Flags().Args())
				}
			}
		}

		// same as above, but no warning message (XXX - can't recall why, indirection?)
		if r, ok := command.Annotations[AnnotationReplacedBy]; ok {
			var newargs []string
			realcmd, newargs, err = command.Root().Find(append(strings.Split(r, " "), args...))
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			if realcmd != nil {
				command.RunE = func(cmd *cobra.Command, args []string) error {
					realcmd.ParseFlags(newargs)
					ParseArgs(realcmd, newargs)
					return realcmd.RunE(realcmd, realcmd.Flags().Args())
				}
			}
		}

		if realcmd != nil {
			if t, _ := command.Flags().GetBool("help"); t {
				command.RunE = nil
				// Run cannot be nil
				command.Run = func(cmd *cobra.Command, args []string) {
					realcmd.Usage()
				}
				return nil
			}
		}

		if t, _ := command.Flags().GetBool("help"); t { // || command.Name() == "help" {
			command.RunE = nil
			// Run cannot be nil
			command.Run = func(cmd *cobra.Command, args []string) {
				command.Usage()
			}
			return nil
		}

		// check initialisation
		if geneos.LocalRoot() == "" && len(geneos.RemoteHosts(false)) == 0 {
			if command.Annotations[AnnotationNeedsHome] == "true" {
				command.SetUsageTemplate(" ")
				return GeneosUnsetError
			}
		}
		if command.Name() == "help" {
			// don't parse args if the command is a help
			return nil
		}

		return ParseArgs(command, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	cordial.RenderHelpAsMD(GeneosCmd)

	err := GeneosCmd.Execute()
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

	Execname = cordial.ExecutableName()

	log.Debug().Msgf("cordial 'geneos' running as '%s', version %s", Execname, cordial.VERSION)

	// `oldConfDir` is the original path to the user configuration,
	// typically directly in `~/geneos`. The LoadConfig() function
	// already looks in standardised user and global directories.
	oldConfDir, _ := config.UserConfigDir()

	cf, err := config.Load(Execname,
		config.SetConfigFile(cfgFile),
		config.UseGlobal(),
		config.AddDirs(oldConfDir),
		config.MergeSettings(),
		config.IgnoreWorkingDir(),
		config.WithEnvs("ITRS", "_"),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// support old set-ups
	cf.BindEnv(Execname, "GENEOS_HOME", "ITRS_HOME")

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
	ParseArgs(alias, newargs)

	return alias.RunE(alias, alias.Flags().Args())
}
