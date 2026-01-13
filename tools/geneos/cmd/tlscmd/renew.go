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
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var renewCmdDays int
var renewCmdPrepare, renewCmdRoll, renewCmdUnroll, renewCmdSigner bool

const (
	newFileSuffix = "new"
	oldFileSuffix = "old"
)

func init() {
	tlsCmd.AddCommand(renewCmd)

	renewCmd.Flags().BoolVar(&renewCmdSigner, "signer", false, "Renew the signing certificate instead of instance certificates")

	renewCmd.Flags().IntVarP(&renewCmdDays, "days", "D", 365, "Instance certificate duration in days. (No effect for signer)")

	renewCmd.Flags().BoolVarP(&renewCmdPrepare, "prepare", "P", false, "Prepare renewal without overwriting existing certificates")
	renewCmd.Flags().BoolVarP(&renewCmdRoll, "roll", "R", false, "Roll previously prepared certificates and backup existing ones")
	renewCmd.Flags().BoolVarP(&renewCmdUnroll, "unroll", "U", false, "Unroll previously rolled certificates to restore backups")

	renewCmd.MarkFlagsMutuallyExclusive("prepare", "roll", "unroll")

	renewCmd.Flags().SortFlags = false
}

//go:embed _docs/renew.md
var renewCmdDescription string

var renewCmd = &cobra.Command{
	Use:          "renew [TYPE] [NAME...]",
	Short:        "Renew instance certificates",
	Long:         renewCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		if renewCmdSigner {
			// renew signing certificate
			confDir := config.AppConfigDir()
			if confDir == "" {
				return config.ErrNoUserConfigDir
			}
			if signer, err := certs.WriteNewSignerCert(
				path.Join(confDir, geneos.SigningCertBasename),
				path.Join(confDir, geneos.RootCABasename),
				cordial.ExecutableName()+" intermediate certificate",
			); err != nil {
				return err
			} else {
				fmt.Printf("signing certificate renewed. expires %s\n", signer.NotAfter.Local().Format(time.RFC3339))
			}
			return
		}

		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, renewInstanceCert).Report(os.Stdout)
		return
	},
}

var chainUpdateMutex sync.Mutex

// renew an instance certificate, reuse private key if it exists
func renewInstanceCert(i geneos.Instance, _ ...any) (resp *responses.Response) {
	var err error

	cf := i.Config()

	confDir := config.AppConfigDir()
	if confDir == "" {
		resp = responses.NewResponse(i)
		resp.Err = config.ErrNoUserConfigDir
		return
	}

	// migrate the TLS config, regardless of roll/unroll, at this point
	resp = migrateInstance(i)

	switch {
	case renewCmdRoll:
		if err = instance.RollFiles(i, newFileSuffix, oldFileSuffix, cf.Join("tls", "certificate"), cf.Join("tls", "privatekey")); err != nil {
			resp.Err = err
			return
		}

		resp.Completed = append(resp.Completed, "new certificate and key deployed, previous versions backed up")
		return

	case renewCmdUnroll:
		if err = instance.RollFiles(i, oldFileSuffix, newFileSuffix, cf.Join("tls", "certificate"), cf.Join("tls", "privatekey")); err != nil {
			resp.Err = err
			return
		}

		resp.Completed = append(resp.Completed, "certificate unrolled, previous versions restored")
		return

	default:
		// check instance for existing cert, and do nothing if none
		cert, err := instance.ReadCertificate(i)
		if cert == nil && errors.Is(err, os.ErrNotExist) {
			return
		}

		signingCert, _, err := geneos.ReadSignerCertificate()
		resp.Err = err
		if resp.Err != nil {
			return
		}

		signingKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
		resp.Err = err
		if resp.Err != nil {
			return
		}

		template := certs.Template("geneos "+i.Type().String()+" "+i.Name(),
			certs.DNSNames(i.Host().GetString("hostname")),
			certs.Days(renewCmdDays),
		)
		expires := template.NotAfter

		cert, key, err := certs.CreateCertificateAndKey(template, signingCert, signingKey)
		resp.Err = err
		if resp.Err != nil {
			return
		}

		if renewCmdPrepare {
			// write new files but do not update instance config
			if err = certs.WriteCertificates(i.Host(), instance.ComponentFilepath(i, "pem", newFileSuffix), cert, signingCert); err != nil {
				return
			}

			if err = certs.WritePrivateKey(i.Host(), instance.ComponentFilepath(i, "key", newFileSuffix), key); err != nil {
				return
			}

			resp.Completed = append(resp.Completed, fmt.Sprintf("certificate prepared (expires %s)", expires.UTC().Format(time.RFC3339)))
			return
		}

		if resp.Err = instance.WriteCertificates(i, []*x509.Certificate{cert, signingCert}); resp.Err != nil {
			return
		}

		if resp.Err = instance.WritePrivateKey(i, key); resp.Err != nil {
			return
		}

		// TODO: migrate other settings, create trusted-roots etc.

		if resp.Err = instance.SaveConfig(i); resp.Err != nil {
			return
		}

		resp.Details = []string{
			fmt.Sprintf("certificate created for %s", i),
			fmt.Sprintf("            Expiry: %s", expires.UTC().Format(time.RFC3339)),
			fmt.Sprintf("  SHA1 Fingerprint: %X", sha1.Sum(cert.Raw)),
			fmt.Sprintf("SHA256 Fingerprint: %X", sha256.Sum256(cert.Raw)),
		}
	}
	return
}
