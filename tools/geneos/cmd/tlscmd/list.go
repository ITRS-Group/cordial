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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

type listCertType struct {
	Type       string        `json:"type,omitempty"`
	Name       string        `json:"name,omitempty"`
	Host       string        `json:"host,omitempty"`
	Remaining  time.Duration `json:"remaining,omitempty"`
	Expires    time.Time     `json:"expires,omitempty"`
	CommonName string        `json:"common_name,omitempty"`
	Valid      bool          `json:"valid,omitempty"`
}

type listCertLongType struct {
	Type              string        `json:"type,omitempty"`
	Name              string        `json:"name,omitempty"`
	Host              string        `json:"host,omitempty"`
	Remaining         time.Duration `json:"remaining,omitempty"`
	Expires           time.Time     `json:"expires,omitempty"`
	CommonName        string        `json:"common_name,omitempty"`
	Valid             bool          `json:"valid,omitempty"`
	Certificate       string        `json:"certificate,omitempty"`
	PrivateKey        string        `json:"privatekey,omitempty"`
	Chainfile         string        `json:"chainfile,omitempty"`
	Issuer            string        `json:"issuer,omitempty"`
	SubAltNames       []string      `json:"sans,omitempty"`
	IPs               []net.IP      `json:"ip_addresses,omitempty"`
	Fingerprint       string        `json:"fingerprint,omitempty"`
	FingerprintSHA256 string        `json:"fingerprint_sha256,omitempty"`
}

var listCmdAll, listCmdCSV, listCmdJSON, listCmdIndent, listCmdLong, listCmdToolkit bool
var listJSONEncoder *json.Encoder

func init() {
	tlsCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdAll, "all", "a", false, "Show all certs, including global and signing certs")
	listCmd.Flags().BoolVarP(&listCmdLong, "long", "l", false, "Long output")

	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")
	listCmd.Flags().BoolVarP(&listCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var rootCert, geneosCert *x509.Certificate
var rootKey, geneosKey *memguard.Enclave
var rootCertFile, geneosCertFile string

var listCmd = &cobra.Command{
	Use:          "list [flags] [TYPE] [NAME...]",
	Short:        "List certificates",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names, params := cmd.ParseTypeNamesParams(command)
		rootCert, rootCertFile, err = geneos.ReadRootCertificate()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Debug().Err(err).Msg("failed to read root cert")
			return
		}
		rootKey, _, _ = geneos.ReadRootPrivateKey()
		geneosCert, geneosCertFile, err = geneos.ReadSignerCertificate()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Debug().Err(err).Msg("failed to read signing cert")
			return
		}
		geneosKey, _, _ = geneos.ReadSignerPrivateKey()

		if listCmdLong {
			return listCertsLongCommand(ct, names, params)
		}
		return listCertsCommand(ct, names, params)
	},
}

