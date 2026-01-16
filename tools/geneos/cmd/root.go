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

// Package cmd contains all the main commands for the `geneos` program
package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profiles"
)

const pkgname = "cordial"

var cfgFile string

// Hostname is the cmd package global host selected ny the `--host`/`-H` option
var Hostname string

// UserKeyFile is the path to the user's key file. It starts as DefaultUserKeyFile but can be changed.
var UserKeyFile = DefaultUserKeyfile

var debug, quiet bool

// DefaultUserKeyfile is the path to the user's key file as a
// config.Keyfile type
var DefaultUserKeyfile = config.KeyFile(
	config.Path("keyfile",
		config.SetAppName(cordial.ExecutableName()),
		config.SetFileExtension("aes"),
		config.IgnoreWorkingDir(),
	),
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
	// cordial.LogInit(pkgname)
	cobra.OnInitialize(func() {
		cordial.LogInit(pkgname)
		initConfig()
		geneos.Init(cordial.ExecutableName())
	})

	config.DefaultKeyDelimiter("::")
	config.ResetConfig(config.KeyDelimiter("::"))

	GeneosCmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are $HOME/.config/"+cordial.ExecutableName()+".json, "+
		config.Path(cordial.ExecutableName(),
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
			fmt.Println("")
			fmt.Println("💡 Use the `help` command to get more detailed help, instead of the `-h` flag, e.g. `" + strings.Replace(c.CommandPath(), cordial.ExecutableName(), cordial.ExecutableName()+" help", 1) + "`")

		} else {
			helpfunc(c, []string{})
		}
	})

	// run initialisers on internal packages, set the executable name
	// this has moves to the function called from cobra.OnInitialize
	// geneos.Init(cordial.ExecutableName())
}

//go:embed _docs/geneos.md
var geneosCmdDescription string

type CmdKeyType string

const CmdKey = CmdKeyType("data")

type CmdValType struct {
	sync.Mutex

	// the resolved component type, nil if none found
	component *geneos.Component

	// if true the absence of instance names or patterns means all instances
	globals bool

	// the list of instance names, build based on wildcarding and globs as applicable
	names []string

	// the list of non-instance names on the command line (invalid names, not match failures)
	params []string
}

func cmddata(command *cobra.Command) *CmdValType {
	ctx := command.Context()
	if ctx == nil {
		return nil
	}
	if a := ctx.Value(CmdKey); a != nil {
		if b, ok := a.(*CmdValType); ok {
			return b
		}
	}
	return nil
}

// GeneosCmd represents the base command when called without any subcommands
var GeneosCmd = &cobra.Command{
	Use:   cordial.ExecutableName() + " COMMAND [flags] [TYPE] [NAME...] [parameters...]",
	Short: "Take control of your Geneos environments",
	Long:  geneosCmdDescription,
	Example: strings.ReplaceAll(`
geneos init demo -u email@example.com -l
geneos ps
geneos restart
`, "|", "`"),
	// SilenceUsage: true,
	Annotations: map[string]string{
		CmdRequireHome: "true",
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	// SilenceErrors:      true, - this blocks all child errors too, don't uncomment it
	PersistentPreRunE: func(command *cobra.Command, args []string) (err error) {
		ctx := context.WithValue(context.Background(), CmdKey, &CmdValType{})
		command.SetContext(ctx)

		// "manually" parse root flags so that legacy commands get conf
		// file, debug etc.
		command.Root().ParseFlags(args)

		// check for AnnotationReplacedBy annotation, warn the user, run the new
		// command later (after prerun) but if the help flag is set
		// output the help for the new command and cleanly exit.
		var realcmd *cobra.Command

		if r, ok := command.Annotations[CmdReplacedBy]; ok {
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
		if r, ok := command.Annotations[CmdReplacedBy]; ok {
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
		if command.Annotations[CmdRequireHome] == "true" && geneos.LocalRoot() == "" && len(geneos.RemoteHosts(false)) == 0 {
			command.SetUsageTemplate(" ")
			return GeneosUnsetError
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

var configPath string

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Execname = cordial.ExecutableName()

	log.Debug().Msgf("cordial 'geneos' running as executable '%s', version %s", cordial.ExecutableName(), cordial.VERSION)

	// `oldConfDir` is the original path to the user configuration,
	// typically directly in `~/geneos`. The LoadConfig() function
	// already looks in standardised user and global directories. If
	// user lookup fails we get an empty path, which is fine - so ignore
	// errors.
	oldConfDir, _ := config.UserConfigDir()

	cf, err := config.Load(cordial.ExecutableName(),
		config.SetConfigFile(cfgFile),
		config.UseGlobal(),
		config.AddDirs(oldConfDir),
		config.MergeSettings(),
		config.IgnoreWorkingDir(),
		config.WithEnvs("ITRS", "_"),
		config.UseDefaults(false),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	configPath = config.Path(cordial.ExecutableName(),
		config.SetConfigFile(cfgFile),
		config.UseGlobal(),
		config.AddDirs(oldConfDir),
		config.MergeSettings(),
		config.IgnoreWorkingDir(),
		config.WithEnvs("ITRS", "_"),
		config.UseDefaults(false),
		config.MustExist(),
	)

	log.Debug().Msgf("configuration loaded from %s", configPath)

	// support old set-ups
	cf.BindEnv(cordial.ExecutableName(), "GENEOS_HOME", "ITRS_HOME")

	// manual alias+remove as the viper.RegisterAlias doesn't work as expected
	if cf.IsSet("itrshome") {
		if !cf.IsSet(cordial.ExecutableName()) {
			cf.Set(cordial.ExecutableName(), cf.GetString("itrshome"))
		}
		cf.Set("itrshome", nil)
	}

	// initialise after config loaded
	geneos.InitHosts(cordial.ExecutableName())

	// for now, always load profiles, even if not used
	_, err = profiles.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load profiles")
	}
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
