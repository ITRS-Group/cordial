package main

import (
	"os"

	pdCmd "github.com/itrs-group/cordial/integrations/pagerduty/cmd"
	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	dv2html "github.com/itrs-group/cordial/tools/dv2html/cmd"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/aescmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/cfgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/hostcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/initcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/pkgcmd"
	_ "github.com/itrs-group/cordial/tools/geneos/cmd/tlscmd"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{cmd.GeneosCmd, "tools/geneos"},

	{dv2html.DV2HTMLCmd, "tools/dv2html"},

	{snowCmd.RootCmd, "integrations/servicenow"},
	{pdCmd.RootCmd, "integrations/pagerduty"},
}

func main() {
	for _, d := range doclist {
		os.MkdirAll(d.dir, 0775)
		if err := doc.GenMarkdownTree(d.command, d.dir); err != nil {
			panic(err)
		}
	}
}
