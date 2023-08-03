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

var disableCmdForce, disableCmdStop bool

func init() {
	GeneosCmd.AddCommand(disableCmd)

	disableCmd.Flags().BoolVarP(&disableCmdStop, "stop", "S", false, "Stop instances")
	disableCmd.Flags().BoolVarP(&disableCmdForce, "force", "F", false, "Force disable instances")
	disableCmd.Flags().SortFlags = false
}

//go:embed _docs/disable.md
var disableCmdDescription string

var disableCmd = &cobra.Command{
	Use:          "disable [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Disable instances",
	Long:         disableCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names := TypeNames(cmd)
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, disableInstance)
		instance.WriteResponseStrings(os.Stdout, responses)
		return
	},
}

func disableInstance(c geneos.Instance) (result instance.Response) {
	if instance.IsDisabled(c) {
		return
	}

	if instance.IsProtected(c) {
		result.Err = geneos.ErrProtected
		return
	}

	if disableCmdStop {
		if c.Type().RealComponent {
			if err := instance.Stop(c, true, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
				result.Err = err
				return
			}
		}
	}

	if !disableCmdForce && instance.IsRunning(c) {
		fmt.Printf("%s is running, skipping. Use the `--stop` option to stop running instances\n", c)
		return
	}

	if !instance.IsProtected(c) || disableCmdForce {
		if result.Err = instance.Disable(c); result.Err == nil {
			result.String = fmt.Sprintf("%s disabled", c)
			return
		}
	}

	result.Err = fmt.Errorf("not disabled. Instances must not be running or use the '--force'/'-F' option")
	return
}
