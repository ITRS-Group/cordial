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
		AnnotationWildcard:  "explicit",
		AnnotationNeedsHome: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := TypeNames(command)
		results := instance.Do(geneos.GetHost(Hostname), ct, names, deleteInstance)
		results.Write(os.Stdout)
	},
}

func deleteInstance(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	if instance.IsProtected(i) {
		resp.Err = geneos.ErrProtected
		return
	}

	if deleteCmdStop {
		if i.Type().RealComponent {
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
