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
	_ "embed"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

func init() {
	Cmd.AddCommand(reloadCmd)
}

//go:embed _docs/reload.md
var reloadCmdDescription string

var reloadCmd = &cobra.Command{
	Use:          "reload [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Reload Instance Configurations",
	Long:         reloadCmdDescription,
	Aliases:      []string{"refresh"},
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return err
		}
		targets := instance.ProfileAndMembers
		if ct != nil && ct.IsA("profile") {
			targets = instance.ProfileOnly
		}
		return instance.Do(cmd.CommandPath(), geneos.GetHost(Hostname), ct, names, ReloadInstance, targets).Report(os.Stdout)
	},
}

// ReloadInstance TODO: Document its behavior, inputs, and effects.
func ReloadInstance(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	if resp.Err = i.Reload(); resp.Err == nil {
		resp.Completed = append(resp.Completed, "reload signal sent")
	}
	return
}
