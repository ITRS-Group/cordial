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
	"os"
	"strings"

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

var stopCmd = &cobra.Command{
	Use:     "stop [flags] [TYPE] [NAME...]",
	GroupID: GROUP_PROCESS,
	Short:   "Stop instances",
	Long: strings.ReplaceAll(`
Stop the matching instances.

Protected instances will not be restarted unless the |--force|/|-F|
option is given.

Normal behaviour is to send, on Linux, a SIGTERM to the process and
wait for a short period before trying again until the process is no
longer running. If this fails to stop the process a SIGKILL is sent
to terminate the process without further action. If the |--kill|/|-K|
option is used then the terminate signal is sent immediately without
waiting. Beware that this can leave instance files corrupted or in an
indeterminate state.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return instance.ForAll(ct, Hostname, stopInstance, args, params)
	},
}

func stopInstance(c geneos.Instance, params []string) error {
	_, err := instance.GetPID(c)
	if err == os.ErrProcessDone {
		return nil
	}
	return instance.Stop(c, stopCmdForce, stopCmdKill)
}
