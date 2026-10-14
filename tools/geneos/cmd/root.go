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
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"src.elv.sh/pkg/md"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
)

const (
	packageName = "cordial"
)

// local vars and flag
var (
	cfgFile      string
	profilesFile string
	debug        bool
	quiet        bool
	log          = logger.Logger
)

// exported vars
var (
	// "global" profiles configuration
	Profiles *config.Config

	// Hostname is the cmd package global host selected ny the `--host`/`-H` option
	Hostname string

	// UserKeyFile is the path to the user's key file. It starts as DefaultUserKeyFile but can be changed.
	UserKeyFile = geneos.DefaultUserKeyfile

	AllowRoot bool
)

// GeneosUnsetError is returned when no existing Geneos environment is
// found. It is rendered through a simple markdown renderer for TTY
// output.
var GeneosUnsetError = errors.New(md.RenderString((strings.ReplaceAll(

	`No existing Geneos environment found.

You can do one of the following:

* For a new environment, run one of the |geneos init| commands. For more details use |geneos help init|

* To adopt an existing environment, which you currently manage using scripts like |gatewayctl| and |netprobectl|, use

    geneos config set geneos=/PATH
	
  where "/PATH" is the location of the Geneos installation. Also consider using |geneos migrate -X| to replace those script with symbolic links to the this program.

* Another option, for advanced uses is to set the |GENEOS_HOME| or |ITRS_HOME| environment variables, either once or in your |${HOME}/.profile| or |${HOME}/.bash_profile|:

    export GENEOS_HOME=/PATH

`, "|", "`")), &md.TTYCodec{Width: 76}))

