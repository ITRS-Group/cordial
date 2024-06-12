/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
