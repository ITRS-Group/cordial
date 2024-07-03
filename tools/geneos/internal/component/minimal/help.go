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

package minimal

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

// Help command and text to hook into Cobra command tree

//go:embed README.md
var minimalDescription string

func init() {
	cmd.GeneosCmd.AddCommand(helpDocCmd)
}

var helpDocCmd = &cobra.Command{
	Use:                   "`minimal",
	GroupID:               cmd.CommandGroupComponents,
	Short:                 "Fix Analyser 2",
	Long:                  minimalDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Run:                   cmd.GeneosCmd.HelpFunc(),
}