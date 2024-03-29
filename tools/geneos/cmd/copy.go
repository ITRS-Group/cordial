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
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func init() {
	GeneosCmd.AddCommand(copyCmd)

}

//go:embed _docs/copy.md
var copyCmdDescription string

var copyCmd = &cobra.Command{
	Use:          "copy [TYPE] SOURCE DESTINATION [KEY=VALUE...]",
	GroupID:      CommandGroupManage,
	Aliases:      []string{"cp"},
	Short:        "Copy instances",
	Long:         copyCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		AnnotationWildcard:  "false",
		AnnotationNeedsHome: "true",
		AnnotationExpand:    "true",
	},
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		ct, names, params := ParseTypeNamesParams(cmd)
		if len(names) > 2 {
			if !strings.HasPrefix(names[len(names)-1], "@") {
				return errors.New("when copying more than one instance the last argument must be of the form @HOST")
			}
			for _, n := range names[:len(names)-1] {
				log.Debug().Msgf("copy %s to %s", n, names[len(names)-1])
				if err = instance.Copy(ct, n, names[len(names)-1]); err != nil {
					return
				}
			}
			return
		}
		if len(names) == 0 && len(params) >= 2 && strings.HasPrefix(params[0], "@") && strings.HasPrefix(params[1], "@") {
			names = params[0:2]
			params = params[2:]
		} else if len(names) == 1 && len(params) >= 1 && strings.HasPrefix(params[0], "@") {
			names = append(names, params[0])
			params = params[1:]
		}

		if len(names) != 2 {
			return geneos.ErrInvalidArgs
		}

		return instance.Copy(ct, names[0], names[1], instance.Params(params...))
	},
}
