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
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var startCmdLogs bool
var startCmdExtras string
var startCmdEnvs values.NameValues
var startCmdPort uint16

func init() {
	Cmd.AddCommand(startCmd)

	startCmd.Flags().Uint16VarP(&startCmdPort, "port", "p", 0, "Start instance matching port (overrides TYPE and NAME)")
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
		CmdGlobal:               "true",
		CmdRequireHome:          "true",
		CmdWildcardNames:        "true",
		CmdNonInstanceArgsError: "true",
		CmdAuditCommand:         "always",
		CmdAuditActions:         "always",
	},
	RunE: func(cmd *cobra.Command, origargs []string) (err error) {
		var autostart bool

		if startCmdPort != 0 {
			instances := []geneos.Instance{}
			for h := range geneos.GetHost(Hostname).OrList(geneos.LOCAL) {
				i, err := instance.ByPort(h, startCmdPort)
				if err != nil {
					continue
				}
				instances = append(instances, i)
			}
			if len(instances) == 0 {
				return fmt.Errorf("no instances using port %d found", startCmdPort)
			}
			instance.DoInstances(cmd.CommandPath(), instances, func(i geneos.Instance, a ...any) (resp *responses.General) {
				resp = responses.New[responses.General](i)
				resp.Err = instance.Start(i,
					instance.StartingExtras(startCmdExtras),
					instance.StartingEnvs(startCmdEnvs),
				)
				resp.Audited = true
				return
			}).Report(os.Stdout, responses.IgnoreErr(geneos.ErrRunning))

			if startCmdLogs {
				// also watch STDERR on start-up
				instance.FollowInstanceLogs(instances, instance.WithStderr(true)) // never returns
			}
			return
		}

		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return err
		}
		// if we have a TYPE and at least one NAME then autostart is on
		if ct != nil && len(origargs) > 1 {
			autostart = true
		}
		return Start(cmd, ct, startCmdLogs, autostart, selectedByName(origargs), names)
	},
}

// Start is a single entrypoint for multiple commands to start
// instances. ct is the component type, nil means all. watchlogs is a
// flag to, well, watch logs while autostart is a flag to indicate if
// Start() is being called as part of a group of instances - this is for
// use by autostart checking.
// selectedByName returns true if any of args, the arguments of a
// command, names an instance, instead of only being a component type,
// "all" or all of the instances on a host.
func selectedByName(args []string) bool {
	return slices.ContainsFunc(args, func(a string) bool {
		return a != "all" && !strings.HasPrefix(a, "@") && geneos.ParseComponent(a) == nil
	})
}

// startReportOptions returns the options to report the results of
// starting instances. Starting an instance that is already running is
// not an error and the result is the same as what was asked for. A
// disabled instance is a failure if it was asked for by name, but is
// reported without failing if it was one of the instances selected
// because they are set to start automatically, so that starting all the
// instances, for example at boot, is not failed by an instance that has
// been deliberately disabled.
func startReportOptions(byName bool) []responses.Option {
	options := []responses.Option{responses.IgnoreErr(geneos.ErrRunning)}
	if !byName {
		options = append(options, responses.WarnErr(geneos.ErrDisabled))
	}
	return options
}

func Start(command *cobra.Command, ct *geneos.Component, watchlogs bool, autostart bool, byName bool, names []string) (err error) {
	err = instance.Do(command.CommandPath(), geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.General) {
		resp = responses.New[responses.General](i)
		if instance.IsAutoStart(i) || autostart {
			resp.Err = instance.Start(i,
				instance.StartingExtras(startCmdExtras),
				instance.StartingEnvs(startCmdEnvs),
			)
			resp.Audited = true
		}
		return
	}).Report(os.Stdout, startReportOptions(byName)...)

	if watchlogs {
		// also watch STDERR on start-up
		instance.FollowLogs(command.CommandPath(), geneos.GetHost(Hostname), ct, names, instance.WithStderr(true)) // never returns
	}

	return
}
