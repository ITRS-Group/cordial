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
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

var setHostCmdPrompt bool
var setHostCmdPassword, setHostDefaultKeyfile, setHostCmdKeyfile string

func init() {
	setCmd.AddCommand(setHostCmd)

	setHostDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

	setHostCmd.Flags().BoolVarP(&setHostCmdPrompt, "prompt", "p", false, "Prompt for password")
	setHostCmd.Flags().StringVarP(&setHostCmdPassword, "password", "P", "", "Password")
	setHostCmd.Flags().StringVarP(&setHostCmdKeyfile, "keyfile", "k", "", "Keyfile")
}

var setHostCmd = &cobra.Command{
	Use:   "host [flags] [NAME...] [KEY=VALUE...]",
	Short: "Set remote host configuration value",
	Long: strings.ReplaceAll(`
Set parameters in the remote host configurations.

Parameters set using the |geneos set host| command will be written or 
updated in file |~/.config/geneos/hosts.json|.

**Note**: In case you set a parameter that is not supported, that parameter
will be written to the |json| configuration file, but will have any effect.
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
			if password, err = readPassword(); err != nil {
				return
			}
		} else if setHostCmdPassword != "" {
			if password, err = encodePassword([]byte(setHostCmdPassword)); err != nil {
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

func encodePassword(plaintext []byte) (encpw string, err error) {
	r, _, err := geneos.OpenSource(setHostCmdKeyfile)
	if err != nil {
		return "", err
	}
	defer r.Close()
	a, err := config.ReadAESValues(r)
	if err != nil {
		return "", err
	}

	e, err := a.EncodeAESBytes(plaintext)
	if err != nil {
		return "", err
	}

	home, _ := os.UserHomeDir()
	if strings.HasPrefix(setHostCmdKeyfile, home) {
		setHostCmdKeyfile = "~" + strings.TrimPrefix(setHostCmdKeyfile, home)
	}
	encpw = fmt.Sprintf("${enc:%s:+encs+%s}", setHostCmdKeyfile, e)
	return
}

func readPassword() (encpw string, err error) {
	var plaintext []byte
	var match bool
	for i := 0; i < 3; i++ {
		plaintext = utils.ReadPasswordPrompt()
		plaintext2 := utils.ReadPasswordPrompt("Re-enter Password")
		if bytes.Equal(plaintext, plaintext2) {
			match = true
			break
		}
		fmt.Println("Passwords do not match. Please try again.")
	}
	if !match {
		return "", fmt.Errorf("too many attempts, giving up")
	}
	return encodePassword(plaintext)
}
