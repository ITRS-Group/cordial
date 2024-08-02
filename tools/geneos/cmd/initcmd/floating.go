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

package initcmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var floatingCmdArchive, floatingCmdVersion, floatingCmdOverride string

func init() {
	initCmd.AddCommand(floatingCmd)

	floatingCmd.Flags().StringVarP(&floatingCmdVersion, "version", "V", "latest", "Download this `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	floatingCmd.Flags().StringVarP(&floatingCmdArchive, "archive", "A", "", archiveOptionsText)
	floatingCmd.Flags().StringVarP(&floatingCmdOverride, "override", "O", "", "Override the `[TYPE:]VERSION` for archive files with non-standard names")

	floatingCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", instance.GatewaysOptionstext)
	floatingCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", instance.AttributesOptionsText)
	floatingCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", instance.TypesOptionsText)
	floatingCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", instance.VarsOptionsText)

	floatingCmd.Flags().SortFlags = false
}

//go:embed _docs/floating.md
var floatingCmdDescription string

var floatingCmd = &cobra.Command{
	Use:          "floating [flags] [USERNAME] [DIRECTORY]",
	Short:        "Initialise a Geneos Floating Netprobe environment",
	Long:         floatingCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdNoneMeansAll: "false",
		cmd.CmdRequireHome:  "false",
	},
	Deprecated: "Please use the `" + cordial.ExecutableName() + " deploy floating` command instead",
}
