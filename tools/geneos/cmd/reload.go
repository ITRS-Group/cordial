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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func init() {
	GeneosCmd.AddCommand(reloadCmd)
}

//go:embed _docs/reload.md
var reloadCmdDescription string

var reloadCmd = &cobra.Command{
	Use:          "reload [TYPE] [NAME...]",
	GroupID:      CommandGroupProcess,
	Short:        "Reload Instance Configurations",
	Long:         reloadCmdDescription,
	Aliases:      []string{"refresh"},
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := TypeNames(cmd)
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, reloadInstance)
		responses.Write(os.Stdout)
	},
}

func reloadInstance(c geneos.Instance) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if err := c.Reload(); err == nil {
		resp.Result = fmt.Sprintf("%s: reload signal sent", c)
	}
	return
}