func listCertsCommand(ct *geneos.Component, names []string, _ []string) (err error) {
	switch {
	case listCmdJSON, listCmdIndent:
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		results := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		if listCmdAll {
			if rootCert != nil {
				results["global "+geneos.RootCABasename] = &responses.Response{
					Value: listCertType{
						"global",
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCertWithKey(rootKey, rootCert),
					}}
			}
			if geneosCert != nil {
				results["global "+geneos.SigningCertBasename] = &responses.Response{
					Value: listCertType{
						"global",
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						geneosCert.NotAfter,
						geneosCert.Subject.CommonName,
						verifyCertWithKey(geneosKey, geneosCert),
					}}
			}
		}
		results.Report(os.Stdout, responses.IndentJSON(listCmdIndent))

	case listCmdToolkit:
		listCSVWriter := csv.NewWriter(os.Stdout)
		listCSVWriter.Write([]string{
			"ID",
			"type",
			"name",
			"host",
			"remaining",
			"expires",
			"commonName",
			"valid",
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					geneos.RootCABasename + "@" + geneos.LOCALHOST,
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(rootKey, rootCert)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					geneos.SigningCertBasename + "@" + geneos.LOCALHOST,
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(geneosKey, geneosCert)),
				})
			}
		}
		resp := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertToolkit)
		resp.Report(listCSVWriter)

		// add headlines
		fmt.Printf("<!>totalCerts,%d\n", len(resp))
		var expiringWeek, expiringMonth, invalid int
		for _, c := range resp {
			var t time.Duration
			if len(c.Rows) == 0 {
				continue
			}
			if t, err = time.ParseDuration(c.Rows[0][4] + "s"); err != nil {
				err = nil
				continue
			}
			if t < time.Hour*24*7*30 {
				expiringMonth++
			}
			if t < time.Hour*24*7 {
				expiringWeek++
			}
			if c.Rows[0][7] != "true" {
				invalid++
			}
		}
		fmt.Printf("<!>expiringLT7Days,%d\n", expiringWeek)
		fmt.Printf("<!>expiringLT30Days,%d\n", expiringMonth)
		fmt.Printf("<!>invalid,%d\n", invalid)

	case listCmdCSV:
		var prequel [][]string
		if listCmdAll {
			if rootCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(rootKey, rootCert)),
				})
			}
			if geneosCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(geneosCert.NotAfter).Seconds(), 'f', 0, 64),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(geneosKey, geneosCert)),
				})
			}
		}

		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Formatted(
			os.Stdout,
			"csv",
			[]string{
				"Type",
				"Name",
				"Host",
				"Remaining",
				"Expires",
				"CommonName",
				"Valid",
			},
			prequel,
			reporter.OrderByColumns(0, 1, 2),
		)

	default:
		var prequel [][]string
		if listCmdAll {
			if rootCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(rootKey, rootCert)),
				})
			}
			if geneosCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(geneosCert.NotAfter).Seconds(), 'f', 0, 64),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(geneosKey, geneosCert)),
				})
			}
		}

		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Formatted(
			os.Stdout,
			"column",
			[]string{
				"Type",
				"Name",
				"Host",
				"Remaining",
				"Expires",
				"CommonName",
				"Valid",
			},
			prequel,
		)
	}
	return
}

