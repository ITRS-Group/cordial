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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/integrations/sdp/internal/sdp"
)

var authCmdCode = &config.Plaintext{}

func init() {
	RootCmd.AddCommand(authCmd)

	authCmd.Flags().VarP(authCmdCode, "code", "c", "Authorization code received from ServiceDesk Plus, prompted if not provided")

	authCmd.Flags().SortFlags = false
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate against ServiceDesk Plus and store the token persistently",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// load user config
		cf := LoadConfigFile()

		if authCmdCode == nil || authCmdCode.IsNil() {
			authCmdCode, err = config.ReadPasswordInput(false, 0, "Grant Code")
			if err != nil {
				return
			}
		}

		if _, err = sdp.InitialAuth(cf, authCmdCode); err != nil {
			log.Error().Msgf("failed to authenticate against ServiceDesk Plus: %v", err)
			return
		}

		log.Info().Msgf("successfully authenticated against ServiceDesk Plus, token saved")

		return
	},
}
