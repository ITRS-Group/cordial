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
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var initFloatingCmdArchive, initFloatingCmdVersion, initFloatingCmdOverride string

func init() {
	initCmd.AddCommand(initFloatingCmd)

	initFloatingCmd.Flags().StringVarP(&initFloatingCmdVersion, "version", "V", "latest", "Download this `VERSION`, defaults to latest. Doesn't work for EL8 archives.")
	initFloatingCmd.Flags().StringVarP(&initFloatingCmdArchive, "archive", "A", "", archiveOptionsText)
	initFloatingCmd.Flags().StringVarP(&initFloatingCmdOverride, "override", "T", "", "Override the `[TYPE:]VERSION` for archive files with non-standard names")

	initFloatingCmd.Flags().VarP(&initCmdExtras.Gateways, "gateway", "g", instance.GatewayValuesOptionstext)
	initFloatingCmd.Flags().VarP(&initCmdExtras.Attributes, "attribute", "a", instance.AttributeValuesOptionsText)
	initFloatingCmd.Flags().VarP(&initCmdExtras.Types, "type", "t", instance.TypeValuesOptionsText)
	initFloatingCmd.Flags().VarP(&initCmdExtras.Variables, "variable", "v", instance.VarValuesOptionsText)

	initFloatingCmd.Flags().SortFlags = false
}

//go:embed _docs/floating.md
var initFloatingCmdDescription string

var initFloatingCmd = &cobra.Command{
	Use:          "floating [flags] [USERNAME] [DIRECTORY]",
	Short:        "Initialise a Geneos Floating Netprobe environment",
	Long:         initFloatingCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.CmdArgsParams(command)
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

		if err = geneos.Init(geneos.LOCAL, options...); err != nil {
			log.Fatal().Err(err).Msg("")
		}

		if err = initMisc(command); err != nil {
			return
		}

		// prefix with netprobe
		if initFloatingCmdOverride != "" && !strings.Contains(initFloatingCmdOverride, ":") {
			initFloatingCmdOverride = "netprobe:" + initFloatingCmdOverride
		}

		options = append(options,
			geneos.Source(initFloatingCmdArchive),
			geneos.Version(initFloatingCmdVersion),
			geneos.OverrideVersion(initFloatingCmdOverride),
		)
		return initFloating(geneos.LOCAL, options...)
	},
}

func initFloating(h *geneos.Host, options ...geneos.Options) (err error) {
	var floatingname string

	e := []string{}

	if initCmdName != "" {
		floatingname = initCmdName
	} else {
		floatingname = h.Hostname()
	}
	if !h.IsLocal() {
		floatingname = floatingname + "@" + geneos.LOCALHOST
	}
	if err = install("floating", geneos.LOCALHOST, options...); err != nil {
		return
	}

	if err = cmd.AddInstance(geneos.FindComponent("floating"), initCmdExtras, []string{}, floatingname); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	return cmd.CommandPS(nil, e, e)
}
