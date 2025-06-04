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
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd"
	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/router"
)

var queryCmdTable, queryCmdUser, queryCmdFormat string

func init() {
	cmd.RootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryCmdTable, "table", "t", "", "servicenow table, defaults to incident")
	queryCmd.Flags().StringVarP(&queryCmdUser, "user", "u", "", "incident user to query")
	queryCmd.Flags().StringVarP(&queryCmdFormat, "format", "f", "json", "output format: csv or `json`")
	queryCmd.Flags().SortFlags = false
}

var queryCmd = &cobra.Command{
	Use:   "query [FLAGS]",
	Short: "Query ServiceNow incidents",
	Long: strings.ReplaceAll(`

`, "|", "`"),
	SilenceUsage: true,
	Run: func(command *cobra.Command, args []string) {
		// fmt.Printf("defaults: %#v", cf.Get("defaults"))
		cf := cmd.LoadConfigFile("client")

		// var result []map[string]string
		var err error
		var result router.ResultsResponse

		for _, ru := range cf.GetStringSlice(cf.Join("router", "url")) {
			rc := rest.NewClient(
				rest.BaseURLString(ru),
				rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ []byte) {
					req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cf.GetString(config.Join("router", "authentication", "token"))))
				}),
			)

			if queryCmdTable == "" {
				queryCmdTable = cf.GetString(config.Join("router", "default-table"), config.Default(router.SNOW_INCIDENT_TABLE))
			}
			if queryCmdUser == "" {
				queryCmdUser = cf.GetString(config.Join("router", "default-user"))
			}

			_, err = rc.Get(context.Background(), queryCmdTable, "user="+queryCmdUser, &result)

			if err == nil {
				break
			}

			log.Debug().Err(err).Msg("connection error, trying next router (if any)")
		}

		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		if len(result.Results) == 0 {
			log.Fatal().Msg("no data returned")
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
	},
}
