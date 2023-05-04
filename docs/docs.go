package main

import (
	"os"

	pdCmd "github.com/itrs-group/cordial/integrations/pagerduty/cmd"
	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	geneosCmd "github.com/itrs-group/cordial/tools/geneos/cmd"
	geneosAESCmd "github.com/itrs-group/cordial/tools/geneos/cmd/aes"
	geneosHostCmd "github.com/itrs-group/cordial/tools/geneos/cmd/host"
	geneosPackageCmd "github.com/itrs-group/cordial/tools/geneos/cmd/package"
	geneosTLSCmd "github.com/itrs-group/cordial/tools/geneos/cmd/tls"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{geneosCmd.RootCmd, "tools/geneos"},
	{geneosAESCmd.AesCmd, "tools/geneos/aes"},
	{geneosHostCmd.HostCmd, "tools/geneos/host"},
	{geneosPackageCmd.PackageCmd, "tools/geneos/package"},
	{geneosTLSCmd.TLSCmd, "tools/geneos/tls"},
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
