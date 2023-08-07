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
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(enableCmd)

	enableCmd.Flags().BoolVarP(&enableCmdStart, "start", "S", false, "Start enabled instances")

	enableCmd.Flags().SortFlags = false
}

var enableCmdStart bool

//go:embed _docs/enable.md
var enableCmdDescription string

var enableCmd = &cobra.Command{
	Use:          "enable [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Enable instance",
	Long:         enableCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "explicit",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := TypeNames(cmd)
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, enableInstance)
		responses.Write(os.Stdout)
	},
}

func enableInstance(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if !instance.IsDisabled(c) {
		return
	}
	if resp.Err = instance.Enable(c); resp.Err == nil {
		resp.Completed = append(resp.Completed, "enabled")
		if enableCmdStart {
			resp.Err = instance.Start(c)
			return
		}
	}
	return
}
