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

var allCmdLicenseFile, allCmdArchive string

func init() {
	initCmd.AddCommand(allCmd)

	allCmd.Flags().StringVarP(&allCmdLicenseFile, "licence", "L", "geneos.lic", "Licence file location")
	allCmd.MarkFlagRequired("licence")

	allCmd.Flags().StringVarP(&allCmdArchive, "archive", "A", "", archiveOptionsText)
	allCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", instance.GatewaysOptionstext)

	allCmd.Flags().SortFlags = false
}

//go:embed _docs/all.md
var allCmdDescription string

var allCmd = &cobra.Command{
	Use:   "all [flags] [USERNAME] [DIRECTORY]",
	Short: "Initialise a more complete Geneos environment",
	Long:  allCmdDescription,
	Example: strings.ReplaceAll(`
geneos init all -L https://myserver/files/geneos.lic -u email@example.com
geneos init all -L ~/geneos.lic -A ~/downloads /opt/itrs
sudo geneos init all -L /tmp/geneos-1.lic -u email@example.com myuser /opt/geneos
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, args, params := cmd.TypeNamesParams(command)
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

		options = append(options, geneos.Archive(allCmdArchive))
		return initAll(geneos.LOCAL, options...)
	},
}

func initAll(h *geneos.Host, options ...geneos.Options) (err error) {
	e := []string{}

	if initCmdName == "" {
		initCmdName = h.Hostname()
	}

	if err = install("licd", h.String(), options...); err != nil {
		return
	}
	if err = install("gateway", h.String(), options...); err != nil {
		return
	}
	if err = install("netprobe", h.String(), options...); err != nil {
		return
	}
	if err = install("webserver", h.String(), options...); err != nil {
		return
	}

	if err = cmd.AddInstance(geneos.ParseComponent("licd"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.ImportFiles(geneos.ParseComponent("licd"), []string{initCmdName}, []string{"geneos.lic=" + allCmdLicenseFile}); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("gateway"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	// if len(initCmdExtras.Gateways) == 0 {
	// 	initCmdExtras.Gateways.Set("localhost")
	// }
	if err = cmd.AddInstance(geneos.ParseComponent("netprobe"), initCmdExtras, []string{}, "localhost@"+h.String()); err != nil {
		return
	}
	if err = cmd.AddInstance(geneos.ParseComponent("webserver"), initCmdExtras, []string{}, initCmdName); err != nil {
		return
	}
	if err = cmd.Start(nil, initCmdLogs, true, e, e); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	return cmd.CommandPS(nil, e, e)
}
