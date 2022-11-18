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

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/netprobe"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/san"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var aesImportCmdKeyfile, aesImportCmdHostname string

func init() {
	aesCmd.AddCommand(aesImportCmd)

	defKeyFile := geneos.UserConfigFilePaths("keyfile.aes")[0]
	aesImportCmd.Flags().StringVarP(&aesImportCmdKeyfile, "keyfile", "k", defKeyFile, "Keyfile to use")
	aesImportCmd.Flags().StringVarP(&aesImportCmdHostname, "host", "H", "", "Import only to named host, default is all")
	aesImportCmd.Flags().SortFlags = false

}

var aesImportCmd = &cobra.Command{
	Use:   "import [flags] [TYPE] [NAME...]",
	Short: "Import shared keyfiles for components",
	Long: strings.ReplaceAll(`
Import keyfiles to the shared directory of the matching components.

The keyfile to import must be defined using:
- Option |-k <keyfile_path>|, where <keyfile_path> can be a file path
  or a URL
- Option |k -| for the key to be read from STDIN.
If no keyfiole is defined, the user's defaut keyfile (typically 
|~/.config/geneos/keyfile.aes|) is imported.

If TYPE or NAME is defined, the keyfile is imported only for the 
corresponding component type.
Otherwise, the keyfile is imported for all supported components
(|gateway|, |netprobe| & |san|).

By default, keyfiles are imported to all configured hosts.
To limit the import to a specific host, use option |-H|.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, _, _ := cmdArgsParams(cmd)

		f, _, err := geneos.OpenSource(aesImportCmdKeyfile)
		if err != nil {
			return err
		}
		defer f.Close()

		a, err := config.ReadAESValues(f)
		if err != nil {
			return err
		}

		h := host.Get(aesImportCmdHostname)

		// at this point we have an AESValue struct and a CRC to use as
		// the filename base. create 'keyfiles' directory as required
		for _, ct := range ct.Range(&gateway.Gateway, &netprobe.Netprobe, &san.San) {
			for _, h := range h.Range(host.AllHosts()...) {
				aesImportSave(ct, h, &a)
			}
		}

		return nil
	},
}

func aesImportSave(ct *geneos.Component, h *host.Host, a *config.AESValues) (err error) {
	if ct == nil || h == nil || a == nil {
		return ErrInvalidArgs
	}

	crc, err := a.Checksum()
	if err != nil {
		return err
	}
	crcstr := fmt.Sprintf("%08X", crc)

	// save given keyfile
	path := h.Filepath(ct, ct.String()+"_shared", "keyfiles", crcstr+".aes")
	if _, err := h.Stat(path); err == nil {
		log.Debug().Msgf("keyfile already exists for host %s, component %s", h, ct)
		return nil
	}
	if err := h.MkdirAll(utils.Dir(path), 0775); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return err
	}
	w, err := h.Create(path, 0600)
	if err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	defer w.Close()

	if err = a.WriteAESValues(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	return
}
