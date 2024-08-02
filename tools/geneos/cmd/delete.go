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
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var deleteCmdStop, deleteCmdForce bool

func init() {
	GeneosCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteCmdStop, "stop", "S", false, "Stop instances first")
	deleteCmd.Flags().BoolVarP(&deleteCmdForce, "force", "F", false, "Force delete of protected instances")

	deleteCmd.Flags().SortFlags = false
}

//go:embed _docs/delete.md
var deleteCmdDescription string

var deleteCmd = &cobra.Command{
	Use:          "delete [flags] [TYPE] [NAME...]",
	GroupID:      CommandGroupConfig,
	Aliases:      []string{"rm"},
	Short:        "Delete Instances",
	Long:         deleteCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		CmdNoneMeansAll: "explicit",
		CmdRequireHome:  "true",
		CmdGlobNames:    "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := ParseTypeNames(command)
		instance.Do(geneos.GetHost(Hostname), ct, names, deleteInstance).Write(os.Stdout)
	},
}

func deleteInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if instance.IsProtected(i) && !deleteCmdForce {
		resp.Err = geneos.ErrProtected
		return
	}

	if deleteCmdStop {
		if i.Type() != &geneos.RootComponent {
			if err := instance.Stop(i, true, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
				resp.Err = err
				return
			}
		}
	}

	if !instance.IsRunning(i) || deleteCmdForce {
		if instance.IsRunning(i) {
			if resp.Err = instance.Stop(i, true, false); resp.Err != nil {
				return
			}
		}
		if resp.Err = i.Host().RemoveAll(i.Home()); resp.Err != nil {
			return
		}
		resp.Completed = append(resp.Completed, fmt.Sprintf("deleted %s:%s", i.Host().String(), i.Home()))
		i.Unload()
		return
	}

	resp.Err = fmt.Errorf("not deleted. Instances must not be running or use the '--force'/'-F' option")
	return
}
