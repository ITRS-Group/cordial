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

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var startCmdLogs bool

func init() {
	GeneosCmd.AddCommand(startCmd)

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
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, origargs []string) error {
		ct, args, params := CmdArgsParams(cmd)
		var autostart bool
		// if we have a TYPE and at least one NAME then autostart is on
		if ct != nil && len(origargs) > 1 {
			autostart = true
		}
		return Start(ct, startCmdLogs, autostart, args, params)
	},
}

// Start is a single entrypoint for multiple commands to start
// instances. ct is the component type, nil means all. watchlogs is a
// flag to, well, watch logs while autostart is a flag to indicate if
// Start() is being called as part of a group of instances - this is for
// use by autostart checking.
func Start(ct *geneos.Component, watchlogs bool, autostart bool, args []string, params []string) (err error) {
	if err = instance.ForAll(ct, Hostname, func(c geneos.Instance, _ ...any) error {
		if instance.IsAutoStart(c) || autostart {
			return instance.Start(c)
		}
		return nil
	}, args); err != nil {
		return
	}

	if watchlogs {
		// also watch STDERR on start-up
		// never returns
		return followLogs(ct, args, true)
	}

	return
}
