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
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	GeneosCmd.AddCommand(homeCmd)
}

//go:embed _docs/home.md
var homeCmdDescription string

var homeCmd = &cobra.Command{
	Use:     "home [flags] [TYPE] [NAME]",
	GroupID: CommandGroupView,
	Short:   "Display Instance and Component Home Directories",
	Long:    homeCmdDescription,
	Example: strings.ReplaceAll(`
cd $(geneos home)
cd $(geneos home gateway example1)
cat $(geneos home gateway example2)/gateway.txt
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, names, _ := ParseTypeNamesParams(cmd)

		if len(names) == 0 {
			if ct == nil {
				fmt.Println(geneos.Root())
				return nil
			}
			fmt.Println(geneos.LOCAL.PathTo(ct))
			return nil
		}

		i, err := instance.ByName(geneos.GetHost(Hostname), ct, names[0])

		if err != nil || !i.Host().IsLocal() {
			fmt.Println(geneos.Root())
			return nil
		}

		fmt.Println(i.Home())
		return nil
	},
}
