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
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// var infoCmdJSON, infoCmdIndent, infoCmdCSV, infoCmdToolkit bool
var infoCmdFormat string
var infoCmdLong bool
var infoCmdPassword *config.Plaintext

func init() {
	tlsCmd.AddCommand(infoCmd)

	infoCmdPassword = &config.Plaintext{}

	infoCmd.Flags().BoolVarP(&infoCmdLong, "long", "l", false, "Output long format (more columns)")

	infoCmd.Flags().StringVarP(&infoCmdFormat, "format", "f", "column", "Output format (column, table, csv, toolkit)")
	infoCmd.Flags().VarP(infoCmdPassword, "password", "p", "Password for PFX file(s), if needed. Defaults to prompting for each file. Use -p \"\" to specify empty password.")
}

//go:embed _docs/info.md
var infoCmdDescription string

type certContents struct {
	Alias        []string
	Certificates []*x509.Certificate
	PrivateKeys  []*memguard.Enclave
}

type certInfo struct {
	Path     string
	Contents certContents
}

var columns = []string{
	"FileAndIndex",
	"CommonName",
	"IssuerCommonName",
	"NotAfter",
	"IsCA",
	"ExtKeyUsage",
	"SANDNSNames",
	"SANIPAddresses",
}

var columnsLong = []string{
	"PathAndIndex",
	"CommonName",
	"IssuerCommonName",
	"Serial",
	"SKID",
	"AKID",
	"NotBefore",
	"NotAfter",
	"IsCA",
	"KeyUsage",
	"ExtKeyUsage",
	"SANDNSNames",
	"SANEmailAddresses",
	"SANIPAddresses",
	"SANURIs",
	"SHA1Fingerprint",
	"SHA256Fingerprint",
	// "Private Key Present",
}

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
		// gather cert info
		certInfos := make([]certInfo, len(paths))

		certInfos, err = readFiles(paths)
		if err != nil {
			return
		}

		// prepare reporter
		rp, err := reporter.NewReporter(infoCmdFormat, os.Stdout)
		if err != nil {
			return
		}

		// output info
		// report
		rp.Prepare(reporter.Report{
			Title: "Certificate Information",
		})

		lines := [][]string{}

		for i := range certInfos {
			for n, c := range certInfos[i].Contents.Certificates {
				var fileandindex string
				if len(certInfos[i].Contents.Alias) > n && certInfos[i].Contents.Alias[n] != "" {
					fileandindex = fmt.Sprintf("%s:%s", path.Base(certInfos[i].Path), certInfos[i].Contents.Alias[n])
				} else {
					fileandindex = fmt.Sprintf("%s:%d", path.Base(certInfos[i].Path), n)
				}

				if !infoCmdLong {
					lines = append(lines, []string{
						fileandindex,
						c.Subject.CommonName,
						c.Issuer.CommonName,
						c.NotAfter.UTC().Format(time.RFC3339),
						strconv.FormatBool(c.IsCA),
						fmt.Sprintf("%v", extKeyUsageToString(c.ExtKeyUsage)),
						fmt.Sprintf("%v", c.DNSNames),
						fmt.Sprintf("%v", infoMap(c.IPAddresses, func(ip net.IP) string { return ip.String() })),
					})
				} else {
					lines = append(lines, []string{
						fileandindex,
						c.Subject.CommonName,
						c.Issuer.CommonName,
						fmt.Sprintf("%X", c.SerialNumber),
						fmt.Sprintf("%X", c.SubjectKeyId),
						fmt.Sprintf("%X", c.AuthorityKeyId),
						c.NotBefore.UTC().Format(time.RFC3339),
						c.NotAfter.UTC().Format(time.RFC3339),
						strconv.FormatBool(c.IsCA),
						fmt.Sprintf("%v", keyUsageToString(c.KeyUsage)),
						fmt.Sprintf("%v", extKeyUsageToString(c.ExtKeyUsage)),
						fmt.Sprintf("%v", c.DNSNames),
						fmt.Sprintf("%v", c.EmailAddresses),
						fmt.Sprintf("%v", infoMap(c.IPAddresses, func(ip net.IP) string { return ip.String() })),
						fmt.Sprintf("%v", infoMap(c.URIs, func(uri *url.URL) string { return uri.String() })),
						fmt.Sprintf("%X", sha1.Sum(c.Raw)),
						fmt.Sprintf("%X", sha256.Sum256(c.Raw)),
					})
				}
			}
		}

		if infoCmdLong {
			rp.UpdateTable(columnsLong, lines)
		} else {
			rp.UpdateTable(columns, lines)
		}
		rp.Render()

		return
	},
}

