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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// aesUpdateCmd represents the 'aes update' command
var aesUpdateCmd = &cobra.Command{
	Use:   "update [-N] [-k FILE|URL|-] [-C CRC] [TYPE] [NAME...]",
	Short: "Update AES key file",
	Long: `Update AES key file for matching instances. Either a path or URL to a
keyfile or the CRC of an existing keyfile in the component's shared
directory must be given. If a path or URL is given then the keyfile
is saved to the component shared directories and the configuration
references that path. Unless the '-N' flag is given any existing
keyfile is copied to the 'prevkeyfile' setting to support key file
updating in Geneos GA6.x.

The argument given with the '-k' flag can be a local file (including
a prefix of '~/' to represent the home directory), a URL or a dash
'-' for STDIN.

If both '-k' and '-C' are given, then the new keyfile is first
checked if it can be accessed and only if that fails will the CRC
check be performed.

Currently only Gateways and Netprobes (and SANs) are supported.`,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, _ := cmdArgsParams(cmd)
		var params []string

		// first copy any file from source to shared dir(s) unless it exists
		if aesUpdateCmdKeyfile != "" {
			f, _, err := geneos.OpenLocalFileOrURL(aesUpdateCmdKeyfile)
			if err != nil {
				//
			}
			defer f.Close()
			a, err := config.ReadAESValues(f)
			if err != nil {
				//
			}
			crc, err := a.Checksum()
			if err != nil {
				//
			}
			crcstr := fmt.Sprintf("%08X", crc)

			// at this point we have an AESValue struct and a CRC to use as a test
			for _, h := range host.AllHosts() {
				if ct == nil {
					//
				}
				path := h.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes")
				if _, err := h.Stat(path); err == nil {
					// something exists

				}
				w, err := h.Create(path, 0600)
				if err != nil {
					//
				}
				if err = a.WriteAESValues(w); err != nil {
					w.Close()
					//
				}
				w.Close()
			}
			params = []string{crcstr}
		}

		if len(params) == 0 && aesUpdateCmdCRC != "" {
			// no file saved above and we have a CRC on the command line
			// we don't check anything now, just use the CRC as given
			params = []string{aesUpdateCmdCRC}
		}

		if len(params) == 0 {
			fmt.Println("nothing to do")
			return nil
		}

		return instance.ForAll(ct, aesUpdateAESInstance, args, params)
	},
}

var aesUpdateCmdKeyfile, aesUpdateCmdCRC string
var aesUpdateCmdNoRoll bool

func init() {
	aesCmd.AddCommand(aesUpdateCmd)

	aesUpdateCmd.Flags().StringVarP(&aesUpdateCmdKeyfile, "keyfile", "k", "", "Keyfile to use.")
	aesUpdateCmd.Flags().StringVarP(&aesUpdateCmdCRC, "crc", "C", "", "CRC of keyfile to use.")
	aesUpdateCmd.Flags().BoolVarP(&aesUpdateCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

func aesUpdateAESInstance(c geneos.Instance, params []string) (err error) {
	var rolled bool

	// roll old file
	if !aesUpdateCmdNoRoll {
		p := c.Config().GetString("keyfile")
		if p != "" {
			c.Config().Set("prevkeyfile", p)
			rolled = true
		}
	}
	c.Config().Set("keyfile", c.Host().Filepath(c.Type(), c.Type().String()+"_shared", "keyfiles", params[0]))

	// in case the configuration in in old format
	if err = instance.Migrate(c); err != nil {
		log.Fatal().Err(err).Msg("cannot migrate existing .rc config to set values in new .json configuration file")
	}

	if err = instance.WriteConfig(c); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	fmt.Printf("%s keyfile %s set", c, params[0])
	if rolled {
		fmt.Printf(", existing keyfile moved to prevkeyfile\n")
	} else {
		fmt.Println()
	}
	return
}
