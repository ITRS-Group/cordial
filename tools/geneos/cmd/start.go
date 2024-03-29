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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var startCmdLogs bool
var startCmdExtras string
var startCmdEnvs instance.NameValues

func init() {
	GeneosCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&startCmdExtras, "extras", "x", "", "Extra args passed to process, split on spaces and quoting ignored")
	startCmd.Flags().VarP(&startCmdEnvs, "env", "e", "Extra environment variable (Repeat as required)")
	startCmd.Flags().BoolVarP(&startCmdLogs, "log", "l", false, "Follow logs after starting instance")
	startCmd.Flags().SortFlags = false
}

//go:embed _docs/start.md
var startCmdDescription string

var startCmd = &cobra.Command{
	Use:          "start [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Start Instances",
	Long:         startCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	RunE: func(cmd *cobra.Command, origargs []string) error {
		ct, names, params := ParseTypeNamesParams(cmd)
		var autostart bool
		// if we have a TYPE and at least one NAME then autostart is on
		if ct != nil && len(origargs) > 1 {
			autostart = true
		}
		return Start(ct, startCmdLogs, autostart, names, params)
	},
}

// Start is a single entrypoint for multiple commands to start
// instances. ct is the component type, nil means all. watchlogs is a
// flag to, well, watch logs while autostart is a flag to indicate if
// Start() is being called as part of a group of instances - this is for
// use by autostart checking.
func Start(ct *geneos.Component, watchlogs bool, autostart bool, names []string, params []string) (err error) {
	instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *instance.Response) {
		resp = instance.NewResponse(i)
		if instance.IsAutoStart(i) || autostart {
			resp.Err = instance.Start(i, instance.StartingExtras(startCmdExtras), instance.StartingEnvs(startCmdEnvs))
		}
		return
	}).Write(os.Stdout, instance.WriterIgnoreErr(geneos.ErrRunning))

	if watchlogs {
		// also watch STDERR on start-up
		// never returns
		return followLogs(ct, names, true)
	}

	return
}
