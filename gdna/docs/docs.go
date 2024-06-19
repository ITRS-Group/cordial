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

// The docs utility builds markdown documentation from the geneos program
// sources and writes to the `docs` directory.
package main

import (
	"os"

	"github.com/itrs-group/cordial/gdna/cmd"

	"github.com/spf13/cobra"
)

type docs struct {
	command *cobra.Command
	dir     string
}

var doclist = []docs{
	{cmd.GDNACmd, "."},
}

func main() {
	for _, d := range doclist {
		os.MkdirAll(d.dir, 0775)
		if err := GenMarkdownTree(d.command, d.dir); err != nil {
			panic(err)
		}
	}
}
