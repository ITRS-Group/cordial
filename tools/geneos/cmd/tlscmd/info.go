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
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

// var infoCmdJSON, infoCmdIndent, infoCmdCSV, infoCmdToolkit bool
var infoCmdFormat string

func init() {
	tlsCmd.AddCommand(infoCmd)

	infoCmd.Flags().StringVarP(&infoCmdFormat, "format", "f", "table", "Output format (table, json, csv, toolkit)")

	// infoCmd.Flags().BoolVarP(&infoCmdJSON, "json", "j", false, "Output JSON")
	// infoCmd.Flags().BoolVarP(&infoCmdIndent, "pretty", "i", false, "Output indented JSON")
	// infoCmd.Flags().BoolVarP(&infoCmdCSV, "csv", "c", false, "Output CSV")
	// infoCmd.Flags().BoolVarP(&infoCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

}

//go:embed _docs/info.md
var infoCmdDescription string

var infoCmd = &cobra.Command{
	Use:          "info",
	Short:        "Info about certificates and keys",
	Long:         infoCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
	},
	RunE: func(command *cobra.Command, paths []string) (err error) {

		rp, err := reporter.NewReporter(infoCmdFormat, os.Stdout)
		if err != nil {
			return
		}

		// paths is a list of files to examine, pre-resolved by the
		// shell so we don't do any wildcard processing
		//
		// extensions are only checked for .pfx/.p12 files, others are
		// assumed to be PEM and may contain certificates, private keys
		// or both
		//
		// each PEM file can contain multiple entries and they are
		// listed in order

		for _, p := range paths {
			rep := reporter.Report{
				Name: fmt.Sprintf("tls-info-for file %q", p),
			}

			if path.Ext(p) == ".pfx" || path.Ext(p) == ".p12" {
				continue
			}

			r, err2 := os.Open(p)
			if err2 != nil {
				log.Error().Err(err2).Str("file", p).Msg("unable to open file")
				continue
			}

			contents, err2 := io.ReadAll(r)
			if err2 != nil {
				log.Error().Err(err2).Str("file", p).Msg("unable to read file")
				continue
			}

			_ = r.Close()

			var certs []*x509.Certificate
			var keys []*memguard.Enclave

			for {
				block, rest := pem.Decode(contents)
				if block == nil {
					break
				}
				switch block.Type {
				case "CERTIFICATE":
					var c *x509.Certificate
					c, err = x509.ParseCertificate(block.Bytes)
					if err != nil {
						return
					}
					log.Debug().Str("file", p).Str("cn", c.Subject.CommonName).Msg("found certificate")
					certs = append(certs, c)
				case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
					// save all private keys for later matching
					keys = append(keys, memguard.NewEnclave(block.Bytes))
				default:
					err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
					continue
				}
				contents = rest
			}

			// report

			fmt.Printf("contents of file %s:\n\n", p)

			for _, c := range certs {
				rep.Title = c.Subject.CommonName
				rp.Prepare(rep)

				// var matchingKey *memguard.Enclave
				// for _, k := range derkeys {
				// 	_, err = tls.X509KeyPair(
				// 		pem.EncodeToMemory(&pem.Block{
				// 			Type:  "CERTIFICATE",
				// 			Bytes: c.Raw,
				// 		}),
				// 		pem.EncodeToMemory(&pem.Block{
				// 			Type:  "PRIVATE KEY",
				// 			Bytes: k.Bytes(),
				// 		}),
				// 	)
				// 	if err == nil {
				// 		matchingKey = k
				// 		break
				// 	}
				// }

				lines := [][]string{
					{"Common Name", c.Subject.CommonName},
					{"Issuer", c.Issuer.CommonName},
					{"Serial", fmt.Sprintf("%X", c.SerialNumber)},
					{"Not Before", c.NotBefore.UTC().Format("2006-01-02 15:04:05 MST")},
					{"Not After", c.NotAfter.UTC().Format("2006-01-02 15:04:05 MST")},
					{"Is CA", fmt.Sprintf("%v", c.IsCA)},
					{"Basic Constraints", fmt.Sprintf("%v", c.BasicConstraintsValid)},
				}
				if c.KeyUsage > 0 {
					lines = append(lines, []string{"Key Usage", fmt.Sprintf("%s", strings.Join(keyUsageToString(c.KeyUsage), ", "))})
				}
				if len(c.ExtKeyUsage) > 0 {
					lines = append(lines, []string{"Ext Key Usage", fmt.Sprintf("%s", strings.Join(extKeyUsageToString(c.ExtKeyUsage), ", "))})
				}
				if len(c.DNSNames) > 0 {
					lines = append(lines, []string{"SAN DNS Names", fmt.Sprintf("%s", strings.Join(c.DNSNames, ", "))})
				}
				if len(c.EmailAddresses) > 0 {
					lines = append(lines, []string{"SAN Email Addresses", fmt.Sprintf("%s", strings.Join(c.EmailAddresses, ", "))})
				}
				if len(c.IPAddresses) > 0 {
					lines = append(lines, []string{"SAN IP Addresses", fmt.Sprintf("%s", strings.Join(infoMap(c.IPAddresses, func(ip net.IP) string { return ip.String() }), ", "))})
				}
				if len(c.URIs) > 0 {
					lines = append(lines, []string{"SAN URIs", fmt.Sprintf("%s", strings.Join(infoMap(c.URIs, func(uri *url.URL) string { return uri.String() }), ", "))})
				}
				lines = append(lines, [][]string{
					{"SHA1 Fingerprint", fmt.Sprintf("%X", sha1.Sum(c.Raw))},
					{"SHA256 Fingerprint", fmt.Sprintf("%X", sha256.Sum256(c.Raw))},
					// {"Private Key Present", fmt.Sprintf("%v", matchingKey != nil)},
				}...)

				rp.UpdateTable([]string{"Item", "Value"}, lines)
			}

			rp.Render()
		}

		return
	},
}

func infoMap[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func keyUsageToString(ku x509.KeyUsage) []string {
	usages := []string{}
	usageMap := map[x509.KeyUsage]string{
		x509.KeyUsageDigitalSignature:  "Digital Signature",
		x509.KeyUsageContentCommitment: "Content Commitment",
		x509.KeyUsageKeyEncipherment:   "Key Encipherment",
		x509.KeyUsageDataEncipherment:  "Data Encipherment",
		x509.KeyUsageKeyAgreement:      "Key Agreement",
	}
	for k, v := range usageMap {
		if ku&k != 0 {
			usages = append(usages, v)
		}
	}
	return usages
}

func extKeyUsageToString(eku []x509.ExtKeyUsage) []string {
	usages := []string{}
	usageMap := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageAny:             "Any",
		x509.ExtKeyUsageServerAuth:      "Server Auth",
		x509.ExtKeyUsageClientAuth:      "Client Auth",
		x509.ExtKeyUsageCodeSigning:     "Code Signing",
		x509.ExtKeyUsageEmailProtection: "Email Protection",
	}
	for k, v := range usageMap {
		if containsExtKeyUsage(eku, k) {
			usages = append(usages, v)
		}
	}
	return usages
}

func containsExtKeyUsage(usages []x509.ExtKeyUsage, usage x509.ExtKeyUsage) bool {
	for _, u := range usages {
		if u == usage {
			return true
		}
	}
	return false
}
