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

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/spf13/cobra"
)

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:                   "disable [TYPE] [NAME...]",
	Short:                 "Stop and disable instances",
	Long:                  `Mark any matching instances as disabled. The instances are also stopped.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		return instance.ForAll(ct, disableInstance, args, params)
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
	disableCmd.Flags().SortFlags = false
}

func disableInstance(c geneos.Instance, params []string) (err error) {
	if instance.IsDisabled(c) {
		return nil
	}

	uid, gid, _, err := utils.GetIDs(c.GetConfig().GetString("user"))
	if err != nil {
		return
	}

	if err = instance.Stop(c, false); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return
	}

	disablePath := instance.ComponentFilepath(c, geneos.DisableExtension)

	h := c.Host()

	f, err := h.Create(disablePath, 0664)
	if err != nil {
		return err
	}
	f.Close()

	if utils.IsSuperuser() {
		if err = h.Chown(disablePath, uid, gid); err != nil {
			h.Remove(disablePath)
		}
	}

	return
}
