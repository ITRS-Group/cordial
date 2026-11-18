/*
Copyright © 2022 ITRS Group

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
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the SSO diagnostic server",
	Long: `Run the diagnostic server using the settings in the SSO Agent configuration
file conf/sso-agent.conf, relative to the directory given by --dir.

The server logs in to Kerberos using the configured account and then serves:

  /status      a simple status page
  /public_key  the public key from the SSO Agent key store (if enabled in the configuration)
  /testuser    authenticates the caller with SPNEGO/Kerberos and shows the LDAP entry for them
  /authorize   a placeholder for the SSO Agent token endpoint

LDAP connections do not verify the server certificate unless --ldap-verify is given.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return start()
	},
}
