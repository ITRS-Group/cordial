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

package cmd

import (
	_ "embed"
	"fmt"
	dbg "runtime/debug"

	"github.com/itrs-group/cordial"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(versionCmd)
}

//go:embed _docs/version.md
var versionCmdDescription string

var versionCmd = &cobra.Command{
	Use:          "version",
	GroupID:      CommandGroupOther,
	Short:        "Show program version",
	Long:         versionCmdDescription,
	SilenceUsage: true,
	Version:      cordial.VERSION,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "false",
	},
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", Execname, cmd.Version)
		if debug {
			info, ok := dbg.ReadBuildInfo()
			if ok {
				fmt.Println("additional info:")
				fmt.Println(info)
			}
		}
	},
}
