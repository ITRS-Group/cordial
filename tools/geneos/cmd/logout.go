/*
Copyright © 2023 ITRS Group

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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
)

var logoutCmdAll bool

func init() {
	GeneosCmd.AddCommand(logoutCmd)

	logoutCmd.Flags().BoolVarP(&logoutCmdAll, "all", "A", false, "remove all credentials")
}

//go:embed _docs/logout.md
var logoutCmdDescription string

var logoutCmd = &cobra.Command{
	Use:          "logout [flags] [DOMAIN...]",
	GroupID:      CommandGroupCredentials,
	Short:        "Remove Credentials",
	Long:         logoutCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:      "false",
		CmdRequireHome: "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if logoutCmdAll {
			config.DeleteAllCreds(config.SetAppName(Execname))
			return
		}
		if len(args) != 0 {
			for _, d := range args {
				config.DeleteCreds(d, config.SetAppName(Execname))
			}
		}
	},
}
