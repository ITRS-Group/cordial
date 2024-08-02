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
)

func init() {
	GeneosCmd.AddCommand(reloadCmd)
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
		CmdNoneMeansAll: "true",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, ReloadInstance)
		responses.Write(os.Stdout)
	},
}

func ReloadInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if err := i.Reload(); err == nil {
		resp.Completed = append(resp.Completed, "reload signal sent")
	}
	return
}
