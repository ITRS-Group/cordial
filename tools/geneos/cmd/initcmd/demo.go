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
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/pkgcmd"
	"github.com/itrs-group/cordial/tools/geneos/cmd/pscmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

var demoCmdMinimal bool

func init() {
	initCmd.AddCommand(demoCmd)

	demoCmd.Flags().BoolVarP(&demoCmdMinimal, "minimal", "M", false, "use a minimal Netprobe release")
	demoCmd.Flags().VarP(&initCmdExtras.Includes, "include", "i", values.IncludeValuesOptionsText)

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
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "false",
		cmd.CmdAuditCommand: "always",
		cmd.CmdAuditActions: "always",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		log := log.With("command", "init demo")
		ct, args, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}

		// none of the arguments can be a reserved type
		if ct != nil {
			log.Error(ct.String(), slog.Any("error", geneos.ErrInvalidArgs))
			return geneos.ErrInvalidArgs
		}

		// merge params into args as there may be a directory path in there
		args = append(args, params...)

		options, err := initProcessArgs(args, initCmdExtras)
		if err != nil {
			return
		}

		if err = geneos.Initialise(command, geneos.LOCAL, options...); err != nil {
			log.Error("initialise failed", slog.Any("error", err))
			os.Exit(1)
		}

		if err = initCommon(); err != nil {
			return
		}

		return initDemo(command, geneos.LOCAL, options...)
	},
}

func initDemo(command *cobra.Command, h *geneos.Host, options ...geneos.PackageOption) (err error) {
	empty := []string{}

	ct := geneos.ParseComponent("gateway")
	if err = pkgcmd.Install(h, ct, options...); err != nil {
		return
	}
	var port uint16 = 7038
	if initCmdInsecure {
		port = 7039
	}
	gatewayValues := initCmdExtras
	gatewayValues.Params = []string{"options=-demo"}
	if _, err = instance.Add(h, ct, "Demo Gateway@"+h.String(),
		instance.WithPort(port),
		instance.WithValues(gatewayValues),
	); err != nil {
		return
	}

	netprobeCT := geneos.ParseComponent("netprobe")
	minimalCT := geneos.ParseComponent("minimal")

	if demoCmdMinimal {
		if err = pkgcmd.Install(h, minimalCT, options...); err != nil {
			return
		}
	} else {
		if err = pkgcmd.Install(h, netprobeCT, options...); err != nil {
			return
		}
	}
	probename := "localhost"
	if demoCmdMinimal {
		probename = "minimal:" + probename
	}
	if _, err = instance.Add(h, netprobeCT, probename+"@"+h.String(),
		instance.WithPort(7036),
		instance.WithValues(initCmdExtras),
	); err != nil {
		return
	}

	ct = geneos.ParseComponent("webserver")
	if err = pkgcmd.Install(h, ct, options...); err != nil {
		return
	}
	port = 8443
	if initCmdInsecure {
		port = 8080
	}
	if _, err = instance.Add(h, ct, "Demo Webserver@"+h.String(),
		instance.WithPort(port),
		instance.WithValues(initCmdExtras),
	); err != nil {
		return
	}

	disp := os.Getenv("DISPLAY")
	if h.IsLocalhost() && disp != "" {
		ct = geneos.ParseComponent("ac2")
		if err = pkgcmd.Install(h, ct, options...); err != nil {
			return
		}
		if _, err = instance.Add(h, ct, "Demo AC2@"+h.String(),
			instance.WithValues(initCmdExtras),
		); err != nil {
			return
		}
	}

	if err = cmd.Start(command, nil, initCmdLogs, true, empty); err != nil {
		return
	}
	time.Sleep(time.Second * 2)
	pscmd.CommandPS(command, nil, empty, empty)
	return
}
