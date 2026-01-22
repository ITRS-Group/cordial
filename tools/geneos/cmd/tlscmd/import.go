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
	_ "embed"
	"fmt"
	"os"
	"path"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var importCmdPrivateKey string
var importCmdPassword = &config.Plaintext{}

func init() {
	tlsCmd.AddCommand(importCmd)

	importCmd.Flags().VarP(importCmdPassword, "password", "p",
		`Plaintext password for PFX/PKCS#12 file decryption.
You will be prompted if not supplied as an argument.
PFX/PKCS#12 files are identified by the .pfx or .p12
file extension and only supported for instance bundles`,
	)

	importCmd.Flags().StringVarP(&importCmdPrivateKey, "key", "k", "", "Private key `file` for certificate, PEM format only")

	importCmd.Flags().SortFlags = false
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
		cmd.CmdRequireHome:   "false",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		var certBundle *certs.CertificateBundle

		ct, names, params := cmd.ParseTypeNamesParams(command)

		// move any "-" names to params
		if slices.Contains(names, "-") {
			params = append(params, "-")
			names = slices.DeleteFunc(names, func(s string) bool { return s == "-" })
		}

		if len(params) > 1 {
			return fmt.Errorf("no more than one file path must be specified when importing a TLS certificate bundle")
		}

		file := "-"
		if len(params) > 0 {
			file = params[0]
		}

		if ct == nil && len(names) == 0 {
			return geneos.TLSImportBundle(file, importCmdPrivateKey)
		}

		if geneos.LocalRoot() == "" && len(geneos.RemoteHosts(false)) == 0 {
			command.SetUsageTemplate(" ")
			return cmd.GeneosUnsetError
		}

		if path.Ext(file) == ".pfx" || path.Ext(file) == ".p12" {
			if importCmdPassword.String() == "" {
				importCmdPassword, err = config.ReadPasswordInput(false, 0, "Password")
				if err != nil {
					return err
				}
			}
			certBundle, err = certs.P12ToCertBundle(file, importCmdPassword)
			if err != nil {
				return err
			}

		} else {
			certChain, err := config.ReadPEMBytes(file, "instance certificate(s)")
			if err != nil {
				return err
			}
			key, err := config.ReadPEMBytes(importCmdPrivateKey, "instance key")
			if err != nil {
				return err
			}
			certBundle, err = certs.ParsePEM(certChain, key)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to decompose PEM")
			}
		}

		if !certBundle.Valid {
			return fmt.Errorf("certificate bundle is not valid, check trust chain and key match")
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

	tlsParam, ok := params[0].(*certs.CertificateBundle)
	if !ok {
		resp.Err = fmt.Errorf("%w: params[0] not a certs.CertificateBundle", geneos.ErrInvalidArgs)
		return
	}

	if resp.Err = instance.WriteCertificateAndKey(i, tlsParam.Key, tlsParam.FullChain...); resp.Err != nil {
		return
	}
	resp.Details = append(resp.Details, fmt.Sprintf("%s certificate, trust chain and key written", i))

	if err := instance.SaveConfig(i); err != nil {
		return
	}

	var updated bool
	updated, resp.Err = certs.UpdateCACertsFiles(i.Host(), geneos.PathToCABundle(i.Host()), tlsParam.Root)
	if resp.Err != nil {
		return
	}
	if updated {
		resp.Details = append(resp.Details, fmt.Sprintf("%s ca-bundle updated", i))
	}

	resp.Err = instance.SaveConfig(i)
	return
}
