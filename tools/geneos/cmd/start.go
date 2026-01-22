/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
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
		CmdGlobal:        "true",
		CmdRequireHome:   "true",
		CmdWildcardNames: "true",
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
	instance.Do(geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.Response) {
		resp = responses.NewResponse(i)
		if instance.IsAutoStart(i) || autostart {
			resp.Err = instance.Start(i,
				instance.StartingExtras(startCmdExtras),
				instance.StartingEnvs(startCmdEnvs),
			)
		}
		return
	}).Report(os.Stdout, responses.IgnoreErr(geneos.ErrRunning))

	if watchlogs {
		// also watch STDERR on start-up
		// never returns
		return followLogs(ct, names, true)
	}

	return
}
