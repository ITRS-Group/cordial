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

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var importCmdChain, importCmdPrivateKey string
var importCmdPassword *config.Plaintext

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmdPassword = &config.Plaintext{}
	// importCmd.Flags().StringVarP(&importCmdCert, "instance-bundle", "c", "", "Instance certificate bundle to import, PEM or PFX/PKCS#12 format")
	// importCmd.Flags().StringVarP(&importCmdSigningBundle, "signing-bundle", "C", "", "Signing certificate bundle to import, PEM format")
	importCmd.Flags().VarP(importCmdPassword, "password", "p", "Plaintext password for PFX/PKCS#12 file decryption.\nYou will be prompted if not supplied as an argument.\nPFX/PKCS#12 files are identified by the .pfx or .p12\nfile extension and only supported for instance bundles")

	importCmd.Flags().StringVarP(&importCmdPrivateKey, "key", "k", "", "Private key `file` for certificate, PEM format")
	importCmd.Flags().MarkDeprecated("key", "include the private key in either the instance or signing bundles")
	importCmd.Flags().StringVar(&importCmdChain, "chain", "", "Certificate chain `file` to import, PEM format")
	importCmd.Flags().MarkDeprecated("chain", "include the trust chain in either the instance or signing bundles")

	importCmd.Flags().SortFlags = false

	// importCmd.MarkFlagsMutuallyExclusive("instance-bundle", "signing-bundle")
	// importCmd.MarkFlagsOneRequired("instance-bundle", "signing-bundle")

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
	Use:                   "import [flags] [TYPE] [NAME...] [PATH]",
	Short:                 "Import certificates",
	Long:                  importCmdDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Example: `
geneos tls import netprobe localhost /path/to/file.pem
geneos tls import /path/to/file.pem
`,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "false",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names, params := cmd.ParseTypeNamesParams(command)

		if len(params) > 1 {
			return fmt.Errorf("no more than one file path must be specified when importing a TLS certificate bundle")
		}

		var file string
		if len(params) == 0 {
			file = "-"
		} else {
			file = params[0]
		}

		if ct == nil && len(names) == 0 {
			log.Debug().Str("file", file).Msg("Importing signing bundle")
			return geneos.TLSImportBundle(file, importCmdPrivateKey)
		}

		log.Debug().Str("file", file).Msg("Importing instance bundle")

		if path.Ext(file) == ".pfx" || path.Ext(file) == ".p12" {
			if importCmdPassword.String() == "" {
				importCmdPassword, err = config.ReadPasswordInput(false, 0, "Password")
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to read password")
					return err
				}
			}
			pfxData, err := os.ReadFile(file)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to read PFX file")
			}
			key, c, chain, err := pkcs12.DecodeChain(pfxData, importCmdPassword.String())
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to decode PFX file")
			}
			pk, err := x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to marshal private key")
			}
			k := memguard.NewEnclave(pk)

			certBundle := certs.CertificateBundle{
				Leaf:      c,
				Key:       k,
				FullChain: chain,
			}
			instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, certBundle).Report(os.Stdout)
			return nil
		}

		certChain, err := config.ReadPEMBytes(file, "instance certificate(s)")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read instance certificate(s)")
		}
		key, err := config.ReadPEMBytes(importCmdPrivateKey, "instance key")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read instance key")
		}
		certBundle, err := certs.ParsePEM(certChain, key)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to decompose PEM")
		}
		if certBundle.Leaf == nil || certBundle.Key == nil {
			return fmt.Errorf("no leaf certificate and/or matching key found in instance bundle")
		}

		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, tlsWriteInstance, certBundle).Report(os.Stdout)
		return nil
	},
}

// tlsWriteInstance writes the certificate, key and chain to the instance.
// It returns a Response indicating success or failure.
func tlsWriteInstance(i geneos.Instance, params ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	if len(params) != 1 {
		resp.Err = geneos.ErrInvalidArgs
		return
	}

	tlsParam, ok := params[0].(certs.CertificateBundle)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[0] not a certs.CertificateBundle", geneos.ErrInvalidArgs)
		return
	}

	if resp.Err = instance.WriteCertificates(i, tlsParam.FullChain); resp.Err != nil {
		return
	}
	resp.Details = append(resp.Details, fmt.Sprintf("%s certificate written", i))

	if resp.Err = instance.WritePrivateKey(i, tlsParam.Key); resp.Err != nil {
		return
	}
	resp.Details = append(resp.Details, fmt.Sprintf("%s private key written", i))

	var updated bool
	updated, resp.Err = certs.UpdatedCACertsFile(i.Host(), geneos.CABundlePaths(i.Host()), tlsParam.Root)
	if resp.Err != nil {
		return
	}
	if updated {
		resp.Details = append(resp.Details, fmt.Sprintf("%s ca-bundle updated", i))
	}

	resp.Err = instance.SaveConfig(i)
	return
}
