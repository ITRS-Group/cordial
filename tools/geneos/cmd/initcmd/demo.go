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

package initcmd

import (
	_ "embed"
	"os"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var demoCmdArchive string

func init() {
	initCmd.AddCommand(demoCmd)

	demoCmd.Flags().StringVarP(&demoCmdArchive, "archive", "A", "", archiveOptionsText)
	demoCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.IncludeValuesOptionsText)
	demoCmd.Flags().SortFlags = false
}

//go:embed _docs/demo.md
var demoCmdDescription string

var demoCmd = &cobra.Command{
	Use:          "demo [flags] [USERNAME] [DIRECTORY]",
	Short:        "Initialise a Geneos Demo environment",
	Long:         demoCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args := cmd.ParseTypeNames(command)
		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error().Err(geneos.ErrInvalidArgs).Msg(ct.String())
			return geneos.ErrInvalidArgs
		}
		options, err := initProcessArgs(args)
		if err != nil {
			return
		}

		if err = geneos.Initialise(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initCommon(command); err != nil {
			return
		}

		options = append(options, geneos.LocalArchive(demoCmdArchive))
		return initDemo(geneos.LOCAL, options...)
	},
}

func initDemo(h *geneos.Host, options ...geneos.PackageOptions) (err error) {
	empty := []string{}
	g := []string{"Demo Gateway@" + h.String()}

	if err = install("gateway", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("gateway"), initCmdExtras, []string{}, "Demo Gateway@"+h.String()); err != nil {
		return
	}
	if err = cmd.Set(geneos.ParseComponent("gateway"), g, []string{"options=-demo"}); err != nil {
		return
	}

	if err = install("netprobe", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("netprobe"), initCmdExtras, []string{}, "localhost@"+h.String()); err != nil {
		return
	}

	if err = install("webserver", geneos.LOCALHOST, options...); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("webserver"), initCmdExtras, []string{}, "demo@"+h.String()); err != nil {
		return
	}

	disp := os.Getenv("DISPLAY")
	if disp != "" {
		if err = install("ac2", geneos.LOCALHOST, options...); err != nil {
			return
		}
		if err = cmd.AddInstance(geneos.ParseComponent("ac2"), initCmdExtras, []string{}, "demo@"+h.String()); err != nil {
			return
		}
	}

	if err = cmd.Start(nil, initCmdLogs, true, empty, empty); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	cmd.CommandPS(nil, empty, empty)
	return
}
