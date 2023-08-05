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
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, args := TypeNames(command)

		instance.Do(geneos.GetHost(Hostname), ct, args, protectInstance, !protectCmdUnprotect)
	},
}

func protectInstance(c geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)
	cf := c.Config()

	if len(params) == 0 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}
	protect, ok := params[0].(bool)
	if !ok {
		panic("wrong param")
	}

	cf.Set("protected", protect)

	if cf.Type == "rc" {
		resp.Err = instance.Migrate(c)
	} else {
		resp.Err = instance.SaveConfig(c)
	}
	if resp.Err != nil {
		return
	}

	if protect {
		resp.Completed = append(resp.Completed, "protected")
	} else {
		resp.Completed = append(resp.Completed, "unprotected")
	}

	return
}
