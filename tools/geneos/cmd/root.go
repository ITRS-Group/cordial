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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
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

type ExtraConfigValues struct {
	Includes   IncludeValues
	Gateways   GatewayValues
	Attributes AttributeValues
	Envs       EnvValues
	Variables  VarValues
	Types      TypeValues
	// Keys       StringSliceValues
}

func init() {
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "geneos",
	Short: "Control your Geneos environment",
	Long: strings.ReplaceAll(`
Manage and control your Geneos environment. With |geneos| you can
initialise a new installation, add and remove components, control
processes and build template based configuration files for SANs and
// new gateways.?
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
	SilenceErrors:      true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		// check initialisation
		geneosdir := host.Geneos()
		if geneosdir == "" {
			// commands that do not require geneos home to be set
			if !(cmd == initCmd ||
				cmd.Parent() == initCmd ||
				cmd == setUserCmd ||
				cmd == setGlobalCmd ||
				cmd == addHostCmd ||
				len(host.RemoteHosts()) > 0) {
				// if cmd != rootCmd && cmd != initCmd && cmd.Parent() != initCmd && cmd != setUserCmd && cmd != setGlobalCmd && cmd != addHostCmd && len(host.RemoteHosts()) == 0 {
				cmd.SetUsageTemplate(" ")
				return fmt.Errorf("%s", `Geneos installation directory not set.

Use one of the following to fix this:

For an existing installation:
	$ geneos set user geneos=/path/to/geneos

To initialise a new installation:
	$ geneos init /path/to/geneos

For temporary usage:
	$ export ITRS_HOME=/path/to/geneos
`)
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
	cordial.LogInit(pkgname)

	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	oldConfDir, _ := os.UserConfigDir()

	cf, err := config.LoadConfig("geneos",
		config.SetConfigFile(cfgFile),
		config.UseGlobal(),
		config.AddConfigDirs(oldConfDir))
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

	u, _ := user.Current()
	cf.SetDefault("defaultuser", u.Username)

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

// Value types for multiple flags

// XXX abstract this for a general case
func setExtendedValues(c geneos.Instance, x ExtraConfigValues) (changed bool) {
	if setSlice(c, x.Attributes, "attributes", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	}) {
		changed = true
	}

	if setSlice(c, x.Envs, "env", func(a string) string {
		return strings.SplitN(a, "=", 2)[0]
	}) {
		changed = true
	}

	if setSlice(c, x.Types, "types", func(a string) string {
		return a
	}) {
		changed = true
	}

	if len(x.Gateways) > 0 {
		gateways := c.Config().GetStringMapString("gateways")
		for k, v := range x.Gateways {
			gateways[k] = v
		}
		c.Config().Set("gateways", gateways)
	}

	if len(x.Includes) > 0 {
		incs := c.Config().GetStringMapString("includes")
		for k, v := range x.Includes {
			incs[k] = v
		}
		c.Config().Set("includes", incs)
	}

	if len(x.Variables) > 0 {
		vars := c.Config().GetStringMapString("variables")
		for k, v := range x.Variables {
			vars[k] = v
		}
		c.Config().Set("variables", vars)
	}

	return
}

// sets 'items' in the settings identified by 'key'. the key() function returns an identifier to use
// in merge comparisons
func setSlice(c geneos.Instance, items []string, setting string, key func(string) string) (changed bool) {
	if len(items) == 0 {
		return
	}

	newvals := []string{}
	vals := c.Config().GetStringSlice(setting)

	if len(vals) == 0 {
		c.Config().Set(setting, items)
		changed = true
		return
	}

	// map to store the identifier and the full value for later checks
	keys := map[string]string{}
	for _, v := range items {
		keys[key(v)] = v
		newvals = append(newvals, v)
	}

	for _, v := range vals {
		if w, ok := keys[key(v)]; ok {
			// exists
			if v != w {
				// only changed if different value
				changed = true
				continue
			}
		} else {
			// copying the old value is not a change
			newvals = append(newvals, v)
		}
	}

	// check old values against map, copy those that do not exist

	c.Config().Set(setting, newvals)
	return
}

// include file - priority:url|path
type IncludeValues map[string]string

func (i *IncludeValues) String() string {
	return ""
}

func (i *IncludeValues) Set(value string) error {
	e := strings.SplitN(value, ":", 2)
	val := "100"
	if len(e) > 1 {
		val = e[1]
	} else {
		// XXX check two values and first is a number
		log.Debug().Msgf("second value missing after ':', using default %s", val)
	}
	(*i)[e[0]] = val
	return nil
}

func (i *IncludeValues) Type() string {
	return "PRIORITY:{URL|PATH}"
}

// gateway - name:port
type GatewayValues map[string]string

func (i *GatewayValues) String() string {
	return ""
}

func (i *GatewayValues) Set(value string) error {
	e := strings.SplitN(value, ":", 2)
	val := "7039"
	if len(e) > 1 {
		val = e[1]
	} else {
		// XXX check two values and first is a number
		log.Debug().Msgf("second value missing after ':', using default %s", val)
	}
	(*i)[e[0]] = val
	return nil
}

func (i *GatewayValues) Type() string {
	return "HOSTNAME:PORT"
}

// attribute - name=value
type AttributeValues []string

func (i *AttributeValues) String() string {
	return ""
}

func (i *AttributeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *AttributeValues) Type() string {
	return "NAME=VALUE"
}

// attribute - name=value
type TypeValues []string

func (i *TypeValues) String() string {
	return ""
}

func (i *TypeValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *TypeValues) Type() string {
	return "NAME"
}

// env NAME=VALUE - string slice
type EnvValues []string

func (i *EnvValues) String() string {
	return ""
}

func (i *EnvValues) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *EnvValues) Type() string {
	return "NAME=VALUE"
}

// variables - [TYPE:]NAME=VALUE
type VarValues map[string]string

func (i *VarValues) String() string {
	return ""
}

func (i *VarValues) Set(value string) error {
	var t, k, v string

	e := strings.SplitN(value, ":", 2)
	if len(e) == 1 {
		t = "string"
		s := strings.SplitN(e[0], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
	} else {
		t = e[0]
		s := strings.SplitN(e[1], "=", 2)
		k = s[0]
		if len(s) > 1 {
			v = s[1]
		}
	}

	// XXX check types here - e[0] options type, default string
	var validtypes map[string]string = map[string]string{
		"string":             "",
		"integer":            "",
		"double":             "",
		"boolean":            "",
		"activeTime":         "",
		"externalConfigFile": "",
	}
	if _, ok := validtypes[t]; !ok {
		log.Error().Msgf("invalid type %q for variable", t)
		return geneos.ErrInvalidArgs
	}
	val := t + ":" + v
	(*i)[k] = val
	return nil
}

func (i *VarValues) Type() string {
	return "[TYPE:]NAME=VALUE"
}
