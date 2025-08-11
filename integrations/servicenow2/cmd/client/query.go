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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
)

var queryCmdTable, queryCmdQuery, queryCmdFormat string

func init() {
	cmd.RootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdTable, "table", "t", "", "servicenow table, defaults to incident")
	queryCmd.Flags().StringVarP(&queryCmdQuery, "query", "q", "", "query")
	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "csv", "output format: `csv` or json")
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
			}{
				Query: queryCmdQuery,
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

		csv := csv.NewWriter(os.Stdout)
		csv.Write(columns)

		for _, line := range result.Results {
			var fields []string
			for _, col := range columns {
				f := line[col]
				if !strconv.CanBackquote(f) {
					f = strings.Trim(strconv.QuoteToASCII(f), `"`)
				}
				fields = append(fields, f)
			}
			csv.Write(fields)
		}
		csv.Flush()

		// write headlines for toolkit consumers
		fmt.Printf("<!>table,%s\n", queryCmdTable)
		fmt.Printf("<!>results,%d\n", len(result.Results))
		fmt.Printf("<!>query,%s\n", queryCmdQuery)
	},
}
