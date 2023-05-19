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

package cfgcmd

import (
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(unsetUserCmd)
}

var unsetUserCmd = &cobra.Command{
	Use:   "unset [KEY...]",
	Short: "Unset a program parameter",
	Long: strings.ReplaceAll(`
Unset removes the program configuration value for any arguments given
on the command line.

No validation is done and there if you mistype a key name it is still
considered valid to remove an non-existing key.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "false",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		var changed bool

		_, args := cmd.CmdArgs(command)
		orig, _ := config.Load(cmd.Execname, config.IgnoreWorkingDir(), config.IgnoreSystemDir())
		new := config.New()

	OUTER:
		for _, k := range orig.AllKeys() {
			for _, a := range args {
				if k == a {
					changed = true
					continue OUTER
				}
			}
			new.Set(k, orig.Get(k))
		}

		if changed {
			log.Debug().Msgf("%v", orig.AllSettings())
			return new.Save(cmd.Execname)
		}
		return nil
	},
}
