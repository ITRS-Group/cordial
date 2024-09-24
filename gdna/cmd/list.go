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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

//go:embed _docs/list.md
var listCmdDescription string

var listCmdFormat string

func init() {
	GDNACmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(&listCmdFormat, "format", "F", "table", "format output. supported formats: 'html', 'table', 'tsv', 'toolkit', 'markdown'")
	listCmd.Flags().StringVarP(&reportNames, "report", "r", "", "report names")

	listCmd.Flags().SortFlags = false
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available reports",
	Long:  listCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Annotations: map[string]string{
		"nolog": "true",
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		r := reporter.NewFormattedReporter(os.Stdout, reporter.RenderAs(listCmdFormat))

		return listReports(cf, r)
	},
}

func listReports(cf *config.Config, r reporter.Reporter) (err error) {
	var reports []string
	for name := range cf.GetStringMap("reports") {
		reports = append(reports, name)
	}
	slices.Sort(reports)

	rows := [][]string{
		{"Report Name", "Title", "Type", "Dataview", "XLSX"},
	}

	for _, name := range reports {
		var rep Report

		if reportNames != "" {
			if !matchReport(name, reportNames) {
				continue
			}
		}

		if err = cf.UnmarshalKey(config.Join("reports", name), &rep); err != nil {
			log.Error().Err(err).Msg("reports configuration format incorrect")
			return
		}

		enabledDataview := "Y"
		if rep.EnableForDataview != nil && !*rep.EnableForDataview {
			enabledDataview = "N"
		}
		enabledXLSX := "Y"
		if rep.EnableForXLSX != nil && !*rep.EnableForXLSX {
			enabledXLSX = "N"
		}
		rows = append(rows, []string{
			name,
			rep.Name,
			rep.Type,
			enabledDataview,
			enabledXLSX,
		})
	}

	r.UpdateTable(rows...)
	r.Flush()

	return
}
