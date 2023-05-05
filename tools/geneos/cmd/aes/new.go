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

package aes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
)

var aesNewCmdKeyfile, aesNewCmdHostname, aesNewCmdBackupKeyfile string
var aesNewCmdImport, aesNewCmdSaveDefault, aesNewCmdOverwriteKeyfile bool

var aesDefaultKeyfile = geneos.UserConfigFilePaths("keyfile.aes")[0]

func init() {
	AesCmd.AddCommand(aesNewCmd)

	aesNewCmd.Flags().StringVarP(&aesNewCmdKeyfile, "keyfile", "k", "", "Optional key file to create, defaults to STDOUT. (Will NOT overwrite without -f)")
	aesNewCmd.Flags().BoolVarP(&aesNewCmdSaveDefault, "default", "D", false, "Save as user default keyfile (will NOT overwrite without -f)")
	aesNewCmd.Flags().StringVarP(&aesNewCmdBackupKeyfile, "backup", "b", ".old", "Backup existing keyfile with extension given")
	aesNewCmd.Flags().BoolVarP(&aesNewCmdOverwriteKeyfile, "overwrite", "f", false, "Overwrite existing keyfile")
	aesNewCmd.Flags().BoolVarP(&aesNewCmdImport, "import", "I", false, "Import the keyfile to components and set on matching instances.")
	aesNewCmd.Flags().StringVarP(&aesNewCmdHostname, "host", "H", "", "Import only to named host, default is all")

	aesNewCmd.MarkFlagsMutuallyExclusive("keyfile", "default")
}

var aesNewCmd = &cobra.Command{
	Use:   "new [flags] [TYPE] [NAME...]",
	Short: "Create a new key file",
	Long: strings.ReplaceAll(`
Create a new key file. Written to STDOUT by default, but can be
written to a file with the |-k FILE| option.

If the flag |-I| is given then the new key file is imported to the
shared directories of matching components, using |CRC32.aes| as the
file base name, where CRC32 is an 8 digit hexadecimal checksum to
help distinguish keyfiles. Currently limited to Gateway and Netprobe
types, including SANs, for use by Toolkit Secure Environment
Variables.

Additionally, when using the |-I| flag any matching Gateway instances
have any existing |keyfile| path setting moved to the |prevkeyfile|
setting to support GA6.x key file rolling.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var crc uint32

		a, err := config.NewAESValues()
		if err != nil {
			return
		}

		if aesNewCmdSaveDefault {
			aesNewCmdKeyfile = cmd.DefaultUserKeyfile
		}

		if aesNewCmdKeyfile != "" {
			if _, err := os.Stat(aesNewCmdKeyfile); err == nil {
				if !aesNewCmdOverwriteKeyfile {
					return fs.ErrExist
				}
				if aesNewCmdBackupKeyfile != "" {
					if err = os.Rename(aesNewCmdKeyfile, aesNewCmdKeyfile+aesNewCmdBackupKeyfile); err != nil {
						return fmt.Errorf("keyfile backup failed: %w", err)
					}
				}
			}
			if err = os.MkdirAll(filepath.Dir(aesNewCmdKeyfile), 0775); err != nil {
				return fmt.Errorf("failed to create keyfile directory %q: %w", filepath.Dir(aesNewCmdKeyfile), err)
			}
			if err = os.WriteFile(aesNewCmdKeyfile, []byte(a.String()), 0600); err != nil {
				return fmt.Errorf("failed to write keyfile to %q: %w", aesNewCmdKeyfile, err)
			}
		} else if !aesNewCmdImport {
			fmt.Print(a)
		}

		crc, err = config.ChecksumString(a.String())
		if err != nil {
			return
		}
		crcstr := fmt.Sprintf("%08X", crc)

		if aesNewCmdKeyfile != "" {
			fmt.Printf("%s created, checksum %s\n", aesNewCmdKeyfile, crcstr)
		}

		if aesNewCmdImport {
			if aesNewCmdKeyfile == "" {
				fmt.Printf("saving keyfile with checksum %s\n", crcstr)
			}

			ct, args, _ := cmd.CmdArgsParams(command)
			h := host.Get(aesNewCmdHostname)

			for _, ct := range ct.Range(componentsWithKeyfiles...) {
				for _, h := range h.Range(host.AllHosts()...) {
					aesImportSave(ct, h, &a)
				}
			}

			if ct == nil {
				ct = &gateway.Gateway
			}

			// set configs only on Gateways for now
			if ct != &gateway.Gateway {
				return
			}

			params := []string{crcstr + ".aes"}
			instance.ForAll(ct, aesNewSetInstance, args, params)
			return
		}
		return
	},
}

func aesNewSetInstance(c geneos.Instance, params []string) (err error) {
	var rolled bool
	// roll old file
	p := c.Config().GetString("keyfile")
	if p != "" {
		c.Config().Set("prevkeyfile", p)
		rolled = true
	}
	c.Config().Set("keyfile", c.Host().Filepath(c.Type(), c.Type().String()+"_shared", "keyfiles", params[0]))

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
