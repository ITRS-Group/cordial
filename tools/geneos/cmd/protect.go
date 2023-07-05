/*
Copyright © 2023 ITRS Group

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
	"fmt"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

var protectCmdUnprotect bool

func init() {
	GeneosCmd.AddCommand(protectCmd)

	protectCmd.Flags().BoolVarP(&protectCmdUnprotect, "unprotect", "U", false, "unprotect instances")
}

//go:embed _docs/protect.md
var protectCmdDescription string

var protectCmd = &cobra.Command{
	Use:          "protect [TYPE] [NAME...]",
	GroupID:      CommandGroupManage,
	Short:        "Mark instances as protected",
	Long:         protectCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := CmdArgs(command)

		return instance.ForAll(ct, Hostname, protectInstance, args, []string{fmt.Sprintf("%v", !protectCmdUnprotect)})
	},
}

func protectInstance(c geneos.Instance, params []string) (err error) {
	cf := c.Config()

	var protect bool
	if len(params) > 0 {
		protect = params[0] == "true"
	}
	cf.Set("protected", protect)

	if cf.Type == "rc" {
		err = instance.Migrate(c)
	} else {
		err = instance.SaveConfig(c)
	}
	if err != nil {
		return
	}

	if protect {
		fmt.Printf("%s set to protected\n", c)
	} else {
		fmt.Printf("%s set to unprotected\n", c)
	}

	return
}
