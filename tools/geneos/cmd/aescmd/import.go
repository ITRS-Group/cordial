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

package aescmd

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/utils"
)

var aesImportCmdKeyfile config.KeyFile
var aesImportCmdHostname string

func init() {
	AesCmd.AddCommand(aesImportCmd)

	aesImportCmdKeyfile = cmd.DefaultUserKeyfile

	aesImportCmd.Flags().VarP(&aesImportCmdKeyfile, "keyfile", "k", "Keyfile to use")
	aesImportCmd.Flags().StringVarP(&aesImportCmdHostname, "host", "H", "", "Import only to named `host`, default is all")
	aesImportCmd.Flags().SortFlags = false

}

var aesImportCmd = &cobra.Command{
	Use:   "import [flags] [TYPE] [NAME...]",
	Short: "Import shared keyfiles for components",
	Long: strings.ReplaceAll(`
Import keyfiles to component TYPE's shared directory.

The argument given with the |-k| flag can be a local file, which can have
a prefix of |~/| to represent the user's home directory, a URL or a dash
|-| for STDIN. If no |-k| flag is given then the user's default
keyfile is imported, if found.

If a TYPE is given then the key is only imported to that component
type, otherwise the keyfile is imported to all supported components.
Currently only Gateways and Netprobes (including SANs) are supported.

Keyfiles are imported to all configured hosts unless |-H| is used to
limit to a specific host.

Instance names can be given to indirectly identify the component
type.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, _, _ := cmd.CmdArgsParams(command)

		a, err := aesImportCmdKeyfile.Read()
		if err != nil {
			return err
		}

		h := geneos.Get(aesImportCmdHostname)

		// at this point we have an AESValue struct and a CRC to use as
		// the filename base. create 'keyfiles' directory as required
		for _, ct := range ct.Range(componentsWithKeyfiles...) {
			for _, h := range h.Range(geneos.AllHosts()...) {
				aesImportSave(ct, h, &a)
			}
		}

		return nil
	},
}

func aesImportSave(ct *geneos.Component, h *geneos.Host, a *config.KeyValues) (err error) {
	if ct == nil || h == nil || a == nil {
		return cmd.ErrInvalidArgs
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

	if err = a.Write(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	return
}
