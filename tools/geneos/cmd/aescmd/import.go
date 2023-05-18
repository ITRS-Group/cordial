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
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var aesImportCmdKeyfile config.KeyFile

func init() {
	AesCmd.AddCommand(aesImportCmd)

	aesImportCmdKeyfile = cmd.DefaultUserKeyfile

	aesImportCmd.Flags().VarP(&aesImportCmdKeyfile, "keyfile", "k", "Keyfile to use `PATH|URL|-`")
	aesImportCmd.MarkFlagRequired("keyfile")
	aesImportCmd.Flags().SortFlags = false

}

var aesImportCmd = &cobra.Command{
	Use:   "import [flags] [TYPE] [NAME...]",
	Short: "Import key files for component TYPE",
	Long: strings.ReplaceAll(`
Import keyfiles to the TYPE |keyfiles| directory in each matching
component TYPE shared directory.

A key file must be provided with the |--keyfile|/|-k| option. The
option value can be a path or a URL or a '-' to read from STDIN. A
prefix of |~/| to the path interprets the rest relative to the home
directory.

The key file is copied from the supplied source to a file with the
base-name of its 8-hexadecimal digit checksum to distinguish it from
other key files. In all examples the CRC is shown as |DEADBEEF| in
honour of many generations of previous UNIX documentation. There is a
very small chance of a checksum clash.

The shared directory for each component is one level above instance
directories and has a |_shared| suffix. The convention is to use this
path for Geneos instances to share common configurations and
resources. e.g. for a Gateway the path would be
|.../gateway/gateway_shared/keyfiles| where instance directories
would be |.../gateway/gateways/NAME|

If a TYPE is given then the key is only imported for that component,
otherwise the key file is imported to all components that are known to
support key files. Currently only Gateways and Netprobes (including
SANs) are supported.

Key files are imported to all configured hosts unless |--host|/|-H| is
used to limit to a specific host.

Instance names can be given to indirectly identify the component
type.
`, "|", "`"),
	Example: strings.ReplaceAll(`
# import local keyfile.aes to GENEOS/gateway/gateway_shared/DEADBEEF.aes
geneos aes import --keyfile ~/keyfile.aes gateway

# import a remote keyfile to the remote Geneos host named |remote1|
geneos aes import -k https://myserver.example.com/secure/keyfile.aes -H remote1
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, _ := cmd.CmdArgs(command)

		a, err := aesImportCmdKeyfile.Read()
		if err != nil {
			return err
		}

		h := geneos.GetHost(cmd.Hostname)

		// at this point we have an AESValue struct and a CRC to use as
		// the filename base. create 'keyfiles' directory as required
		for _, ct := range ct.OrList(componentsWithKeyfiles...) {
			for _, h := range h.OrList(geneos.AllHosts()...) {
				aesImportSave(ct, h, a)
			}
		}

		return nil
	},
}

func aesImportSave(ct *geneos.Component, h *geneos.Host, a *config.KeyValues) (err error) {
	if ct == nil || h == nil || a == nil {
		return geneos.ErrInvalidArgs
	}

	crc, err := a.Checksum()
	if err != nil {
		return err
	}
	crcstr := fmt.Sprintf("%08X", crc)

	// save given keyfile
	file := ct.SharedPath(h, "keyfiles", crcstr+".aes")
	if _, err := h.Stat(file); err == nil {
		log.Debug().Msgf("keyfile %s already exists for host %s, component %s", file, h, ct)
		return nil
	}
	if err := h.MkdirAll(path.Dir(file), 0775); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return err
	}
	w, err := h.Create(file, 0600)
	if err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	defer w.Close()

	if err = a.Write(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	fmt.Printf("key file %s.aes saved to shared directory for %s on %s\n", crcstr, ct, h)
	return
}
