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
	GeneosCmd.AddCommand(copyCmd)
}

//go:embed _docs/copy.md
var copyCmdDescription string

var copyCmd = &cobra.Command{
	Use:          "copy [TYPE] SOURCE DESTINATION",
	GroupID:      CommandGroupManage,
	Aliases:      []string{"cp"},
	Short:        "Copy instances",
	Long:         copyCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		if len(args) == 0 && len(params) == 2 && strings.HasPrefix(params[0], "@") && strings.HasPrefix(params[1], "@") {
			args = params
		}
		if len(args) == 1 && len(params) == 1 && strings.HasPrefix(params[0], "@") {
			args = append(args, params[0])
		}
		if len(args) != 2 {
			return geneos.ErrInvalidArgs
		}

		return instance.Copy(ct, args[0], args[1], false)
	},
}
