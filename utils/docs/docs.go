/*
Copyright © 2023 ITRS Group

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

// docs creates documentation from the tools and integrations in the
// repo (except `geneos` which has it's own copy of this with custom
// mods)
//
// The tools derive their name, and so the names in their default file
// names and help text, from the name of the running executable. To get
// documentation that matches each tool, `docs` runs a copy of itself
// under the name of each tool in turn (see the runas package) and each
// copy writes the documentation for just that tool.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/utils/docs/runas"

	gdnaCmd "github.com/itrs-group/cordial/gdna/cmd"

	pdCmd "github.com/itrs-group/cordial/integrations/pagerduty/cmd"
	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	snow2Cmd "github.com/itrs-group/cordial/integrations/servicenow2/cmd"

	// subcommands of servicenow2, which add themselves to the root command
	_ "github.com/itrs-group/cordial/integrations/servicenow2/cmd/client"
	_ "github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"

	dv2email "github.com/itrs-group/cordial/tools/dv2email/cmd"
	gatewayReporter "github.com/itrs-group/cordial/tools/gateway-reporter/cmd"
	imsGatewayCmd "github.com/itrs-group/cordial/tools/ims-gateway/cmd"
	sanCfgCmd "github.com/itrs-group/cordial/tools/san-config/cmd"

	"github.com/spf13/cobra"
)

type docs struct {
	name    string // executable name of the documented tool
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{"gdna", gdnaCmd.Cmd, "../../gdna/docs"},

	{"dv2email", dv2email.Cmd, "../../tools/dv2email/docs"},
	{"ims-gateway", imsGatewayCmd.Cmd, "../../tools/ims-gateway/docs"},
	{"gateway-reporter", gatewayReporter.Cmd, "../../tools/gateway-reporter/docs"},
	{"san-config", sanCfgCmd.Cmd, "../../tools/san-config/docs"},

	{"servicenow", snowCmd.Cmd, "../../integrations/servicenow/docs"},
	{"servicenow2", snow2Cmd.Cmd, "../../integrations/servicenow2/docs"},
	{"pagerduty", pdCmd.Cmd, "../../integrations/pagerduty/docs"},
}

func main() {
	// when run under the name of a documented tool, only generate for that tool
	for _, d := range doclist {
		if d.name == cordial.ExecutableName() {
			if err := generate(d); err != nil {
				slog.Error("generate command documentation", "directory", d.dir, "error", err)
				os.Exit(1)
			}
			return
		}
	}

	if err := generateAll(); err != nil {
		slog.Error("generate command documentation", "error", err)
		os.Exit(1)
	}
}

func generate(d docs) error {
	if err := os.MkdirAll(d.dir, 0775); err != nil {
		return err
	}
	d.command.Use = d.name
	return GenMarkdownTree(d.command, d.dir)
}

// generateAll runs a copy of this executable, named after each tool in
// turn, in the current directory
func generateAll() error {
	for _, d := range doclist {
		if err := runas.Run(d.name); err != nil {
			return fmt.Errorf("%s: %w", d.name, err)
		}
	}
	return nil
}
