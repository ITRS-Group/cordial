/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
