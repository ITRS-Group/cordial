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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/spf13/cobra"
)

// aesNewCmd represents the aesNew command
var aesNewCmd = &cobra.Command{
	Use:                   "new",
	Short:                 "Create a new AES key file",
	Long:                  ``,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		var buf bytes.Buffer
		var crc uint32

		a, err := config.NewAESValues()
		if err != nil {
			return
		}
		err = a.WriteAESValues(&buf)
		if err != nil {
			return
		}

		// write keyfile to STDOUT unless told otherwise
		w := os.Stdout

		if aesNewCmdKeyfile != "" {
			if _, err := os.Stat(aesNewCmdKeyfile); err == nil {
				return fs.ErrExist
			}
			w, err = os.OpenFile(aesNewCmdKeyfile, os.O_RDWR|os.O_CREATE, 0600)
			if err != nil {
				return
			}
			defer w.Close()
		}

		k := buf.Bytes()
		_, err = w.Write(k)
		if err != nil {
			return
		}

		r := bytes.NewReader(k)

		crc, err = config.Checksum(r)
		if err != nil {
			return
		}
		crcstr := fmt.Sprintf("%08X", crc)

		if aesNewCmdKeyfile != "" {
			fmt.Printf("%s created, checksum %s\n", aesNewCmdKeyfile, crcstr)
		}

		if aesNewCmdSetSync {
			var keyfile string

			ct, args, _ := cmdArgsParams(cmd)
			if ct == nil {
				for _, ct := range []*geneos.Component{&gateway.Gateway, &netprobe.Netprobe} {
					if aesNewCmdKeyfile == "" {
						keyfile = host.LOCAL.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes")
						log.Debug().Msgf("writing %d bytes of %q to keyfile %q", len(k), string(k), keyfile)
						if err = os.WriteFile(keyfile, k, 0600); err != nil {
							log.Error().Err(err).Msg("")
							return
						}
					} else {
						keyfile, _ = filepath.Abs(aesNewCmdKeyfile)
					}

					for _, h := range host.RemoteHosts() {
						log.Debug().Msgf("copying to host %s", h)
						host.CopyFile(host.LOCAL, keyfile, h, h.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes"))
					}
					// set configs - only Gateways
					if ct == nil {
						ct = &gateway.Gateway
					}
					if ct != &gateway.Gateway {
						return geneos.ErrInvalidArgs
					}

					params := []string{crcstr + ".aes"}
					err = instance.ForAll(ct, aesNewSetInstance, args, params)
					if err != nil {
						return
					}
				}
			} else {
				if aesNewCmdKeyfile == "" {
					keyfile = host.LOCAL.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes")
					os.WriteFile(keyfile, k, 0600)
				} else {
					keyfile, _ = filepath.Abs(aesNewCmdKeyfile)
				}
				for _, h := range host.RemoteHosts() {
					log.Debug().Msgf("copying to host %s", h)
					host.CopyFile(host.LOCAL, keyfile, h, h.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes"))
				}
				// set configs - only Gateways
				if ct == nil {
					ct = &gateway.Gateway
				}
				if ct != &gateway.Gateway {
					return geneos.ErrInvalidArgs
				}

				params := []string{crcstr + ".aes"}
				return instance.ForAll(ct, aesNewSetInstance, args, params)

			}
		}
		return
	},
}

var aesNewCmdKeyfile string
var aesNewCmdSetSync bool

func init() {
	aesCmd.AddCommand(aesNewCmd)

	aesNewCmd.Flags().StringVarP(&aesNewCmdKeyfile, "keyfile", "k", "", "Optional key file to create, defaults to STDOUT")
	aesNewCmd.Flags().BoolVarP(&aesNewCmdSetSync, "set", "S", false, "Set instances to use this keyfile. Remote hosts have keyfile synced.")
}

func aesNewSetInstance(c geneos.Instance, params []string) (err error) {
	c.Config().Set("keyfile", c.Host().Filepath(c.Type(), c.Type().String()+"_shared", "keyfiles", params[0]))

	// now loop through the collected results and write out
	if err = instance.Migrate(c); err != nil {
		log.Fatal().Err(err).Msg("cannot migrate existing .rc config to set values in new .json configuration file")
	}

	if err = instance.WriteConfig(c); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return
}
