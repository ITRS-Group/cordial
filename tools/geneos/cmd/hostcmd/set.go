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

package hostcmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var setCmdPrompt bool
var setCmdPassword *config.Plaintext
var setCmdKeyfile config.KeyFile

func init() {
	hostCmd.AddCommand(setCmd)

	setCmdPassword = &config.Plaintext{}

	setCmd.Flags().BoolVarP(&setCmdPrompt, "prompt", "p", false, "Prompt for password")
	setCmd.Flags().VarP(setCmdPassword, "password", "P", "password")
	setCmd.Flags().VarP(&setCmdKeyfile, "keyfile", "k", "Keyfile")

	setCmd.Flags().SortFlags = false
}

//go:embed _docs/set.md
var setCmdDescription string

var setCmd = &cobra.Command{
	Use:                   "set [flags] [NAME...] [KEY=VALUE...]",
	Short:                 "Set host configuration value",
	Long:                  setCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "false",
	},
	RunE: func(command *cobra.Command, origargs []string) (err error) {
		if len(origargs) == 0 && command.Flags().NFlag() == 0 {
			return command.Usage()
		}
		_, args, params := cmd.ParseTypeNamesParams(command)
		var password string
		var hosts []*geneos.Host

		if len(args) == 0 {
			hosts = geneos.RemoteHosts(false)
		} else {
			for _, a := range args {
				h := geneos.GetHost(a)
				if h.Exists() {
					hosts = append(hosts, h)
				}
			}
		}
		if len(hosts) == 0 {
			// nothing to do
			fmt.Println("nothing to do")
			return nil
		}

		if setCmdKeyfile == "" {
			setCmdKeyfile = cmd.DefaultUserKeyfile
		}

		// check for passwords
		if setCmdPrompt {
			if password, err = setCmdKeyfile.EncodePasswordInput(host.Localhost, true); err != nil {
				return
			}
		} else if !setCmdPassword.IsNil() {
			if password, err = setCmdKeyfile.Encode(host.Localhost, setCmdPassword, true); err != nil {
				return
			}
		}

		for _, h := range hosts {
			for _, set := range params {
				if !strings.Contains(set, "=") {
					continue
				}
				s := strings.SplitN(set, "=", 2)
				k, v := s[0], s[1]
				h.Set(k, v)
			}

			if password != "" {
				h.Set("password", password)
			}
		}

		if err = geneos.SaveHostConfig(); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	},
}
