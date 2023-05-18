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
	"fmt"
	"os"
	"strings"

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

var disableCmd = &cobra.Command{
	Use:     "disable [TYPE] [NAME...]",
	GroupID: GROUP_MANAGE,
	Short:   "Disable instances",
	Long: strings.ReplaceAll(`

Mark any matching instances as disabled. The instances are also
stopped.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := CmdArgsParams(cmd)
		return instance.ForAll(ct, Hostname, disableInstance, args, params)
	},
}

func disableInstance(c geneos.Instance, params []string) (err error) {
	if instance.IsDisabled(c) {
		return nil
	}

	if instance.IsProtected(c) {
		return geneos.ErrProtected
	}

	if disableCmdStop {
		if c.Type().RealComponent {
			if err = instance.Stop(c, true, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return
			}
		}
	}

	if !instance.IsProtected(c) || disableCmdForce {
		if err = instance.Disable(c); err == nil {
			fmt.Printf("%s disabled\n", c)
			return nil
		}
	}

	return fmt.Errorf("not disabled. Instances must not be running or use the '--force'/'-F' option")
}
