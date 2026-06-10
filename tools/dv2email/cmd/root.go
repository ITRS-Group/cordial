/*
Copyright © 2023 ITRS Group

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

package cmd

import (
	_ "embed"
	"log/slog"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
	"github.com/itrs-group/cordial/pkg/geneos/commands"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, quiet bool
var inlineCSS bool

var entityArg, samplerArg, typeArg, dataviewArg string
var toArg, ccArg, bccArg, subjectArg string

var log = cordial.Logger

func init() {
	// cobra.OnInitialize(initConfig)

	Cmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2email.yaml)")

	Cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	Cmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	Cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	Cmd.PersistentFlags().MarkHidden("help")

	Cmd.PersistentFlags().StringVarP(&entityArg, "entity", "E", "", "entity name, ignored if _VARIBLEPATH set in environment")
	Cmd.PersistentFlags().StringVarP(&samplerArg, "sampler", "S", "", "sampler name, ignored if _VARIBLEPATH set in environment")
	Cmd.PersistentFlags().StringVarP(&typeArg, "type", "T", "", "type name, ignored if _VARIBLEPATH set in environment\nTo explicitly select empty/no type use --type/-T \"\"")
	Cmd.PersistentFlags().StringVarP(&dataviewArg, "dataview", "D", "", "dataview name, ignored if _VARIBLEPATH set in environment")

	Cmd.Flags().BoolVarP(&inlineCSS, "inline-css", "i", true, "inline CSS for better mail client support")

	Cmd.Flags().StringVarP(&toArg, "to", "t", "", "`\"TO, ...\"` recipients as a comma-separated list of email addresses")
	Cmd.Flags().StringVarP(&ccArg, "cc", "c", "", "`\"CC, ...\"` recipients as a comma-separated list of email addresses")
	Cmd.Flags().StringVarP(&bccArg, "bcc", "b", "", "`\"BCC, ...\"` recipients as a comma-separated list of email addresses")
	Cmd.Flags().StringVarP(&subjectArg, "subject", "s", "", "`\"SUBJECT\"` of the email")

	Cmd.Flags().SortFlags = false
}

// global config
var globalCf *config.Config

func initConfig() {
	var err error

	log = cordial.LogInit(execname)

	if quiet {
		cordial.LogLevel.Set(slog.LevelError)
	} else if debug {
		cordial.LogLevel.Set(slog.LevelDebug)
	} else {
		cordial.LogLevel.Set(slog.LevelInfo)
	}

	// config.DefaultKeyDelimiter("::")
	if config.AppConfigDir() == "" {
		log.Warn("no user config dir found", slog.Any("error", config.ErrNoUserConfigDir))
	}
	opts := []config.FileOption{
		config.AppName("geneos"),
		config.FilePath(cfgFile),
		config.MergeSources(),
		config.Format("yaml"),
		config.WithDefaults(defaults, "yaml"),
		config.WithEnvs("DV2EMAIL", "_"),
	}

	globalCf, err = config.Read(execname, opts...)
	if err != nil {
		log.Error("loading config failed", slog.Any("error", err), slog.String("path", config.Path(execname, opts...)))
		os.Exit(1)
	}
}

type DV2EMailData struct {
	// Dataviews is a slice of each Dataview's data, including Columns
	// and Rows which are ordered names for the columns and rows
	// respectively, suitable for range loops. See
	// https://pkg.go.dev/github.com/itrs-group/cordial/pkg/geneos/commands#Dataview
	// for details
	Dataviews []*commands.Dataview

	// Env is a map of environment variable, names to values
	Env map[string]string
}

//go:embed dv2email.defaults.yaml
var defaults []byte

//go:embed _docs/root.md
var DV2EMAILCmdDescription string

// Cmd represents the base command when called without any subcommands
var Cmd = &cobra.Command{
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
		gw, err := dialGateway(globalCf)
		if err != nil {
			log.Error("failed to dial gateway", slog.Any("error", err))
			os.Exit(1)
		}

		// we need to pass filters etc. to fetchDataviews
		em := email.NewEmailConfig(globalCf, toArg, ccArg, bccArg, subjectArg)

		data, err := fetchDataviews(cmd, gw,
			config.Get[string](em, "_firstcolumn"),
			config.Get[string](em, "__headlines"),
			config.Get[string](em, "__rows"),
			config.Get[string](em, "__columns"),
			config.Get[string](em, "__roworder"),
		)
		if err != nil {
			log.Error("failed to fetch dataviews", slog.Any("error", err))
			return
		}

		switch config.Get[string](globalCf, globalCf.Join("email", "split")) {
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
				if err = sendEmail(globalCf, em, many, inlineCSS); err != nil {
					log.Error("failed to send email", slog.Any("error", err))
					os.Exit(1)
				}
			}
		case "dataview":
			for _, d := range data.Dataviews {
				one := DV2EMailData{
					Dataviews: []*commands.Dataview{d},
					Env:       data.Env,
				}
				if err = sendEmail(globalCf, em, one, inlineCSS); err != nil {
					log.Error("failed to send email", slog.Any("error", err))
					os.Exit(1)
				}
			}
		default:
			if err = sendEmail(globalCf, em, data, inlineCSS); err != nil {
				log.Error("failed to send email", slog.Any("error", err))
				os.Exit(1)
			}
		}

		return
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
	checks := config.Get[map[string][]string](globalCf, confkey)
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
