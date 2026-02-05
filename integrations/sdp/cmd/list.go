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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/integrations/sdp/internal/sdp"

	"github.com/itrs-group/cordial/pkg/rest"
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
		var response sdp.RequestGetListResponse

		cf := LoadConfigFile()

		client, err := sdp.Client(cmd.Context(), cf)
		if err != nil {
			return
		}

		rc := rest.NewClient(
			rest.HTTPClient(client),
			rest.BaseURLString(cf.GetString(cf.Join("datacentres", cf.GetString("datacentre"), "api"))),
			rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
				req.Header.Set("Accept", "application/vnd.manageengine.sdp.v3+json")
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}),
		)

		v := url.Values{}
		v.Add("input_data", `{"list_info":{"start_index":"1"}}`)

		endpoint, err := url.JoinPath("app", cf.GetString("portal"), "/api/v3/requests")
		if err != nil {
			return
		}
		resp, err := rc.Get(context.Background(), endpoint, v.Encode(), &response)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		b, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))

		log.Info().Msgf("successfully listed requests from ServiceDesk Plus")
		return
	},
}
