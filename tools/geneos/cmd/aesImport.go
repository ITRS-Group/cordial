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
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// aesImportCmd represents the 'aes import' command
var aesImportCmd = &cobra.Command{
	Use:   "import [-N] [-k FILE|URL|-] [-C CRC] [TYPE] [NAME...]",
	Short: "Import AES key file",
	Long: `Import AES key file for matching instances. Either a path or URL to a
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
		f, _, err := geneos.OpenLocalFileOrURL(aesImportCmdKeyfile)
		if err != nil {
			return err
		}
		defer f.Close()
		a, err := config.ReadAESValues(f)
		if err != nil {
			return err
		}
		crc, err := a.Checksum()
		if err != nil {
			return err
		}
		crcstr := fmt.Sprintf("%08X", crc)

		// at this point we have an AESValue struct and a CRC to use as a test
		// create 'keyfiles' directory as required
		for _, ct := range ct.Range(&gateway.Gateway, &netprobe.Netprobe, &san.San) {
			for _, h := range host.AllHosts() {
				path := h.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes")
				if err := h.MkdirAll(filepath.Dir(path), 0700); err != nil {
					log.Error().Err(err).Msgf("host %s, component %s", h, ct)
					continue
				}
				if _, err := h.Stat(path); err == nil {
					log.Error().Msgf("keyfile already exists. host %s, component %s", h, ct)
					continue
				}
				w, err := h.Create(path, 0600)
				if err != nil {
					log.Error().Err(err).Msgf("host %s, component %s", h, ct)
					continue
				}
				// don't defer file close here, it's a loop over multiple files
				if err = a.WriteAESValues(w); err != nil {
					w.Close()
					log.Error().Err(err).Msgf("host %s, component %s", h, ct)
					continue
				}
				w.Close()
			}
		}
		params = []string{crcstr}

		if len(params) == 0 && aesImportCmdCRC != "" {
			// no file saved above and we have a CRC on the command line
			// we don't check anything now, just use the CRC as given
			params = []string{aesImportCmdCRC}
		}

		if len(params) == 0 {
			fmt.Println("nothing to do")
			return nil
		}

		// params[0] is the CRC
		instance.ForAll(ct, aesImportAESInstance, args, params)
		return nil
	},
}

var aesImportCmdKeyfile, aesImportCmdCRC string
var aesImportCmdNoRoll bool

func init() {
	aesCmd.AddCommand(aesImportCmd)

	defKeyFile := geneos.UserConfigFilePaths("keyfile.aes")[0]
	aesImportCmd.Flags().StringVarP(&aesImportCmdKeyfile, "keyfile", "k", defKeyFile, "Keyfile to use")
	aesImportCmd.Flags().StringVarP(&aesImportCmdCRC, "crc", "C", "", "CRC of keyfile to use.")
	aesImportCmd.Flags().BoolVarP(&aesImportCmdNoRoll, "noroll", "N", false, "Do not roll any existing keyfile to previous keyfile setting")
}

func aesImportAESInstance(c geneos.Instance, params []string) (err error) {
	var rolled bool

	path := c.Host().Filepath(c.Type(), c.Type().String()+"_shared", "keyfiles", params[0])

	// roll old file
	if !aesImportCmdNoRoll {
		p := c.Config().GetString("keyfile")
		if p != "" {
			if p == path {
				fmt.Printf("%s: new and existing keyfile have same CRC. Not updating.", c)
			} else {
				c.Config().Set("keyfile", path)
				fmt.Printf("%s keyfile %s set", c, params[0])
				c.Config().Set("prevkeyfile", p)
				rolled = true
			}
		}
	} else {
		c.Config().Set("keyfile", path)
		fmt.Printf("%s keyfile %s set", c, params[0])
	}

	if rolled {
		fmt.Printf(", existing keyfile moved to prevkeyfile\n")
	} else {
		fmt.Println()
	}

	// in case the configuration in in old format
	if err = instance.Migrate(c); err != nil {
		log.Fatal().Err(err).Msg("cannot migrate existing .rc config to set values in new .json configuration file")
	}

	if err = instance.WriteConfig(c); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	return
}
