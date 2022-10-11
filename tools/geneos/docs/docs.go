package main

import (
	"os"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/spf13/cobra/doc"
)

const docdir = "../../../docs/tools/geneos/"

func main() {
	os.MkdirAll(docdir, 0775)
	if err := doc.GenMarkdownTree(cmd.RootCmd(), docdir); err != nil {
		panic(err)
	}

}
