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
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

var setHostCmdPrompt bool
var setHostCmdPassword, setHostDefaultKeyfile, setHostCmdKeyfile string

func init() {
	setCmd.AddCommand(setHostCmd)

	setHostDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

	setHostCmd.Flags().BoolVarP(&setHostCmdPrompt, "prompt", "p", false, "Prompt for password")
	setHostCmd.Flags().StringVarP(&setHostCmdPassword, "password", "P", "", "Password")
	setHostCmd.Flags().StringVarP(&setHostCmdKeyfile, "keyfile", "k", "", "Keyfile")

	setHostCmd.Flags().SortFlags = false
}

var setHostCmd = &cobra.Command{
	Use:   "host [flags] [NAME...] [KEY=VALUE...]",
	Short: "Set remote host configuration value",
	Long: strings.ReplaceAll(`
Set options on remote host configurations.
`, "|", "`"),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "false",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		_, args, params := cmdArgsParams(cmd)
		var password string
		var hosts []*host.Host

		if len(args) == 0 {
			hosts = host.RemoteHosts()
		} else {
			for _, a := range args {
				h := host.Get(a)
				if h != nil && h.Exists() {
					hosts = append(hosts, h)
				}
			}
		}
		if len(hosts) == 0 {
			// nothing to do
			fmt.Println("nothing to do")
			return nil
		}

		if setHostCmdKeyfile == "" {
			setHostCmdKeyfile = setHostDefaultKeyfile
		}

		// check for passwords
		if setHostCmdPrompt {
			if password, err = config.EncodePasswordPrompt(setHostCmdKeyfile, true); err != nil {
				return
			}
		} else if setHostCmdPassword != "" {
			if password, err = config.EncodeWithKeyfile([]byte(setHostCmdPassword), setHostCmdKeyfile, true); err != nil {
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

		if err = host.WriteConfig(); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	},
}
