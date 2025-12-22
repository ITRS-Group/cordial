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
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var renewCmdDays int
var renewCmdNewKey, renewCmdPrepare, renewCmdRoll, renewCmdUnroll bool

const (
	newFileSuffix = "new"
	oldFileSuffix = "old"
)

func init() {
	tlsCmd.AddCommand(renewCmd)

	renewCmd.Flags().IntVarP(&renewCmdDays, "days", "D", 365, "Certificate duration in days")

	renewCmd.Flags().BoolVarP(&renewCmdNewKey, "new-key", "n", false, "Always generate a new private key for the renewed certificate")

	renewCmd.Flags().BoolVarP(&renewCmdPrepare, "prepare", "P", false, "Prepare renewal without overwriting existing certificates")
	renewCmd.Flags().BoolVarP(&renewCmdRoll, "roll", "R", false, "Roll previously prepared certificates and backup existing ones")
	renewCmd.Flags().BoolVarP(&renewCmdUnroll, "unroll", "U", false, "Unroll previously rolled certificates to restore backups")

	renewCmd.MarkFlagsMutuallyExclusive("prepare", "roll", "unroll")
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
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, renewInstanceCert).Write(os.Stdout)
	},
}

var chainUpdateMutex sync.Mutex

// renew an instance certificate, reuse private key if it exists
func renewInstanceCert(i geneos.Instance, _ ...any) (resp *instance.Response) {
	var err error

	resp = instance.NewResponse(i)

	confDir := config.AppConfigDir()
	if confDir == "" {
		resp.Err = config.ErrNoUserConfigDir
		return
	}

	switch {
	case renewCmdRoll:
		if err = instance.RollFiles(i, newFileSuffix, oldFileSuffix, "certificate", "privatekey"); err != nil {
			resp.Err = err
			return
		}

		resp.Completed = append(resp.Completed, "new certificate and key deployed, previous versions backed up")
		return

	case renewCmdUnroll:
		if err = instance.RollFiles(i, oldFileSuffix, newFileSuffix, "certificate", "privatekey"); err != nil {
			resp.Err = err
			return
		}

		resp.Completed = append(resp.Completed, "certificate unrolled, previous versions restored")
		return

	default:
		// check instance for existing cert, and do nothing if none
		cert, _, _, err := instance.ReadCert(i)
		if cert == nil && errors.Is(err, os.ErrNotExist) {
			return
		}

		signingCert, _, err := geneos.ReadSigningCert()
		resp.Err = err
		if resp.Err != nil {
			return
		}

		signingKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
		resp.Err = err
		if resp.Err != nil {
			return
		}

		hostname := i.Host().GetString("hostname")

		serial, err := rand.Prime(rand.Reader, 64)
		if err != nil {
			return
		}
		duration := 365 * 24 * time.Hour
		if renewCmdDays != 0 {
			duration = 24 * time.Hour * time.Duration(renewCmdDays)
		}
		expires := time.Now().Add(duration)
		template := x509.Certificate{
			SerialNumber: serial,
			Subject: pkix.Name{
				CommonName: fmt.Sprintf("geneos %s %s", i.Type(), i.Name()),
			},
			NotBefore:      time.Now().Add(-60 * time.Second),
			NotAfter:       expires,
			KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			MaxPathLenZero: true,
			DNSNames:       []string{hostname},
			// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
		}

		cert, key, err := certs.CreateCertificateAndKey(&template, signingCert, signingKey)
		resp.Err = err
		if resp.Err != nil {
			return
		}

		if renewCmdPrepare {
			// write new files but do not update instance config
			if err = certs.WriteCertificates(i.Host(), instance.ComponentFilepath(i, "pem", newFileSuffix), cert, signingCert); err != nil {
				return
			}

			if err = instance.WriteKey(i, key, newFileSuffix); err != nil {
				return
			}

			resp.Completed = append(resp.Completed, fmt.Sprintf("certificate prepared (expires %s)", expires.UTC().Format(time.RFC3339)))
			return
		}

		if resp.Err = certs.WriteCertificates(i.Host(), instance.ComponentFilepath(i, "pem"), cert, signingCert); resp.Err != nil {
			return
		}

		if resp.Err = instance.WriteKey(i, key); resp.Err != nil {
			return
		}

		// root cert optional to create instance specific chain file
		rootCert, _, _ := geneos.ReadRootCert()
		if rootCert == nil {
			i.Config().SetString("certchain", i.Host().PathTo("tls", geneos.ChainCertFile))
		} else {
			chainfile := instance.PathOf(i, "certchain")
			if chainfile == "" {
				chainfile = path.Join(i.Home(), "chain.pem")
				i.Config().SetString("certchain", chainfile, config.Replace("home"))
			}

			chainUpdateMutex.Lock()
			if updated, err := certs.UpdateCertChainFile(i.Host(), chainfile, signingCert, rootCert); err != nil {
				resp.Err = err
				chainUpdateMutex.Unlock()
				return
			} else if updated {
				resp.Lines = append(resp.Lines, fmt.Sprintf("%s certificate chain %q updated", i, chainfile))
			}
			chainUpdateMutex.Unlock()
		}

		if resp.Err = instance.SaveConfig(i); resp.Err != nil {
			return
		}

		resp.Lines = []string{
			fmt.Sprintf("certificate created for %s", i),
			fmt.Sprintf("            Expiry: %s", expires.UTC().Format(time.RFC3339)),
			fmt.Sprintf("  SHA1 Fingerprint: %X", sha1.Sum(cert.Raw)),
			fmt.Sprintf("SHA256 Fingerprint: %X", sha256.Sum256(cert.Raw)),
		}
	}
	return
}
