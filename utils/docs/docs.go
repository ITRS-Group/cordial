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
package main

import (
	"os"

	gdnaCmd "github.com/itrs-group/cordial/gdna/cmd"

	pdCmd "github.com/itrs-group/cordial/integrations/pagerduty/cmd"
	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	snow2Cmd "github.com/itrs-group/cordial/integrations/servicenow2/cmd"

	dv2email "github.com/itrs-group/cordial/tools/dv2email/cmd"
	gatewayReporter "github.com/itrs-group/cordial/tools/gateway-reporter/cmd"
	imsGatewayCmd "github.com/itrs-group/cordial/tools/ims-gateway/cmd"
	sanCfgCmd "github.com/itrs-group/cordial/tools/san-config/cmd"

	"github.com/spf13/cobra"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{gdnaCmd.Cmd, "../../gdna/docs"},

	{dv2email.Cmd, "../../tools/dv2email/docs"},
	{imsGatewayCmd.Cmd, "../../tools/ims-gateway/docs"},
	{gatewayReporter.Cmd, "../../tools/gateway-reporter/docs"},
	{sanCfgCmd.Cmd, "../../tools/san-config/docs"},

	{snowCmd.Cmd, "../../integrations/servicenow/docs"},
	{snow2Cmd.Cmd, "../../integrations/servicenow2/docs"},
	{pdCmd.Cmd, "../../integrations/pagerduty/docs"},
}

func main() {
	for _, d := range doclist {
		os.MkdirAll(d.dir, 0775)
		if err := GenMarkdownTree(d.command, d.dir); err != nil {
			panic(err)
		}
	}
}
