/*
Copyright © 2023 ITRS Group

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
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		if logoutCmdAll {
			config.DeleteAllCreds(config.SetAppName(Execname))
			return
		}
		if len(args) == 0 {
			for _, d := range args {
				config.DeleteCreds(d, config.SetAppName(Execname))
			}
		}
	},
}
