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
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

type listCertType struct {
	Type       string        `json:"type,omitempty"`
	Name       string        `json:"name,omitempty"`
	Host       string        `json:"host,omitempty"`
	Remaining  time.Duration `json:"remaining,omitempty"`
	Expires    time.Time     `json:"expires"`
	CommonName string        `json:"common_name,omitempty"`
	Valid      string        `json:"valid,omitempty"`
}

type listCertLongType struct {
	Type              string        `json:"type,omitempty"`
	Name              string        `json:"name,omitempty"`
	Host              string        `json:"host,omitempty"`
	Remaining         time.Duration `json:"remaining,omitempty"`
	Expires           time.Time     `json:"expires"`
	CommonName        string        `json:"common_name,omitempty"`
	Valid             string        `json:"valid,omitempty"`
	Certificate       string        `json:"certificate,omitempty"`
	PrivateKey        string        `json:"privatekey,omitempty"`
	TrustChain        string        `json:"trustchain,omitempty"`
	Issuer            string        `json:"issuer,omitempty"`
	SubAltNames       []string      `json:"sans,omitempty"`
	IPs               []net.IP      `json:"ip_addresses,omitempty"`
	Fingerprint       string        `json:"fingerprint,omitempty"`
	FingerprintSHA256 string        `json:"fingerprint_sha256,omitempty"`
}

// limits for the toolkit headlines
const (
	expiringWeekLimit  = 7 * 24 * time.Hour
	expiringMonthLimit = 30 * 24 * time.Hour
)

var listCmdAll, listCmdCSV, listCmdJSON, listCmdIndent, listCmdLong, listCmdToolkit bool
var listJSONEncoder *json.Encoder

func init() {
	tlsCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdAll, "all", "a", false, "Show all certs, including root and signing certs")
	listCmd.Flags().BoolVarP(&listCmdLong, "long", "l", false, "Long output")

	cmd.AddOutputFormatFlags(listCmd.Flags(), &listCmdJSON, &listCmdIndent, &listCmdCSV, &listCmdToolkit)

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var rootCert, signingCert *x509.Certificate
var rootKey, signingKey certs.PrivateKey
var rootCertFile, signingCertFile string

var listCmd = &cobra.Command{
	Use:          "list [flags] [TYPE] [NAME...]",
	Short:        "List certificates",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:               "true",
		cmd.CmdRequireHome:          "true",
		cmd.CmdWildcardNames:        "true",
		cmd.CmdAllowRoot:            "true",
		cmd.CmdNonInstanceArgsError: "true",
		cmd.CmdAuditCommand:         "never",
		cmd.CmdAuditActions:         "never",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		log = log.With("command", "tls list")
		localCAs = sync.OnceValue(readLocalCAs)
		ct, names, params, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}
		rootCertFile, err = geneos.RootCertificatePath()
		if err != nil {
			return
		}
		// TODO: this may not be an error
		rootCert, rootKey, err = geneos.ReadRootCertificateAndKey()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}

		signingCertFile, err = geneos.SigningCertificatePath()
		if err != nil {
			return
		}

		// TODO: this may not be an error
		signingCert, signingKey, err = geneos.ReadSigningCertificateAndKey()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}

		if listCmdLong {
			return listCertsLongCommand(command, ct, names, params)
		}
		return listCertsCommand(command, ct, names, params)
	},
}

