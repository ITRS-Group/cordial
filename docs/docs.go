package main

import (
	"os"

	snowCmd "github.com/itrs-group/cordial/integrations/servicenow/cmd"
	geneosCmd "github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{geneosCmd.RootCmd(), "tools/geneos"},
	{snowCmd.RootCmd(), "integrations/servicenow"},
}

func main() {
	for _, d := range doclist {
		os.MkdirAll(d.dir, 0775)
		if err := doc.GenMarkdownTree(d.command, d.dir); err != nil {
			panic(err)
		}
	}

}
