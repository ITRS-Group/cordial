/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tlscmd

import (
	"crypto/x509"
	_ "embed"
	"fmt"
	"os"
	"path"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var importCmdCert, importCmdSigningBundle, importCmdChain, importCmdPrivateKey string
var importCmdPassword *config.Plaintext

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmdPassword = &config.Plaintext{}
	importCmd.Flags().StringVarP(&importCmdCert, "instance-bundle", "c", "", "Instance certificate bundle to import, PEM or PFX/PKCS#12 format")
	importCmd.Flags().VarP(importCmdPassword, "password", "p", "Password for private key decryption, if needed, for pfx files")

	importCmd.Flags().StringVarP(&importCmdSigningBundle, "signing-bundle", "C", "", "Signing certificate bundle to import, PEM format")

	importCmd.Flags().StringVarP(&importCmdPrivateKey, "key", "k", "", "Private key `file` for certificate, PEM format")
	importCmd.Flags().MarkDeprecated("key", "include the private key in either the instance or signing bundles")
	importCmd.Flags().StringVar(&importCmdChain, "chain", "", "Certificate chain `file` to import, PEM format")
	importCmd.Flags().MarkDeprecated("chain", "include the trust chain in either the instance or signing bundles")

	importCmd.Flags().SortFlags = false

	importCmd.MarkFlagsMutuallyExclusive("instance-bundle", "signing-bundle")
	importCmd.MarkFlagsOneRequired("instance-bundle", "signing-bundle")

	importCmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
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
$ geneos tls import netprobe localhost -c /path/to/file.pem
$ geneos tls import --signing-bundle /path/to/file.pem
`,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names := cmd.ParseTypeNames(command)
		if len(names) == 0 {
			return fmt.Errorf("%w: no instance names specified", geneos.ErrInvalidArgs)
		}

		if importCmdSigningBundle != "" {
			return geneos.TLSImportBundle(importCmdSigningBundle, importCmdPrivateKey, importCmdChain)
		}

		if importCmdCert != "" {
			if path.Ext(importCmdCert) == ".pfx" || path.Ext(importCmdCert) == ".p12" {
				if importCmdPassword.String() == "" {
					importCmdPassword, err = config.ReadPasswordInput(false, 0, "Password")
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to read password")
						// return err
					}
				}
				pfxData, err := os.ReadFile(importCmdCert)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to read PFX file")
					// return err
				}
				key, c, chain, err := pkcs12.DecodeChain(pfxData, importCmdPassword.String())
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to decode PFX file")
					// return err
				}
				pk, err := x509.MarshalPKCS8PrivateKey(key)
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to marshal private key")
					// return err
				}
				k := memguard.NewEnclave(pk)
				instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, c, k, chain).Report(os.Stdout)
				return nil
			}

			certs, err := config.ReadInputPEMString(importCmdCert, "instance certificate(s)")
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to read instance certificate(s)")
				// return err
			}
			key, err := config.ReadInputPEMString(importCmdPrivateKey, "instance key")
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to read instance key")
				// return err
			}
			c, k, chain, err := geneos.DecomposePEM(certs, key)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to decompose PEM")
				// return err
			}
			if c == nil || k == nil {
				return fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
			}

			instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, c, k, chain).Report(os.Stdout)
			return nil
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
		fmt.Printf("%s certificate chain written using %s\n", cordial.ExecutableName(), importCmdChain)

		return
	},
}

// tlsWriteInstance expects 3 params, of *x509.Certificate,
// *memguard.Enclave and a []*x509.Certificate or it will return an
// error or panic.
func tlsWriteInstance(i geneos.Instance, params ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	cf := i.Config()

	if len(params) != 3 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	// validate and convert params
	cert, ok := params[0].(*x509.Certificate)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[0] not a certificate", geneos.ErrInvalidArgs)
		return
	}
	key, ok := params[1].(*memguard.Enclave)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[1] not a secure enclave", geneos.ErrInvalidArgs)
		return
	}
	chain, ok := params[2].([]*x509.Certificate)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[2] not a slice of certificates", geneos.ErrInvalidArgs)
		return
	}

	if resp.Err = instance.WriteCertificate(i, cert); resp.Err != nil {
		return
	}
	resp.Details = append(resp.Details, fmt.Sprintf("%s certificate written", i))

	if resp.Err = instance.WritePrivateKey(i, key); resp.Err != nil {
		return
	}
	resp.Details = append(resp.Details, fmt.Sprintf("%s private key written", i))

	if len(chain) > 0 {
		chainfile := path.Join(i.Home(), "chain.pem")
		if err := certs.WriteCertificates(i.Host(), chainfile, chain...); err == nil {
			resp.Details = append(resp.Details, fmt.Sprintf("%s certificate chain written", i))
			if cf.GetString("certchain") == chainfile {
				return
			}
			cf.SetString("certchain", chainfile, config.Replace("home"))
		}
	}

	resp.Err = instance.SaveConfig(i)
	return
}
