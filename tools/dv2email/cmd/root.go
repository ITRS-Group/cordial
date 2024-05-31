/*
Copyright © 2023 ITRS Group

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
	"os"
	"path"
	"sort"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, quiet bool
var inlineCSS bool

var entityArg, samplerArg, typeArg, dataviewArg string
var toArg, ccArg, bccArg string

func init() {
	// cobra.OnInitialize(initConfig)

	DV2EMAILCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2email.yaml)")

	DV2EMAILCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	DV2EMAILCmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	DV2EMAILCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	DV2EMAILCmd.PersistentFlags().MarkHidden("help")

	DV2EMAILCmd.PersistentFlags().StringVarP(&entityArg, "entity", "E", "", "entity name, ignored if _VARIBLEPATH set in environment")
	DV2EMAILCmd.PersistentFlags().StringVarP(&samplerArg, "sampler", "S", "", "sampler name, ignored if _VARIBLEPATH set in environment")
	DV2EMAILCmd.PersistentFlags().StringVarP(&typeArg, "type", "T", "", "type name, ignored if _VARIBLEPATH set in environment\nTo explicitly select empty/no type use --type/-T \"\"")
	DV2EMAILCmd.PersistentFlags().StringVarP(&dataviewArg, "dataview", "D", "", "dataview name, ignored if _VARIBLEPATH set in environment")

	DV2EMAILCmd.Flags().BoolVarP(&inlineCSS, "inline-css", "i", true, "inline CSS for better mail client support")
	DV2EMAILCmd.Flags().StringVarP(&toArg, "to", "t", "", "To as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&ccArg, "cc", "c", "", "Cc as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&bccArg, "bcc", "b", "", "Bcc as comma-separated emails")

	DV2EMAILCmd.Flags().SortFlags = false
}

// global config
var cf *config.Config

func initConfig() {
	var err error

	cordial.LogInit(execname)

	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// config.DefaultKeyDelimiter("::")

	opts := []config.FileOptions{
		config.SetAppName("geneos"),
		config.SetConfigFile(cfgFile),
		config.MergeSettings(),
		config.SetFileExtension("yaml"),
		config.WithDefaults(defaults, "yaml"),
	}

	cf, err = config.Load(execname, opts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("loading from file %s", config.Path(execname, opts...))
	}
}

type DV2EMailData struct {
	// Dataviews is a slice of each Dataview's data, including Columns
	// and Rows which are ordered names for the columns and rows
	// respectively, suitable for range loops. See
	// https://pkg.go.dev/github.com/itrs-group/cordial/pkg/commands#Dataview
	// for details
	Dataviews []*commands.Dataview

	// Env is a map of environment variable, names to values
	Env map[string]string
}

//go:embed dv2email.defaults.yaml
var defaults []byte

//go:embed _docs/root.md
var DV2EMAILCmdDescription string

// DV2EMAILCmd represents the base command when called without any subcommands
var DV2EMAILCmd = &cobra.Command{
	Use:          "dv2email",
	Short:        "Email a Dataview following Geneos Action/Effect conventions",
	Long:         DV2EMAILCmdDescription,
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initConfig()
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		gw, err := dialGateway(cf)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		// we need to pass filters etc. to fetchDataviews
		em := email.NewEmailConfig(cf, toArg, ccArg, bccArg)

		data, err := fetchDataviews(cmd, gw,
			em.GetString("_firstcolumn"),
			em.GetString("__headlines"),
			em.GetString("__rows"),
			em.GetString("__columns"),
			em.GetString("__roworder"),
		)
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}

		switch cf.GetString("email.split") {
		case "entity":
			entities := map[string][]*commands.Dataview{}
			for _, d := range data.Dataviews {
				if len(entities[d.XPath.Entity.Name]) == 0 {
					entities[d.XPath.Entity.Name] = []*commands.Dataview{}
				}
				entities[d.XPath.Entity.Name] = append(entities[d.XPath.Entity.Name], d)
			}
			for _, e := range entities {
				many := DV2EMailData{
					Dataviews: e,
					Env:       data.Env,
				}
				if err = sendEmail(cf, em, many, inlineCSS); err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		case "dataview":
			for _, d := range data.Dataviews {
				one := DV2EMailData{
					Dataviews: []*commands.Dataview{d},
					Env:       data.Env,
				}
				if err = sendEmail(cf, em, one, inlineCSS); err != nil {
					log.Fatal().Err(err).Msg("")
				}
			}
		default:
			if err = sendEmail(cf, em, data, inlineCSS); err != nil {
				log.Fatal().Err(err).Msg("sending failed")
			}
		}

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	cordial.RenderHelpAsMD(DV2EMAILCmd)

	err := DV2EMAILCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// match will return either a fixed list of matches from a comma
// separated override or, if override is empty, it will search for a
// section in the global configuration instance for `confkey` and return
// the values for the longest matching key (using globbing rules)
//
// match returns a the first slice of values of the member of
// confkey that matches name using globbing rules, longest match wins.
// e.g. for
//
// confkey:
//
//	col*: this, that, other
//	columnFullName: only, these
//
// and if name is 'columnFullName' then [ 'only', 'these' ] is returned.
func match(name, confkey, override string) (matches []string) {
	if override != "" {
		matches = strings.Split(override, ",")
		return
	}

	name = strings.ToLower(name)
	checks := cf.GetStringMapStringSlice(confkey)
	if len(checks) == 0 {
		return
	}
	keys := []string{}
	for k := range checks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	for _, m := range keys {
		if ok, _ := path.Match(m, name); ok {
			matches = checks[m]
			break
		}
	}
	return
}
