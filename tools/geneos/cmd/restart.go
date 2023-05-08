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
	"errors"
	"os"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var restartCmdAll, restartCmdKill, restartCmdForce, restartCmdLogs bool

func init() {
	RootCmd.AddCommand(restartCmd)

	restartCmd.Flags().BoolVarP(&restartCmdAll, "all", "a", false, "Start all matching instances, not just those already running")
	restartCmd.Flags().BoolVarP(&restartCmdForce, "force", "F", false, "Force restart of protected instances")
	restartCmd.Flags().BoolVarP(&restartCmdKill, "kill", "K", false, "Force stop by sending an immediate SIGKILL")
	restartCmd.Flags().BoolVarP(&restartCmdLogs, "log", "l", false, "Run 'logs -f' after starting instance(s)")

	restartCmd.Flags().SortFlags = false
}

var restartCmd = &cobra.Command{
	Use:   "restart [flags] [TYPE] [NAME...]",
	Short: "Restart instances",
	Long: strings.ReplaceAll(`
Restart the matching instances. This is identical to running |geneos
stop| followed by |geneos start| except if the |-a| flag is given
then all matching instances are started regardless of whether they
were stopped by the command. The command also accepts the same flags
as both start and stop.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return commandRestart(ct, args, params)
	},
}

func commandRestart(ct *geneos.Component, args []string, params []string) (err error) {
	if err = instance.ForAll(ct, restartInstance, args, params); err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	if restartCmdLogs {
		// never returns
		return followLogs(ct, args, params)
	}
	return
}

func restartInstance(c geneos.Instance, params []string) (err error) {
	err = instance.Stop(c, restartCmdForce, false)
	if err == nil || errors.Is(err, os.ErrProcessDone) || restartCmdAll {
		return instance.Start(c)
	}
	return
}