func listCertsLongCommand(ct *geneos.Component, names []string, params []string) (err error) {
	switch {
	case listCmdJSON, listCmdIndent:
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		results := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		if listCmdAll {
			if rootCert != nil {
				results["global "+geneos.RootCABasename] = &responses.Response{
					Value: listCertLongType{
						"global",
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCertWithKey(rootKey, rootCert),
						rootCertFile,
						"",
						rootCertFile,
						rootCert.Issuer.CommonName,
						nil,
						nil,
						fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
						fmt.Sprintf("%X", sha256.Sum256(rootCert.Raw)),
					}}
			}
			if geneosCert != nil {
				results["global "+geneos.SigningCertBasename] = &responses.Response{
					Value: listCertLongType{
						"global",
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						geneosCert.NotAfter,
						geneosCert.Subject.CommonName,
						verifyCertWithKey(geneosKey, geneosCert),
						geneosCertFile,
						"",
						rootCertFile,
						geneosCert.Issuer.CommonName,
						nil,
						nil,
						fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
						fmt.Sprintf("%X", sha256.Sum256(rootCert.Raw)),
					}}
			}
		}
		results.Report(os.Stdout, responses.IndentJSON(listCmdIndent))

	case listCmdToolkit:
		listCSVWriter := csv.NewWriter(os.Stdout)
		listCSVWriter.Write([]string{
			"ID",
			"type",
			"name",
			"host",
			"remaining",
			"expires",
			"commonName",
			"valid",
			"certificate",
			"privateKey",
			"chain",
			"issuer",
			"subjAltNames",
			"IPs",
			"fingerprint",
			"fingerprintSHA256",
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					geneos.RootCABasename + "@" + geneos.LOCALHOST,
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(rootKey, rootCert)),
					rootCertFile,
					"",
					rootCertFile,
					rootCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					geneos.SigningCertBasename + "@" + geneos.LOCALHOST,
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(geneosKey, geneosCert)),
					geneosCertFile,
					"",
					rootCertFile,
					geneosCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(geneosCert.Raw)),
				})
			}
		}
		resp := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertToolkit)
		resp.Report(listCSVWriter)

		// add headlines
		// add headlines
		fmt.Printf("<!>totalCerts,%d\n", len(resp))
		var expiringWeek, expiringMonth, invalid int
		for _, c := range resp {
			var t time.Duration
			if len(c.Rows) == 0 {
				continue
			}
			if t, err = time.ParseDuration(c.Rows[0][4] + "s"); err != nil {
				err = nil
				continue
			}
			if t < time.Hour*24*7*30 {
				expiringMonth++
			}
			if t < time.Hour*24*7 {
				expiringWeek++
			}
			if c.Rows[0][7] != "true" {
				invalid++
			}
		}
		fmt.Printf("<!>expiringLT7Days,%d\n", expiringWeek)
		fmt.Printf("<!>expiringLT30Days,%d\n", expiringMonth)
		fmt.Printf("<!>invalid,%d\n", invalid)
	case listCmdCSV:
		listCSVWriter := csv.NewWriter(os.Stdout)
		listCSVWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
			"Valid",
			"CertificateFile",
			"PrivateKeyFile",
			"ChainFile",
			"Issuer",
			"SubjAltNames",
			"IPs",
			"Fingerprint",
			"FingerprintSHA256",
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(rootKey, rootCert)),
					rootCertFile,
					"",
					rootCertFile,
					rootCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(geneosKey, geneosCert)),
					geneosCertFile,
					"",
					rootCertFile,
					geneosCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(geneosCert.Raw)),
				})
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Report(listCSVWriter)

	default:
		var prequel [][]string
		if listCmdAll {
			if rootCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(rootKey, rootCert)),
					rootCertFile,
					rootCertFile,
					"",
					rootCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				prequel = append(prequel, []string{
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(geneosCert.NotAfter).Seconds(), 'f', 0, 64),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					strconv.FormatBool(verifyCertWithKey(geneosKey, geneosCert)),
					geneosCertFile,
					rootCertFile,
					"",
					geneosCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(geneosCert.Raw)),
				})
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Formatted(
			os.Stdout,
			"column",
			[]string{
				"Type",
				"Name",
				"Host",
				"Remaining",
				"Expires",
				"CommonName",
				"Valid",
				"CertificateFile",
				"PrivateKeyFile",
				"ChainFile",
				"Issuer",
				"SubjAltNames",
				"IPs",
				"Fingerprint",
				"FingerprintSHA256",
			},
			prequel,
		)
	}
	return
}

func listCmdInstanceCert(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		return
	}
	cert := certChain[0]
	key, _ := instance.ReadPrivateKey(i)
	valid := certs.IsValidLeafCert(cert) && verifyCertWithKey(key, certChain...)
	chainfile := i.Config().GetString("chainfile")

	if err != nil && errors.Is(err, os.ErrNotExist) {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return
	}

	if cert == nil && err != nil {
		return
	}

	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())
	cols := []string{i.Type().String(), i.Name(), i.Host().String(), until, expires.Format(time.RFC3339), cert.Subject.CommonName, fmt.Sprint(valid)}
	if listCmdLong {
		cols = append(cols, i.Config().GetString("certificate"))
		cols = append(cols, i.Config().GetString("privatekey"))
		cols = append(cols, chainfile)
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
		cols = append(cols, fmt.Sprintf("%X", sha256.Sum256(cert.Raw)))
	}

	resp.Rows = append(resp.Rows, cols)
	return
}

