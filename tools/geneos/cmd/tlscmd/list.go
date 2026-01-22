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
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
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
	Valid      string        `json:"valid,omitempty"`
}

type listCertLongType struct {
	Type              string        `json:"type,omitempty"`
	Name              string        `json:"name,omitempty"`
	Host              string        `json:"host,omitempty"`
	Remaining         time.Duration `json:"remaining,omitempty"`
	Expires           time.Time     `json:"expires,omitempty"`
	CommonName        string        `json:"common_name,omitempty"`
	Valid             string        `json:"valid,omitempty"`
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

	listCmd.Flags().BoolVarP(&listCmdAll, "all", "a", false, "Show all certs, including root and signing certs")
	listCmd.Flags().BoolVarP(&listCmdLong, "long", "l", false, "Long output")

	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")
	listCmd.Flags().BoolVarP(&listCmdToolkit, "toolkit", "t", false, "Output Toolkit formatted CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var rootCert, signingCert *x509.Certificate
var rootKey, signingKey *memguard.Enclave
var rootCertFile, signingCertFile string

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
		rootCertFile, err = geneos.RootCertificatePath()
		if err != nil {
			return
		}
		rootCert, rootKey, err = geneos.ReadRootCertificateAndKey()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		// if rootKey == nil {
		// 	return fmt.Errorf("no root private key found")
		// }

		signingCertFile, err = geneos.SigningCertificatePath()
		if err != nil {
			return
		}
		signingCert, signingKey, err = geneos.ReadSigningCertificateAndKey()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		if signingKey == nil {
			return
		}

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
				results[cordial.ExecutableName()+" "+geneos.RootCABasename] = &responses.Response{
					Value: listCertType{
						cordial.ExecutableName(),
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCertWithKey(rootKey, rootCert),
					}}
			}
			if signingCert != nil {
				results[cordial.ExecutableName()+" "+geneos.SigningCertBasename] = &responses.Response{
					Value: listCertType{
						cordial.ExecutableName(),
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						signingCert.NotAfter,
						signingCert.Subject.CommonName,
						verifyCertWithKey(signingKey, signingCert),
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
					cordial.ExecutableName(),
					geneos.RootCABasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(rootKey, rootCert)),
				})
			}
			if signingCert != nil {
				listCSVWriter.Write([]string{
					geneos.SigningCertBasename + "@" + geneos.LOCALHOST,
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(signingCert.NotAfter).Seconds()),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(signingKey, signingCert)),
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
					cordial.ExecutableName(),
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					verifyCertWithKey(rootKey, rootCert),
				})
			}
			if signingCert != nil {
				prequel = append(prequel, []string{
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(signingCert.NotAfter).Seconds(), 'f', 0, 64),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					verifyCertWithKey(signingKey, signingCert),
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
					cordial.ExecutableName(),
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					verifyCertWithKey(rootKey, rootCert),
				})
			}
			if signingCert != nil {
				prequel = append(prequel, []string{
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(signingCert.NotAfter).Seconds(), 'f', 0, 64),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					verifyCertWithKey(signingKey, signingCert),
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
				results[cordial.ExecutableName()+" "+geneos.RootCABasename] = &responses.Response{
					Value: listCertLongType{
						cordial.ExecutableName(),
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
			if signingCert != nil {
				results[cordial.ExecutableName()+" "+geneos.SigningCertBasename] = &responses.Response{
					Value: listCertLongType{
						cordial.ExecutableName(),
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						signingCert.NotAfter,
						signingCert.Subject.CommonName,
						verifyCertWithKey(signingKey, signingCert),
						signingCertFile,
						"",
						rootCertFile,
						signingCert.Issuer.CommonName,
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
					cordial.ExecutableName(),
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
			if signingCert != nil {
				listCSVWriter.Write([]string{
					geneos.SigningCertBasename + "@" + geneos.LOCALHOST,
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(signingCert.NotAfter).Seconds()),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(signingKey, signingCert)),
					signingCertFile,
					"",
					rootCertFile,
					signingCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(signingCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(signingCert.Raw)),
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
					cordial.ExecutableName(),
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
			if signingCert != nil {
				listCSVWriter.Write([]string{
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(signingCert.NotAfter).Seconds()),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					fmt.Sprint(verifyCertWithKey(signingKey, signingCert)),
					signingCertFile,
					"",
					rootCertFile,
					signingCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(signingCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(signingCert.Raw)),
				})
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Report(listCSVWriter)

	default:
		var prequel [][]string
		if listCmdAll {
			if rootCert != nil {
				prequel = append(prequel, []string{
					cordial.ExecutableName(),
					geneos.RootCABasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(rootCert.NotAfter).Seconds(), 'f', 0, 64),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					verifyCertWithKey(rootKey, rootCert),
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
			if signingCert != nil {
				prequel = append(prequel, []string{
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(signingCert.NotAfter).Seconds(), 'f', 0, 64),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					verifyCertWithKey(signingKey, signingCert),
					signingCertFile,
					rootCertFile,
					"",
					signingCert.Issuer.CommonName,
					"",
					"",
					fmt.Sprintf("%X", sha1.Sum(signingCert.Raw)),
					fmt.Sprintf("%X", sha256.Sum256(signingCert.Raw)),
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
	valid := verifyCertWithKey(key, certChain...)
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
	valid := verifyCertWithKey(key, certChain...)
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
	valid := verifyCertWithKey(key, certChain...)
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
	valid := verifyCertWithKey(key, certChain...)
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
// certificate and returns a string that represents the result. "OK" if
// all is well.
func verifyCertWithKey(key *memguard.Enclave, certChain ...*x509.Certificate) string {
	if key == nil || key.Size() == 0 {
		return "NoKey"
	}
	if len(certChain) == 0 {
		return "NoCert"
	}
	roots, _ := certs.ReadCertificates(geneos.LOCAL, geneos.PathToCABundlePEM(geneos.LOCAL))
	certChain = append(certChain, roots...)
	if !certs.Verify(certChain...) {
		return "NotVerified"
	}
	match := certs.CheckKeyMatch(key, certChain[0])
	if !match {
		return "KeyMismatch"
	}
	return "OK"
}
