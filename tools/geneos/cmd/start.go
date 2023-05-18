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
	"strings"

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

var startCmd = &cobra.Command{
	Use:     "start [flags] [TYPE] [NAME...]",
	GroupID: GROUP_PROCESS,
	Short:   "Start instances",
	Long: strings.ReplaceAll(`
Start the matching instances.

The start-up command and environment can be seen using the |geneos
command| command.

Any matching instances that are marked as |disabled| are not started.

With the |--log|/|-l| option the command will follow the logs of all
instances started, including the STDERR logs as these are good
sources of start-up issues.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return Start(ct, startCmdLogs, args, params)
	},
}

func Start(ct *geneos.Component, watchlogs bool, args []string, params []string) (err error) {
	if err = instance.ForAll(ct, Hostname, func(c geneos.Instance, _ []string) error {
		return instance.Start(c)
	}, args, params); err != nil {
		return
	}

	if watchlogs {
		// watch STDERR on start-up
		logCmdStderr = true
		// never returns
		return followLogs(ct, args, params)
	}

	return
}
