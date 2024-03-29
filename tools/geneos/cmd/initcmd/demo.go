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
	demoCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.GatewaysOptionstext)
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
		ct, args, params := cmd.ParseTypeNamesParams(command)
		log.Debug().Msgf("%s %v %v", ct, args, params)
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

		if err = initMisc(command); err != nil {
			return
		}

		options = append(options, geneos.Archive(demoCmdArchive))
		return initDemo(geneos.LOCAL, options...)
	},
}

func initDemo(h *geneos.Host, options ...geneos.Options) (err error) {
	e := []string{}
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

	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	cmd.CommandPS(nil, e, e)
	return
}