func listCmdInstanceCertCSV(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		return
	}
	cert := certChain[0]
	key, _ := instance.ReadPrivateKey(i)
	if err != nil {
		return
	}
	valid := certs.IsValidLeafCert(cert) && verifyCertWithKey(key, certChain...)
	chainfile := i.Config().GetString("chainfile")
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return
	}

	if cert == nil && err != nil {
		return
	}

	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())
	cols := []string{i.Type().String(), i.Name(), i.Host().String(), until, expires.Format(time.RFC3339), cert.Subject.CommonName, fmt.Sprint(valid)}
	if listCmdLong {
		cols = append(cols, i.Config().GetString("certificate"))
		cols = append(cols, i.Config().GetString("privatekey"))
		cols = append(cols, chainfile)
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
		cols = append(cols, fmt.Sprintf("%X", sha256.Sum256(cert.Raw)))
	}

	resp.Rows = append(resp.Rows, cols)
	return
}

func listCmdInstanceCertToolkit(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		return
	}
	cert := certChain[0]
	key, _ := instance.ReadPrivateKey(i)
	valid := certs.IsValidLeafCert(cert) && verifyCertWithKey(key, certChain...)
	chainfile := i.Config().GetString("chainfile")
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return
	}

	if cert == nil && err != nil {
		return
	}

	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())

	cols := []string{
		instance.IDString(i),
		i.Type().String(),
		i.Name(),
		i.Host().String(),
		until,
		expires.Format(time.RFC3339),
		cert.Subject.CommonName,
		fmt.Sprint(valid),
	}
	if listCmdLong {
		cols = append(cols,
			i.Config().GetString("certificate"),
			i.Config().GetString("privatekey"),
			chainfile,
			cert.Issuer.CommonName,
			strings.Join(cert.DNSNames, " "),
		)
		if len(cert.IPAddresses) > 0 {
			ips := []string{}
			for _, ip := range cert.IPAddresses {
				ips = append(ips, ip.String())
			}
			cols = append(cols, strings.Join(ips, " "))
		} else {
			cols = append(cols, "")
		}
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
		cols = append(cols, fmt.Sprintf("%X", sha256.Sum256(cert.Raw)))
	}

	resp.Rows = append(resp.Rows, cols)
	return
}

func listCmdInstanceCertJSON(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	certChain, err := instance.ReadCertificates(i)
	if err != nil {
		return
	}
	cert := certChain[0]
	key, _ := instance.ReadPrivateKey(i)
	valid := certs.IsValidLeafCert(cert) && verifyCertWithKey(key, certChain...)
	chainfile := i.Config().GetString("chainfile")
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return
	}

	if cert == nil && err != nil {
		return
	}

	if listCmdLong {
		resp.Value = listCertLongType{
			i.Type().String(),
			i.Name(),
			i.Host().String(),
			time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter,
			cert.Subject.CommonName,
			valid,
			i.Config().GetString("certificate"),
			i.Config().GetString("privatekey"),
			chainfile,
			cert.Issuer.CommonName,
			cert.DNSNames,
			cert.IPAddresses,
			fmt.Sprintf("%X", sha1.Sum(cert.Raw)),
			fmt.Sprintf("%X", sha256.Sum256(cert.Raw)),
		}
		return
	}
	resp.Value = listCertType{
		i.Type().String(),
		i.Name(),
		i.Host().String(),
		time.Duration(time.Until(cert.NotAfter).Seconds()),
		cert.NotAfter,
		cert.Subject.CommonName,
		valid,
	}
	return
}

// verifyCertWithKey checks the certChain after appending the local CA
// bundle and checks if the private key provided matches the leaf
// certificate
func verifyCertWithKey(key *memguard.Enclave, certChain ...*x509.Certificate) bool {
	if key == nil || len(certChain) == 0 {
		return false
	}
	roots, _ := certs.ReadCertificates(geneos.LOCAL, geneos.PathToCABundlePEM(geneos.LOCAL))
	log.Debug().Msgf("loaded %d root CA certificates from %s", len(roots), geneos.PathToCABundlePEM(geneos.LOCAL))
	certChain = append(certChain, roots...)
	if !certs.Verify(certChain...) {
		return false
	}
	return certs.CheckKeyMatch(key, certChain[0])
}
