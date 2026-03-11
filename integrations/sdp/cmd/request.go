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
	"crypto/sha1"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/integrations/sdp/internal/sdp"
	"github.com/itrs-group/cordial/pkg/config"
)

func init() {
	RootCmd.AddCommand(requestCmd)

	requestCmd.Flags().SortFlags = false
}

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Raise or update a Request in ServiceDesk Plus",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		cf := LoadConfigFile()

		client, err := sdp.NewClient(cmd.Context(), cf, "SDPOnDemand.requests.ALL")
		if err != nil {
			return
		}

		correlationID := fmt.Sprintf("%X", sha1.Sum(cf.GetBytes("__itrs_correlation_id")))
		lookup := map[string]string{
			"correlation_id": correlationID,
		}

		log.Debug().Msgf("existing search: %s", cf.Get(cf.Join("requests", "existing_search")))

		if len(cf.GetBytes("__itrs_correlation_id")) == 0 {
			return fmt.Errorf("correlation_id is required")
		}

		response, err := client.GetRequests(cmd.Context(), cf.Get(cf.Join("requests", "existing_search")), config.LookupTable(lookup))
		if err != nil {
			return
		}
		if response.ResponseStatus[0].StatusCode != 2000 {
			return fmt.Errorf("API error: %s", response.ResponseStatus[0].Status)
		}
		log.Debug().Msgf("response: %+v", response)
		if response.ListInfo.RowCount == 0 {
			log.Info().Msgf("no existing request found, creating new request")
			createResponse, err := client.CreateRequest(cmd.Context(), cf.Sub("sdp"), cf.Join("requests", "create"), lookup)
			if err != nil {
				return err
			}
			log.Debug().Msgf("create response: %+v", createResponse)
		}

		return
	},
}
