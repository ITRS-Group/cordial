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
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var cfgFile string
var execname = cordial.ExecutableName()
var debug, quiet bool
var d int

func init() {
	cobra.OnInitialize(initConfig)

	cordial.LogInit(execname)

	FILE2DVCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2email.yaml)")

	FILE2DVCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	FILE2DVCmd.PersistentFlags().MarkHidden("debug")

	// how to remove the help flag help text from the help output! Sigh...
	FILE2DVCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	FILE2DVCmd.PersistentFlags().MarkHidden("help")

	FILE2DVCmd.Flags().IntVarP(&d, "dataview", "D", 0, "Dataview index in configuration file, starting from zero")
}

// Column holds the options for each column in the output
type Column struct {
	Name  string         `mapstructure:"name"`
	Value string         `mapstructure:"value"`
	Match *regexp.Regexp `mapstructure:"match,omitempty"`
	Fail  string         `mapstructure:"fail,omitempty"`
}

// func (c *Column) UnmarshalJSON(d []byte) (err error) {
// 	return json.Unmarshal(d, c)
// }

type Headline struct {
	Name  string
	Value string
}

type Dataview struct {
	Name      string
	Headlines []Headline
	Table     [][]string
}

// FILE2DVCmd represents the base command when called without any subcommands
var FILE2DVCmd = &cobra.Command{
	Use:          "file2dv",
	Short:        "Extract and Transform from multiple files to CSV, for Geneos Toolkit input",
	Long:         ``,
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		d := config.Join("dataviews", strconv.Itoa(d))
		if !cf.IsSet(d) {
			return errors.New("no dataviews found in configuration")
		}

		dv := cf.Sub(d)

		dataview, err := processFiles(dv)

		if err != nil {
			return
		}
		for _, r := range dataview.Table {
			var row []string
			for _, c := range r {
				row = append(row, strings.ReplaceAll(c, ",", "\\,"))
			}
			fmt.Println(strings.Join(row, ","))
		}
		if dataview.Name != "" {
			fmt.Printf("<!>dataview,%s\n", dataview.Name)
		}
		for _, h := range dataview.Headlines {
			fmt.Printf("<!>%s,%s\n", h.Name, strings.ReplaceAll(h.Value, ",", "\\,"))
		}
		return
	},
}

var cf *config.Config

//go:embed defaults.yaml
var defaults []byte

func initConfig() {
	var err error
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
		config.WithDefaults(defaults, "yaml"),
	}

	cf, err = config.Load(execname, opts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("loading from %s", config.Path(execname, opts...))
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cordial.RenderHelpAsMD(FILE2DVCmd)

	err := FILE2DVCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
