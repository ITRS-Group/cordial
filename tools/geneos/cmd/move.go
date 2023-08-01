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
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(moveCmd)
}

//go:embed _docs/move.md
var moveCmdDescription string

var moveCmd = &cobra.Command{
	Use:          "move [TYPE] SOURCE DESTINATION",
	GroupID:      CommandGroupManage,
	Aliases:      []string{"mv", "rename"},
	Short:        "Move instances",
	Long:         moveCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, params := TypeNamesParams(cmd)
		if len(names) == 0 && len(params) == 2 && strings.HasPrefix(params[0], "@") && strings.HasPrefix(params[1], "@") {
			names = params
		}
		if len(names) == 1 && len(params) == 1 && strings.HasPrefix(params[0], "@") {
			names = append(names, params[0])
		}
		if len(names) != 2 {
			return geneos.ErrInvalidArgs
		}

		return instance.Copy(ct, names[0], names[1], true)
	},
}
