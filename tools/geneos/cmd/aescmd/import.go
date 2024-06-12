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

package aescmd

import (
	_ "embed"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

var importCmdKeyfileSource string

func init() {
	aesCmd.AddCommand(importCmd)

	importCmdKeyfileSource = string(cmd.DefaultUserKeyfile)

	importCmd.Flags().StringVarP(&importCmdKeyfileSource, "keyfile", "k", "", "`PATH` to key file. Use a dash (`-`) to be prompted for the contents of the key file on the console")

	importCmd.Flags().SortFlags = false

}

//go:embed _docs/import.md
var importCmdDescription string

var importCmd = &cobra.Command{
	Use:          "import [flags] [TYPE] [NAME...]",
	Short:        "Import key files for component TYPE",
	Long:         importCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
	},
	Deprecated: "Please use the `" + cordial.ExecutableName() + " aes set` command instead",
	RunE: func(command *cobra.Command, _ []string) (err error) {
		return
	},
}