func init() {
	cobra.OnInitialize(func() {
		// initialise standard logging before loading global config
		logger.Init(packageName)

		_, err := initGlobalConfig()
		if err != nil {
			log.Error("failed to initialize global config", slog.Any("error", err))
		}

		log.Debug("flags parsed?", slog.Bool("parsed", Cmd.Flags().Parsed()))

		Profiles, err = profile.ReadConfig(profilesFile)
		if err != nil {
			log.Error("failed to initialize profiles config", slog.Any("error", err))
		}

		// if we are initialising and environment via init, deploy etc.
		// then this is not run, do it in those
		if geneos.LocalRoot() != "" {
			if err := geneos.InitGlobalAudit(); err != nil {
				log.Error("failed to initialize global audit", slog.Any("error", err))
			}
		}
		geneos.Init(cordial.ExecutableName())
	})

	config.DefaultKeyDelimiter("::")
	config.ResetConfig(config.KeyDelimiter("::"))

	Cmd.PersistentFlags().StringVarP(&cfgFile, "config", "G", "", "config file (defaults are "+
		config.AbbreviateHome(
			config.Path(cordial.ExecutableName(),
				config.SkipSystemDir(),
				config.SkipWorkingDir(),
			),
		)+
		", "+
		config.Path(cordial.ExecutableName(),
			config.SkipUserConfDir(),
			config.SkipWorkingDir())+
		")")

	Cmd.PersistentFlags().StringVar(&profilesFile, "profiles", "", "profiles configuration (defaults are "+
		config.AbbreviateHome(
			config.Path("profiles",
				config.AppName(cordial.ExecutableName()),
				config.Format("yaml"),
				config.SkipSystemDir(),
				config.SkipWorkingDir(),
			),
		)+
		", "+
		config.Path("profiles",
			config.AppName(cordial.ExecutableName()),
			config.Format("yaml"),
			config.SkipUserConfDir(),
			config.SkipWorkingDir())+
		")")

	Cmd.PersistentFlags().BoolVar(&AllowRoot, "allow-root", false, "allow running as root (not recommended)")
	Cmd.PersistentFlags().StringVarP(&Hostname, "host", "H", "all", "Limit actions to `HOSTNAME` (not for commands given instance@host parameters)")
	Cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")
	Cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet mode")
	Cmd.PersistentFlags().MarkHidden("quiet")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	// catch common abbreviations and typos
	Cmd.PersistentFlags().SetNormalizeFunc(cmdNormalizeFunc)

	// this doesn't work as expected, define sort = false in each command
	// RootCmd.PersistentFlags().SortFlags = false
	Cmd.Flags().SortFlags = false

	// save orig help func, check if this is a --help call or a help command
	helpfunc := Cmd.HelpFunc()
	Cmd.SetHelpFunc(func(c *cobra.Command, s []string) {
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

// CmdValType is the struct used to store the results of parsing the
// command line arguments in a PreRun function and pass them to the main
// RunE function.
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

// cmddata is a helper to get the command data struct from the command
// context. It returns nil if not found or of the wrong type.
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

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
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
		CmdAllowRoot:   "true",
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

		// boot-strap profile function map, no that flags are parsed
		// set defaults for all profile expansion

		profile.SetExpandOptions(command.Flags(), Profiles)

		// check if user is root, and only proceed if conditions are met:
		// * the command has the `allow-root` annotation set to "true", or
		// * the command is `help`, or
		// * the global `allow-root` flag is set to true (which sets the
		//   `allow-root` annotation to "true" for all commands)
		// otherwise return an error
		if os.Geteuid() == 0 && !AllowRoot && command.Annotations[CmdAllowRoot] != "true" && command.Name() != "help" {
			return fmt.Errorf("running as root is not allowed, use --allow-root to override")
		}

		// check for AnnotationReplacedBy annotation, warn the user, run the new
		// command later (after prerun) but if the help flag is set
		// output the help for the new command and cleanly exit.
		var realcmd *cobra.Command

		if r, ok := command.Annotations[CmdReplacedBy]; ok {
			var newargs []string
			realcmd, newargs, err = command.Root().Find(append(strings.Split(r, " "), args...))
			if err != nil {
				log.Error("error finding replacement command", slog.Any("error", err))
				return err
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

		// same as above, but no warning message (TODO - can't recall why, indirection?)
		if r, ok := command.Annotations[CmdReplacedBy]; ok {
			var newargs []string
			realcmd, newargs, err = command.Root().Find(append(strings.Split(r, " "), args...))
			if err != nil {
				log.Error("error finding replacement command", slog.Any("error", err))
				return err
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
	cordial.RenderHelpAsMD(Cmd)

	err := Cmd.Execute()
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

// initGlobalConfig reads in global config file and and ENV variables if
// set.
func initGlobalConfig() (cf *config.Config, err error) {
	if debug {
		logger.LogLevel.Set(slog.LevelDebug)
	}

	log.Debug("cordial 'geneos' running", slog.String("executable", cordial.ExecutableName()), slog.String("version", cordial.VERSION))

	// `oldConfDir` is the original path to the user configuration,
	// typically directly in `~/geneos`. The LoadConfig() function
	// already looks in standardised user and global directories. If
	// user lookup fails we get an empty path, which is fine - so ignore
	// errors.
	oldConfDir, _ := config.UserConfigPath()

	cf, err = config.Read(cordial.ExecutableName(),
		config.FilePath(cfgFile),
		config.UseGlobal(),
		config.SearchDirs(oldConfDir),
		config.MergeSources(),
		config.SkipWorkingDir(),
		config.WithEnvs("ITRS", "_"),
		config.UseDefaults(false),
	)
	if err != nil {
		log.Error("error reading configuration", slog.Any("error", err))
	}

	configPath = config.Path(cordial.ExecutableName(),
		config.FilePath(cfgFile),
		config.UseGlobal(),
		config.SearchDirs(oldConfDir),
		config.MergeSources(),
		config.SkipWorkingDir(),
		config.WithEnvs("ITRS", "_"),
		config.UseDefaults(false),
		config.MustExist(),
	)

	if configPath != "" {
		log.Debug("configuration loaded", slog.String("path", configPath))
	} else {
		log.Debug("no configuration file found, using defaults and environment variables")
	}

	// support old set-ups
	cf.BindEnv(cordial.ExecutableName(), "GENEOS_HOME", "ITRS_HOME")

	// manual alias+remove as the config.RegisterAlias doesn't work as expected
	if ih, ok := config.Lookup[string](cf, "itrshome"); ok {
		if _, ok := config.Lookup[string](cf, cordial.ExecutableName()); !ok {
			config.Set(cf, cordial.ExecutableName(), ih)
		}
		config.Delete(cf, "itrshome")
	}

	// initialise after config loaded
	geneos.InitHosts(cordial.ExecutableName())
	log.Debug("hosts loaded")

	return
}