func readFiles(paths []string) (certInfos []certInfo, err error) {
	certInfos = make([]certInfo, len(paths))

	// paths is a list of files to examine, pre-resolved by the
	// shell so we don't do any wildcard processing
	//
	// extensions are only checked for .pfx/.p12 files, others are
	// assumed to be PEM and may contain certificates, private keys
	// or both
	//
	// each PEM file can contain multiple entries and they are
	// listed in order
	for i, p := range paths {
		certInfos[i].Path, err = filepath.Abs(p)
		if err != nil {
			log.Error().Err(err).Str("file", p).Msg("unable to get absolute path")
			continue
		}
		certInfos[i].Contents = certContents{}

		contents, err2 := os.ReadFile(p)
		if err2 != nil {
			log.Error().Err(err2).Str("file", p).Msg("unable to read file")
			continue
		}

		ext := strings.ToLower(path.Ext(p))

		if path.Base(p) == "cacerts" {
			k, err := certs.ReadKeystore(geneos.LOCAL, p, config.NewPlaintext([]byte("changeit")))
			if err != nil {
				log.Error().Err(err).Str("file", p).Msg("unable to read Java keystore")
				continue
			}
			for _, alias := range k.Aliases() {
				entry, err := k.GetTrustedCertificateEntry(alias)
				if err != nil {
					log.Error().Err(err).Str("alias", alias).Msgf("unable to get certificate entry %q from Java keystore", alias)
					continue
				}
				cert, err := x509.ParseCertificate(entry.Certificate.Content)
				if err != nil {
					log.Error().Err(err).Str("alias", alias).Msg("unable to parse certificate from Java keystore")
					continue
				}
				log.Debug().Str("file", p).Str("alias", alias).Str("cn", cert.Subject.CommonName).Msg("found certificate in Java keystore")
				certInfos[i].Contents.Alias = append(certInfos[i].Contents.Alias, alias)
				certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, cert)
			}
			continue
		}

		if ext == ".db" {
			if infoCmdPassword.String() == "" {
				infoCmdPassword, err = config.ReadPasswordInput(false, 0, "Password (for file "+p+")")
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to read password")
					// return err
				}
			}
			k, err := certs.ReadKeystore(geneos.LOCAL, p, infoCmdPassword)
			if err != nil {
				log.Error().Err(err).Str("file", p).Msg("unable to read Java keystore")
				continue
			}
			for _, alias := range k.Aliases() {
				switch {
				case k.IsPrivateKeyEntry(alias):
					chain, err := k.GetPrivateKeyEntryCertificateChain(alias)
					if err != nil {
						log.Error().Err(err).Str("alias", alias).Msgf("unable to get private certificate chain %q from Java keystore", alias)
						continue
					}
					for n, cert := range chain {
						parsedCert, err := x509.ParseCertificate(cert.Content)
						if err != nil {
							log.Error().Err(err).Str("alias", alias).Int("cert", n).Msg("unable to parse certificate from Java keystore")
							continue
						}
						certInfos[i].Contents.Alias = append(certInfos[i].Contents.Alias, alias+"["+strconv.Itoa(n)+"]")
						certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, parsedCert)
					}
				case k.IsTrustedCertificateEntry(alias):
					entry, err := k.GetTrustedCertificateEntry(alias)
					if err != nil {
						log.Error().Err(err).Str("alias", alias).Msgf("unable to get CA certificate entry %q from Java keystore", alias)
						continue
					}
					cert, err := x509.ParseCertificate(entry.Certificate.Content)
					if err != nil {
						log.Error().Err(err).Str("alias", alias).Msg("unable to parse certificate from Java keystore")
						continue
					}
					log.Debug().Str("file", p).Str("alias", alias).Str("cn", cert.Subject.CommonName).Msg("found certificate in Java keystore")
					if slices.Contains(certInfos[i].Contents.Alias, alias) {
						log.Debug().Str("file", p).Str("alias", alias).Msg("duplicate certificate alias in Java keystore, skipping")
						continue
					}
					certInfos[i].Contents.Alias = append(certInfos[i].Contents.Alias, alias)
					certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, cert)
				default:
					continue
				}
			}
			continue
		}

		if ext == ".pfx" || ext == ".p12" {
			if infoCmdPassword.String() == "" {
				infoCmdPassword, err = config.ReadPasswordInput(false, 0, "Password (for file "+p+")")
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to read password")
					// return err
				}
			}

			key, c, chain, err := pkcs12.DecodeChain(contents, infoCmdPassword.String())
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to decode PFX file")
				// return err
			}
			certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, c)
			certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, chain...)

			pk, err := x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to marshal private key")
				// return err
			}
			certInfos[i].Contents.PrivateKeys = append(certInfos[i].Contents.PrivateKeys, memguard.NewEnclave(pk))
			continue
		}

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
				certInfos[i].Contents.Certificates = append(certInfos[i].Contents.Certificates, c)
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				certInfos[i].Contents.PrivateKeys = append(certInfos[i].Contents.PrivateKeys, memguard.NewEnclave(block.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
				continue
			}
			contents = rest
		}
	}

	return
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
		x509.KeyUsageDigitalSignature:  "DigitalSignature",
		x509.KeyUsageContentCommitment: "ContentCommitment",
		x509.KeyUsageKeyEncipherment:   "KeyEncipherment",
		x509.KeyUsageDataEncipherment:  "DataEncipherment",
		x509.KeyUsageKeyAgreement:      "KeyAgreement",
		x509.KeyUsageCertSign:          "CertSign",
		x509.KeyUsageCRLSign:           "CRLSign",
		x509.KeyUsageEncipherOnly:      "EncipherOnly",
		x509.KeyUsageDecipherOnly:      "DecipherOnly",
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
		x509.ExtKeyUsageAny:                            "Any",
		x509.ExtKeyUsageServerAuth:                     "ServerAuth",
		x509.ExtKeyUsageClientAuth:                     "ClientAuth",
		x509.ExtKeyUsageCodeSigning:                    "CodeSigning",
		x509.ExtKeyUsageEmailProtection:                "EmailProtection",
		x509.ExtKeyUsageIPSECEndSystem:                 "IPSECEndSystem",
		x509.ExtKeyUsageIPSECTunnel:                    "IPSECTunnel",
		x509.ExtKeyUsageIPSECUser:                      "IPSECUser",
		x509.ExtKeyUsageTimeStamping:                   "TimeStamping",
		x509.ExtKeyUsageOCSPSigning:                    "OCSPSigning",
		x509.ExtKeyUsageMicrosoftServerGatedCrypto:     "MicrosoftServerGatedCrypto",
		x509.ExtKeyUsageNetscapeServerGatedCrypto:      "NetscapeServerGatedCrypto",
		x509.ExtKeyUsageMicrosoftCommercialCodeSigning: "MicrosoftCommercialCodeSigning",
		x509.ExtKeyUsageMicrosoftKernelCodeSigning:     "MicrosoftKernelCodeSigning",
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
