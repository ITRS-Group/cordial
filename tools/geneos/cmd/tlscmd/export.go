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
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var exportCmdOutput string
var exportCmdNoRoot bool

func init() {
	tlsCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&exportCmdOutput, "output", "o", "", "Output destination, default to stdout")
	exportCmd.Flags().BoolVarP(&exportCmdNoRoot, "no-root", "N", false, "Do not include the root CA certificate")

	exportCmd.Flags().SortFlags = false
}

//go:embed _docs/export.md
var exportCmdDescription string

var exportCmd = &cobra.Command{
	Use:                   "export [flags] [TYPE] [NAME...]",
	Short:                 "Export certificates",
	Long:                  exportCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
# export 
$ geneos tls export --output file.pem
`,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "false",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		// gather the rootCA cert, the geneos cert and key
		root, rootFile, err := geneos.ReadRootCert(true)
		if err != nil {
			err = fmt.Errorf("local root certificate (%s) not valid: %w", rootFile, err)
			return
		}
		signer, signerFile, err := geneos.ReadSigningCert(true)
		if err != nil {
			err = fmt.Errorf("local signing root certificate (%s) not valid: %w", signerFile, err)
			return
		}
		signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"))
		if err != nil {
			return
		}

		var pembytes []byte

		pembytes = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: signer.Raw,
		})

		if !exportCmdNoRoot {
			pembytes = append(pembytes, pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: root.Raw,
			})...)
		}

		l, _ := signingKey.Open()
		pembytes = append(pembytes, pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: l.Bytes(),
		})...)
		l.Destroy()

		if exportCmdOutput != "" {
			return os.WriteFile(exportCmdOutput, pembytes, 0600)
		}

		fmt.Println(string(pembytes))
		return
	},
}