func listCertsCommand(command *cobra.Command, ct *geneos.Component, names []string, _ []string) (err error) {
	switch {
	case listCmdJSON, listCmdIndent:
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		results := instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		if listCmdAll {
			if rootCert != nil {
				results[cordial.ExecutableName()+" "+geneos.RootCABasename] = &responses.General{
					Value: listCertType{
						cordial.ExecutableName(),
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Until(rootCert.NotAfter).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCertWithKey(rootKey, rootCert),
					}}
			}
			if signingCert != nil {
				results[cordial.ExecutableName()+" "+geneos.SigningCertBasename] = &responses.General{
					Value: listCertType{
						cordial.ExecutableName(),
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Until(signingCert.NotAfter).Truncate(time.Second),
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
		resp := instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertToolkit)
		resp.Report(listCSVWriter)

		// add headlines
		fmt.Printf("<!>totalCerts,%d\n", len(resp))
		var expiringWeek, expiringMonth, invalid int

		// TODO do this with more structured data (use responses.Value?)
		for _, c := range resp {
			var t time.Duration
			if len(c.Dataview.Table) == 0 {
				continue
			}
			if t, err = time.ParseDuration(c.Dataview.Table[0][4] + "s"); err != nil {
				err = nil
				continue
			}
			if t < expiringMonthLimit {
				expiringMonth++
			}
			if t < expiringWeekLimit {
				expiringWeek++
			}
			if c.Dataview.Table[0][7] != "OK" {
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

		instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Formatted(
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

		instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Formatted(
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

func listCertsLongCommand(command *cobra.Command, ct *geneos.Component, names []string, params []string) (err error) {
	switch {
	case listCmdJSON, listCmdIndent:
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		results := instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		if listCmdAll {
			if rootCert != nil {
				results[cordial.ExecutableName()+" "+geneos.RootCABasename] = &responses.General{
					Value: listCertLongType{
						cordial.ExecutableName(),
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Until(rootCert.NotAfter).Truncate(time.Second),
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
				results[cordial.ExecutableName()+" "+geneos.SigningCertBasename] = &responses.General{
					Value: listCertLongType{
						cordial.ExecutableName(),
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Until(signingCert.NotAfter).Truncate(time.Second),
						signingCert.NotAfter,
						signingCert.Subject.CommonName,
						verifyCertWithKey(signingKey, signingCert),
						signingCertFile,
						"",
						rootCertFile,
						signingCert.Issuer.CommonName,
						nil,
						nil,
						fmt.Sprintf("%X", sha1.Sum(signingCert.Raw)),
						fmt.Sprintf("%X", sha256.Sum256(signingCert.Raw)),
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
		resp := instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertToolkit)
		resp.Report(listCSVWriter)

		// add headlines
		fmt.Printf("<!>totalCerts,%d\n", len(resp))
		var expiringWeek, expiringMonth, invalid int
		for _, c := range resp {
			var t time.Duration
			if len(c.Dataview.Table) == 0 {
				continue
			}
			if t, err = time.ParseDuration(c.Dataview.Table[0][4] + "s"); err != nil {
				err = nil
				continue
			}
			if t < expiringMonthLimit {
				expiringMonth++
			}
			if t < expiringWeekLimit {
				expiringWeek++
			}
			if c.Dataview.Table[0][7] != "OK" {
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
			"TrustChain",
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
		instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Report(listCSVWriter)

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
				prequel = append(prequel, []string{
					cordial.ExecutableName(),
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					strconv.FormatFloat(time.Until(signingCert.NotAfter).Seconds(), 'f', 0, 64),
					signingCert.NotAfter.Format(time.RFC3339),
					signingCert.Subject.CommonName,
					verifyCertWithKey(signingKey, signingCert),
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
		instance.Do(command.CommandPath(), geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Formatted(
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
				"TrustChain",
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

// instanceCertInfo holds the details of an instance certificate used
// for the listings
type instanceCertInfo struct {
	cf         *config.Config
	cert       *x509.Certificate
	valid      string
	trustchain string
}

// readInstanceCertInfo returns the details of the instance certificate
// or false if the instance has no (readable) certificate, which is not
// an error for the listing.
func readInstanceCertInfo(i geneos.Instance) (info instanceCertInfo, ok bool) {
	certChain, err := instance.ReadCertificates(i)
	if err != nil || len(certChain) == 0 {
		return
	}
	cf := i.Config()
	key, _ := instance.ReadPrivateKey(i)
	return instanceCertInfo{
		cf:         cf,
		cert:       certChain[0],
		valid:      verifyCertWithKey(key, certChain...),
		trustchain: config.Get[string](cf, cf.Join(instance.TLSBASE, instance.CABUNDLE), config.PromoteFrom(instance.CERTCHAIN)),
	}, true
}

// certFile returns the instance certificate file path
func (info instanceCertInfo) certFile() string {
	return config.Get[string](info.cf, info.cf.Join(instance.TLSBASE, instance.CERTIFICATE), config.PromoteFrom(instance.CERTIFICATE))
}

// keyFile returns the instance private key file path
func (info instanceCertInfo) keyFile() string {
	return config.Get[string](info.cf, info.cf.Join(instance.TLSBASE, instance.PRIVATEKEY), config.PromoteFrom(instance.PRIVATEKEY))
}

func listCmdInstanceCert(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	info, ok := readInstanceCertInfo(i)
	if !ok {
		return
	}
	cert := info.cert

	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())
	cols := []string{i.Type().String(), i.Name(), i.Host().String(), until, expires.Format(time.RFC3339), cert.Subject.CommonName, info.valid}
	if listCmdLong {
		cols = append(cols,
			info.certFile(),
			info.keyFile(),
			info.trustchain,
			cert.Issuer.CommonName,
			fmt.Sprintf("%v", cert.DNSNames),
			fmt.Sprintf("%v", cert.IPAddresses),
			fmt.Sprintf("%X", sha1.Sum(cert.Raw)),
			fmt.Sprintf("%X", sha256.Sum256(cert.Raw)),
		)
	}

	resp.Dataview.Table = append(resp.Dataview.Table, cols)
	return
}

func listCmdInstanceCertCSV(i geneos.Instance, params ...any) (resp *responses.General) {
	// the CSV rows are the same as the plain table rows
	return listCmdInstanceCert(i, params...)
}

func listCmdInstanceCertToolkit(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	info, ok := readInstanceCertInfo(i)
	if !ok {
		return
	}
	cert := info.cert

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
		info.valid,
	}
	if listCmdLong {
		cols = append(cols,
			info.certFile(),
			info.keyFile(),
			info.trustchain,
			cert.Issuer.CommonName,
			strings.Join(cert.DNSNames, " "),
		)
		ips := []string{}
		for _, ip := range cert.IPAddresses {
			ips = append(ips, ip.String())
		}
		cols = append(cols,
			strings.Join(ips, " "),
			fmt.Sprintf("%X", sha1.Sum(cert.Raw)),
			fmt.Sprintf("%X", sha256.Sum256(cert.Raw)),
		)
	}

	resp.Dataview.Table = append(resp.Dataview.Table, cols)
	return
}

func listCmdInstanceCertJSON(i geneos.Instance, _ ...any) (resp *responses.General) {
	resp = responses.New[responses.General](i)

	info, ok := readInstanceCertInfo(i)
	if !ok {
		return
	}
	cert := info.cert

	remaining := time.Until(cert.NotAfter).Truncate(time.Second)

	if listCmdLong {
		resp.Value = listCertLongType{
			i.Type().String(),
			i.Name(),
			i.Host().String(),
			remaining,
			cert.NotAfter,
			cert.Subject.CommonName,
			info.valid,
			info.certFile(),
			info.keyFile(),
			info.trustchain,
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
		remaining,
		cert.NotAfter,
		cert.Subject.CommonName,
		info.valid,
	}
	return
}

// localCAs returns the certificates in the local CA bundle, which is
// only read once for each run of the command
var localCAs = sync.OnceValue(readLocalCAs)

func readLocalCAs() []*x509.Certificate {
	roots, _ := certs.ReadCertificates(geneos.LOCAL, geneos.PathToCABundlePEM(geneos.LOCAL))
	return roots
}

// verifyCertWithKey checks the certChain after appending the local CA
// bundle and checks if the private key provided matches the leaf
// certificate and returns a string that represents the result. "OK" if
// all is well.
func verifyCertWithKey(key certs.PrivateKey, certChain ...*x509.Certificate) string {
	if len(key) == 0 {
		return "NoKey"
	}
	if len(certChain) == 0 {
		return "NoCert"
	}
	certChain = slices.Concat(certChain, localCAs())
	if !certs.Verify(certChain...) {
		return "NotVerified"
	}
	match := certs.CheckKeyMatch(key, certChain[0])
	if !match {
		return "KeyMismatch"
	}
	return "OK"
}
