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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/spf13/cobra"
)

var restartCmdAll, restartCmdKill, restartCmdForce, restartCmdLogs bool

func init() {
	GeneosCmd.AddCommand(restartCmd)

	restartCmd.Flags().BoolVarP(&restartCmdAll, "all", "a", false, "Start all matching instances, not just those already running")
	restartCmd.Flags().BoolVarP(&restartCmdForce, "force", "F", false, "Force restart of protected instances")
	restartCmd.Flags().BoolVarP(&restartCmdKill, "kill", "K", false, "Force stop by sending an immediate SIGKILL")
	restartCmd.Flags().BoolVarP(&restartCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance(s)")

	restartCmd.Flags().SortFlags = false
}

//go:embed _docs/restart.md
var restartCmdDescription string

var restartCmd = &cobra.Command{
	Use:          "restart [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Restart Instances",
	Long:         restartCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names := TypeNames(cmd)
		return commandRestart(ct, names)
	},
}

func commandRestart(ct *geneos.Component, args []string) (err error) {
	instance.Do(geneos.GetHost(Hostname), ct, args, restartInstance)

	if restartCmdLogs {
		// also watch STDERR on start-up
		// never returns
		return followLogs(ct, args, true)
	}
	return
}

func restartInstance(c geneos.Instance) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if !instance.IsAutoStart(c) {
		return
	}
	resp.Err = instance.Stop(c, restartCmdForce, false)
	if resp.Err == nil || restartCmdAll {
		resp.Err = instance.Start(c)
	}
	return
}
