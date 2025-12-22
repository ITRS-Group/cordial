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

package tlscmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var migrateCmdDays int
var migrateCmdNewKey, migrateCmdPrepare, migrateCmdRoll, migrateCmdUnroll bool

func init() {
	tlsCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().BoolVarP(&migrateCmdPrepare, "prepare", "P", false, "Prepare migration without changing existing files")
	migrateCmd.Flags().BoolVarP(&migrateCmdRoll, "roll", "R", false, "Roll previously prepared migrated files and backup existing ones")
	migrateCmd.Flags().BoolVarP(&migrateCmdUnroll, "unroll", "U", false, "Unroll previously rolled migrated files to earlier backups")

	migrateCmd.MarkFlagsMutuallyExclusive("prepare", "roll", "unroll")
}

//go:embed _docs/migrate.md
var migrateCmdDescription string

var migrateCmd = &cobra.Command{
	Use:          "migrate [TYPE] [NAME...]",
	Short:        "Migrate certificates and related files to the updated layout",
	Long:         migrateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, migrateInstanceCert).Write(os.Stdout)
	},
}

func migrateInstanceCert(i geneos.Instance, _ ...any) (resp *instance.Response) {
	return
}

func migrateNonInstanceCert(i geneos.Instance, _ ...any) (resp *instance.Response) {
	return
}
