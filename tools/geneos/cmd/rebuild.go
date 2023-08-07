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
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var rebuildCmdForce, rebuildCmdReload bool

func init() {
	GeneosCmd.AddCommand(rebuildCmd)

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
		AnnotationWildcard:  "true",
		AnnotationNeedsHome: "true",
	},
	Run: func(cmd *cobra.Command, _ []string) {
		ct, names := TypeNames(cmd)
		responses := instance.Do(geneos.GetHost(Hostname), ct, names, rebuildInstance)
		responses.Write(os.Stdout)
	},
}

func rebuildInstance(c geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(c)

	if resp.Err = c.Rebuild(rebuildCmdForce); resp.Err != nil {
		return
	}
	log.Debug().Msgf("%s configuration rebuilt (if supported)", c)
	if !rebuildCmdReload {
		return
	}
	return reloadInstance(c)
}
