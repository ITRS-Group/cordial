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

package tlscmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

func init() {
	tlsCmd.AddCommand(tlsSyncCmd)
}

//go:embed _docs/sync.md
var tlsSyncCmdDescription string

var tlsSyncCmd = &cobra.Command{
	Use:          "sync",
	Short:        "Sync remote hosts certificate chain files",
	Long:         tlsSyncCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "false",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		return tlsSync()
	},
}

// tlsSync creates and copies a certificate chain file to all remote
// hosts
//
// the cert chain is kept in the geneos tls directory, not the app
// config directory
func tlsSync() (err error) {
	rootCert, err := instance.ReadRootCert()
	if err != nil {
		rootCert = nil
	}
	geneosCert, err := instance.ReadSigningCert()
	if err != nil {
		return os.ErrNotExist
	}

	if rootCert == nil && geneosCert == nil {
		return
	}

	for _, r := range geneos.AllHosts() {
		tlsPath := r.Filepath("tls")
		if err = r.MkdirAll(tlsPath, 0775); err != nil {
			return
		}
		chainpath := filepath.Join(tlsPath, geneos.ChainCertFile)
		if err = config.WriteCerts(r, chainpath, rootCert, geneosCert); err != nil {
			return
		}

		fmt.Printf("Updated certificate chain %s pem on %s\n", chainpath, r.String())
	}
	return
}
