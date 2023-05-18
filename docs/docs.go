package main

import (
	"os"

	pdCmd "github.com/itrs-group/cordial/integrations/pagerduty/cmd"
	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	dv2email "github.com/itrs-group/cordial/tools/dv2email/cmd"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/aescmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/cfgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/hostcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/initcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/pkgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/tlscmd"

	"github.com/spf13/cobra"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{cmd.GeneosCmd, "../tools/geneos/docs/commands"},

	{dv2email.DV2EMAILCmd, "tools/dv2email"},

	{snowCmd.RootCmd, "integrations/servicenow"},
	{pdCmd.RootCmd, "integrations/pagerduty"},
}

func main() {
	for _, d := range doclist {
		os.MkdirAll(d.dir, 0775)
		if err := GenMarkdownTree(d.command, d.dir); err != nil {
			panic(err)
		}
	}
}
