/*
Copyright © 2024 ITRS Group

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
	"os"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed _docs/list.md
var listCmdDescription string

var listIncludeCmdDescription string
var listExcludeCmdDescription string

var listCmdFormat string

var listFormat string

func init() {
	GDNACmd.AddCommand(listCmd)
	listCmd.AddCommand(listReportCmd)
	listCmd.AddCommand(listExcludeCmd)
	listCmd.AddCommand(listIncludeCmd)

	listCmd.PersistentFlags().StringVarP(&listCmdFormat, "format", "F", "table", "format output. supported formats: 'html', 'table', 'tsv', 'toolkit', 'markdown'")

	listReportCmd.Flags().StringVarP(&reportNames, "report", "r", "", reportNamesDescription)
	listReportCmd.Flags().SortFlags = false

	listExcludeCmd.Flags().StringVarP(&listFormat, "format", "F", "", "output format")
	listIncludeCmd.Flags().StringVarP(&listFormat, "format", "F", "", "output format")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List commands",
	Long:  listCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
}

var listReportCmd = &cobra.Command{
	Use:     "reports",
	Short:   "List available reports",
	Long:    listCmdDescription,
	Aliases: []string{"report"},
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		r, _ := reporter.NewReporter("table", os.Stdout, reporter.RenderAs(listCmdFormat))

		return listReports(cf, r)
	},
}

var listIncludeCmd = &cobra.Command{
	Use:     "includes [CATEGORY]",
	Short:   "List excluded items",
	Long:    listIncludeCmdDescription,
	Aliases: []string{"include"},
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var category string
		if len(args) > 0 {
			category = args[0]
		}
		return listFilters("include", category, listFormat)
	},
}

var listExcludeCmd = &cobra.Command{
	Use:     "excludes [CATEGORY]",
	Short:   "List excluded items",
	Long:    listExcludeCmdDescription,
	Aliases: []string{"exclude"},
	Args:    cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"defaultlog": os.DevNull,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var category string
		if len(args) > 0 {
			category = args[0]
		}
		return listFilters("exclude", category, listFormat)
	},
}

func listReports(cf *config.Config, r reporter.Reporter) (err error) {
	var reports []string
	for name := range cf.GetStringMap("reports") {
		reports = append(reports, name)
	}
	slices.Sort(reports)

	var rows [][]string

	for _, name := range reports {
		var rep Report

		if reportNames != "" {
			if match, _ := matchReport(name, reportNames); !match {
				continue
			}
		}

		if err = cf.UnmarshalKey(config.Join("reports", name), &rep); err != nil {
			log.Error().Err(err).Msg("reports configuration format incorrect")
			return
		}

		enabledDataview := "Y"
		if rep.Dataview.Enable != nil && !*rep.Dataview.Enable {
			enabledDataview = "N"
		}
		enabledXLSX := "Y"
		if rep.XLSX.Enable != nil && !*rep.XLSX.Enable {
			enabledXLSX = "N"
		}
		rows = append(rows, []string{
			name,
			rep.Dataview.Group,
			rep.Title,
			rep.Type,
			enabledDataview,
			enabledXLSX,
		})
	}

	slices.SortFunc(rows, func(a, b []string) int {
		if a[1] == b[1] {
			return strings.Compare(a[2], b[2])
		}
		return strings.Compare(a[1], b[1])
	})
	r.UpdateTable([]string{"Report Name", "Group", "Title", "Type", "Dataview", "XLSX"}, rows)
	r.Render()

	return
}
