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

var stopCmdForce, stopCmdKill bool

func init() {
	GeneosCmd.AddCommand(stopCmd)

	stopCmd.Flags().BoolVarP(&stopCmdForce, "force", "F", false, "Stop protected instances")
	stopCmd.Flags().BoolVarP(&stopCmdKill, "kill", "K", false, "Force immediate stop by sending an immediate SIGKILL")

	stopCmd.Flags().SortFlags = false
}

//go:embed _docs/stop.md
var stopCmdDescription string

var stopCmd = &cobra.Command{
	Use:          "stop [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Stop Instances",
	Long:         stopCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(cmd)
		instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, a ...any) (resp *instance.Response) {
			resp = instance.NewResponse(i)
			resp.Err = instance.Stop(i, stopCmdForce, stopCmdKill)
			return
		}).Write(os.Stdout,
			instance.WriterShowTimes(),
			instance.WriterTimingFormat("%s stopped in %.2fs\n"),
		)
	},
}
