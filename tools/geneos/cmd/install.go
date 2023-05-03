/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

func init() {
	RootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install [flags] [TYPE] [FILE|URL...]",
	Short: "Alias for `package install`",
	Long: strings.ReplaceAll(`
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	DisableFlagParsing: true,
	RunE: func(command *cobra.Command, args []string) (err error) {
		return RunE(command.Root(), []string{"package", "install"}, args)
	},
}

// XXX this is a duplicate of the function in package/packageInstall.go
func install(ct *geneos.Component, target string, options ...geneos.Options) (err error) {
	for _, h := range host.Match(target) {
		if err = ct.MakeComponentDirs(h); err != nil {
			return err
		}
		if err = geneos.Install(h, ct, options...); err != nil {
			return
		}
	}
	return
}
