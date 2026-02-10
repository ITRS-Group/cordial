/*
Copyright © 2026 ITRS Group

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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/integrations/sdp/internal/sdp"
	"github.com/itrs-group/cordial/pkg/config"
)

func init() {
	RootCmd.AddCommand(listCmd)

	listCmd.Flags().SortFlags = false
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List items from ServiceDesk Plus",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// var response sdp.RequestGetListResponse

		cf := LoadConfigFile()

		client, err := sdp.NewClient(cmd.Context(), cf, "SDPOnDemand.requests.ALL")
		if err != nil {
			return
		}

		response, err := client.GetRequests(cmd.Context(), cf.Get("requests.search"))
		log.Debug().Msgf("response: %+v", response)
		if response.ResponseStatus[0].StatusCode != 2000 {
			log.Error().Msgf("API error: %s", response.ResponseStatus[0].Status)
			return
		}

		columnNames := cf.GetStringSlice("requests.columns")
		fmt.Println(strings.Join(columnNames, ","))

		for _, req := range response.Requests {
			var err error
			b, err := json.MarshalIndent(req, "", "  ")
			if err != nil {
				log.Debug().Err(err).Msgf("failed to marshal request")
				continue
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))

			r, err := config.Load("request",
				config.SetAppName("sdp"),
				config.SetFileExtension("json"),
				config.SetConfigReader(bytes.NewReader(b)),
			)
			if err != nil {
				continue
			}

			columns := make([]string, 0, len(columnNames))

			for _, c := range columnNames {
				columns = append(columns, r.GetString(c))
			}
			fmt.Println(strings.Join(columns, ","))

			// log.Debug().Msgf("request: %+v", request.AllSettings())
		}

		log.Info().Msgf("successfully listed requests from ServiceDesk Plus")
		return
	},
}
