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
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var importCmdCert, importCmdSigningBundle, importCmdChain, importCmdPrivateKey string

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importCmdCert, "instance-bundle", "c", "", "Instance certificate bundle to import, PEM format")
	importCmd.Flags().StringVarP(&importCmdSigningBundle, "signing-bundle", "C", "", "Signing certificate bundle to import, PEM format")

	importCmd.Flags().StringVarP(&importCmdPrivateKey, "key", "k", "", "Private key `file` for certificate, PEM format")
	importCmd.Flags().MarkDeprecated("key", "include the private key in either the instance or signing bundles")
	importCmd.Flags().StringVar(&importCmdChain, "chain", "", "Certificate chain `file` to import, PEM format")
	importCmd.Flags().MarkDeprecated("chain", "include the trust chain in either the instance or signing bundles")

	importCmd.Flags().SortFlags = false

	importCmd.MarkFlagsMutuallyExclusive("instance-bundle", "signing-bundle")
	importCmd.MarkFlagsOneRequired("instance-bundle", "signing-bundle")

	importCmd.PersistentFlags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		switch name {
		case "privkey":
			name = "key"
		case "signing":
			name = "signing-bundle"
		case "cert":
			name = "instance-bundle"
		}
		return pflag.NormalizedName(name)
	})
}

//go:embed _docs/import.md
var importCmdDescription string

var importCmd = &cobra.Command{
	Use:                   "import [flags] [TYPE] [NAME...]",
	Short:                 "Import certificates",
	Long:                  importCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
# import file.pem and extract parts
$ geneos tls import netprobe file.pem
$ geneos tls import --signer file.pem
`,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "explicit-none",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.ParseTypeNames(command)

		if importCmdCert != "" && importCmdSigningBundle != "" {
			return errors.New("you can only import an instance *or* a signing certificate, not both")
		}

		if importCmdSigningBundle != "" {
			return geneos.TLSImportBundle(importCmdSigningBundle, importCmdPrivateKey, importCmdChain)
		}

		if importCmdCert != "" {
			certs, err := config.ReadInputPEMString(importCmdCert, "instance certificate(s)")
			if err != nil {
				return err
			}
			key, err := config.ReadInputPEMString(importCmdPrivateKey, "instance key")
			if err != nil {
				return err
			}
			c, k, chain, err := geneos.DecomposePEM(certs, key)
			if err != nil {
				return err
			}

			instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, c, k, chain).Write(os.Stdout)
		}

		if importCmdChain == "" {
			return
		}

		b, err := os.ReadFile(importCmdChain)
		if err != nil {
			log.Error().Err(err).Msg("")
			return err
		}
		_, _, chain, err := geneos.DecomposePEM(string(b))
		if err != nil {
			return err
		}
		if err = geneos.WriteChainLocal(chain); err != nil {
			return err
		}
		fmt.Printf("%s certificate chain written using %s\n", cmd.Execname, importCmdChain)

		return
	},
}

// tlsWriteInstance expects 3 params, of *x509.Certificate,
// *memguard.Enclave and a []*x509.Certificate or it will return an
// error or panic.
func tlsWriteInstance(i geneos.Instance, params ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cf := i.Config()

	if len(params) != 3 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	cert, key, chain := params[0].(*x509.Certificate), params[1].(*memguard.Enclave), params[2].([]*x509.Certificate)

	if resp.Err = instance.WriteCert(i, cert); resp.Err != nil {
		return
	}
	resp.Lines = append(resp.Lines, fmt.Sprintf("%s certificate written", i))

	if resp.Err = instance.WriteKey(i, key); resp.Err != nil {
		return
	}
	resp.Lines = append(resp.Lines, fmt.Sprintf("%s private key written", i))

	if len(chain) > 0 {
		chainfile := path.Join(i.Home(), "chain.pem")
		if err := config.WriteCertChain(i.Host(), chainfile, chain...); err == nil {
			resp.Lines = append(resp.Lines, fmt.Sprintf("%s certificate chain written", i))
			if cf.GetString("certchain") == chainfile {
				return
			}
			cf.SetString("certchain", chainfile, config.Replace("home"))
		}
	}

	resp.Err = instance.SaveConfig(i)
	return
}
