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
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

var rebuildCmdForce, rebuildCmdReload bool

func init() {
	Cmd.AddCommand(rebuildCmd)

	rebuildCmd.Flags().BoolVarP(&rebuildCmdForce, "force", "F", false, "Force rebuild")
	rebuildCmd.Flags().BoolVarP(&rebuildCmdReload, "reload", "r", false, "Reload instances after rebuild")
	rebuildCmd.Flags().SortFlags = false
}

//go:embed _docs/rebuild.md
var rebuildCmdDescription string

var rebuildCmd = &cobra.Command{
	Use:          "rebuild [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Short:        "Rebuild Instance Configurations From Templates",
	Long:         rebuildCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdGlobal:                "true",
		CmdRequireHome:           "true",
		CmdWildcardNames:         "true",
		CmdAllInstancesMustMatch: "true",
		CmdNonInstanceArgsError:  "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, _, err := FetchArgs(cmd)
		if err != nil {
			return
		}
		return instance.Do(cmd.CommandPath(), geneos.GetHost(Hostname), ct, names, func(i geneos.Instance, _ ...any) (resp *responses.General) {
			return rebuildInstance(i, rebuildCmdForce, rebuildCmdReload)
		}, instance.ProfileOnly).Report(os.Stdout)
	},
}

func writeWithRebuild(i geneos.Instance, rebuild, force bool) *responses.General {
	if !rebuild && !force {
		return instance.Write(i)
	}
	resp := instance.Write(i, instance.NoRebuild())
	if resp.Err != nil {
		return resp
	}
	return responses.MergeResponse(resp, rebuildInstance(i, force, true))
}

func rebuildInstance(i geneos.Instance, force, reload bool) (resp *responses.General) {
	resp = responses.New[responses.General](i)
	if config.Get[string](i.Config(), config.Join("config", "rebuild")) == "never" {
		return
	}

	var changed bool
	if changed, resp.Err = i.Rebuild(force); resp.Err != nil {
		return
	}
	if changed {
		resp.Completed = append(resp.Completed, "configuration rebuilt")
	}
	i.Log().Debug("configuration rebuilt (if supported)", slog.Bool("force", force))
	if !reload {
		return
	}
	if !i.Type().IsA("profile") {
		return responses.MergeResponse(resp, ReloadInstance(i))
	}

	for _, member := range instance.Instances(i.Host(), nil, instance.MatchParameters(config.Join("config", "profile")+"="+i.Name())) {
		if config.Get[string](member.Config(), config.Join("config", "rebuild")) == "never" {
			continue
		}
		if err := member.Reload(); err != nil {
			if !errors.Is(err, geneos.ErrNotSupported) && !errors.Is(err, os.ErrProcessDone) {
				resp.Err = errors.Join(resp.Err, fmt.Errorf("reload %s: %w", member, err))
			}
			continue
		}
		resp.Completed = append(resp.Completed, fmt.Sprintf("reload signal sent to %s", member))
	}
	return
}
