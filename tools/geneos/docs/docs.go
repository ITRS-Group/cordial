package main

import (
	"github.com/spf13/cobra/doc"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

func main() {
	// doc.GenManTree(cmd.RootCmd(), nil, "./")
	doc.GenMarkdownTree(cmd.RootCmd(), "./")
}
