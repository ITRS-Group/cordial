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

package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var queryTable, queryUser, queryFormat string
var queryColumns []string

func init() {
	RootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringVarP(&queryTable, "table", "t", "", "servicenow table, defaults to incident")
	queryCmd.Flags().StringVarP(&queryUser, "user", "u", "", "incident user to query")
	queryCmd.Flags().StringVarP(&queryFormat, "format", "f", "json", "output format: csv or `json`")
	queryCmd.Flags().StringSliceVarP(&queryColumns, "columns", "C", nil, "order of columns for CSV format")
	queryCmd.Flags().SortFlags = false
}

var queryCmd = &cobra.Command{
	Use:   "query [FLAGS]",
	Short: "Query ServiceNow incidents",
	Long: strings.ReplaceAll(`

`, "|", "`"),
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("defaults: %#v", cf.Get("defaults"))
		cf := loadConfigFile("client")
		query(cf, args)
	},
}

func query(cf *config.Config, args []string) {
	var result []map[string]string

	for _, ru := range cf.GetStringSlice(cf.Join("router", "url")) {
		rc := rest.NewClient(
			rest.BaseURL(ru),
			rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ string, _ []byte) {
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cf.GetString(config.Join("router", "authentication", "token"))))
			}),
		)

		if queryTable == "" {
			queryTable = cf.GetString(config.Join("router", "default-table"), config.Default("incident"))
		}
		if queryUser == "" {
			queryUser = cf.GetString(config.Join("router", "default-user"))
		}

		var err error

		_, err = rc.Get(context.Background(), queryTable, "user="+queryUser, &result)

		if err != nil {
			log.Debug().Err(err).Msg("connection error, trying next router (if any)")
			continue
		}

		break
	}

	if result == nil {
		log.Debug().Msg("no data returned")
	}

	if !strings.EqualFold(queryFormat, "csv") {
		b, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}
		fmt.Println(string(b))
		return
	}

	columns := queryColumns
	if len(columns) == 0 {
		columns = cf.GetStringSlice(cf.Join("query", "fields"))
	}

	csv := csv.NewWriter(os.Stdout)
	csv.Write(columns)

	for _, line := range result {
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
}
