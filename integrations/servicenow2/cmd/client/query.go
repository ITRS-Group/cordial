/*
Copyright © 2025 ITRS Group

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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
)

var queryCmdTable, queryCmdQuery, queryCmdFormat string
var queryCmdRaw bool

func init() {
	cmd.RootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdTable, "table", "t", "", "servicenow table, defaults to incident")
	queryCmd.Flags().StringVarP(&queryCmdQuery, "query", "q", "", "query")
	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "csv", "output format: `csv` or json")
	queryCmd.Flags().BoolVarP(&queryCmdRaw, "raw", "r", false, "turn ServiceNow sys_display off, i.e. return raw values instead of display values")
	queryCmd.Flags().SortFlags = false
}

var queryCmd = &cobra.Command{
	Use:   "query [FLAGS]",
	Short: "Query ServiceNow incidents",
	Long: strings.ReplaceAll(`

`, "|", "`"),
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		cf := cmd.LoadConfigFile("client")

		var err error
		var result proxy.ResultsResponse

		for _, r := range cf.GetStringSlice(cf.Join("proxy", "url")) {
			rc := newRestClient(cf, r)

			if queryCmdTable == "" {
				queryCmdTable = proxy.SNOW_INCIDENT_TABLE
			}

			if queryCmdQuery == "" {
				queryCmdQuery = cf.GetString(config.Join("proxy", "default-query"))
			}

			query := struct {
				Query string `url:"query,omitempty"`
				Raw   bool   `url:"raw,omitempty"`
			}{
				Query: queryCmdQuery,
				Raw:   queryCmdRaw,
			}

			if _, err = rc.Get(context.Background(), queryCmdTable, query, &result); err == nil {
				// all OK ?
				break
			}

			log.Debug().Err(err).Msg("connection error, trying next proxy (if any)")
		}

		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if !strings.EqualFold(queryCmdFormat, "csv") {
			b, err := json.MarshalIndent(result.Results, "", "    ")
			if err != nil {
				log.Error().Err(err).Msg("")
				return
			}
			fmt.Println(string(b))
			return
		}

		columns := result.Fields

		fmt.Println(strings.Join(columns, ","))

		for _, line := range result.Results {
			var fields []string
			for _, col := range columns {
				f := line[col]
				// escape commas for Toolkit input
				f = strings.ReplaceAll(f, ",", "\\,")
				fields = append(fields, f)
			}
			fmt.Println(strings.Join(fields, ","))
		}

		// write headlines for toolkit consumers
		fmt.Printf("<!>table,%s\n", queryCmdTable)
		fmt.Printf("<!>results,%d\n", len(result.Results))
		fmt.Printf("<!>query,%s\n", queryCmdQuery)
	},
}
