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
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
	Type        string        `json:"type,omitempty"`
	Name        string        `json:"name,omitempty"`
	Host        string        `json:"host,omitempty"`
	Remaining   time.Duration `json:"remaining,omitempty"`
	Expires     time.Time     `json:"expires,omitempty"`
	CommonName  string        `json:"common_name,omitempty"`
	Valid       bool          `json:"validd,omitempty"`
	Chainfile   string        `json:"chainfile,omitempty"`
	Issuer      string        `json:"issuer,omitempty"`
	SubAltNames []string      `json:"sans,omitempty"`
	IPs         []net.IP      `json:"ip_addresses,omitempty"`
	Signature   string        `json:"signature,omitempty"`
}

var listCmdAll, listCmdCSV, listCmdJSON, listCmdIndent, listCmdLong bool
var listJSONEncoder *json.Encoder

func init() {
	tlsCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listCmdAll, "all", "a", false, "Show all certs, including global and signing certs")
	listCmd.Flags().BoolVarP(&listCmdLong, "long", "l", false, "Long output")
	listCmd.Flags().BoolVarP(&listCmdJSON, "json", "j", false, "Output JSON")
	listCmd.Flags().BoolVarP(&listCmdIndent, "pretty", "i", false, "Output indented JSON")
	listCmd.Flags().BoolVarP(&listCmdCSV, "csv", "c", false, "Output CSV")

	listCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var listCmdDescription string

var rootCert, geneosCert *x509.Certificate
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
		rootCert, rootCertFile, err = geneos.ReadRootCert(true)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		geneosCert, geneosCertFile, err = geneos.ReadSigningCert()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
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
				results["global "+geneos.RootCABasename] = &instance.Response{
					Value: listCertType{
						"global",
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCert(rootCert),
					}}
			}
			if geneosCert != nil {
				results["global "+geneos.SigningCertBasename] = &instance.Response{
					Value: listCertType{
						"global",
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						geneosCert.NotAfter,
						geneosCert.Subject.CommonName,
						verifyCert(geneosCert),
					}}
			}
		}
		results.Write(os.Stdout, instance.WriterIndent(listCmdIndent))

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
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%0f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCert(rootCert)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					fmt.Sprintf("%0f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCert(geneosCert)),
				})
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Write(listCSVWriter)
	default:
		listTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintln(listTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tValid")
		if listCmdAll {
			if rootCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\n",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					verifyCert(rootCert))
			}
			if geneosCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\n",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert))
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Write(listTabWriter)
		listTabWriter.Flush()
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
				results["global "+geneos.RootCABasename] = &instance.Response{
					Value: listCertLongType{
						"global",
						geneos.RootCABasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						rootCert.NotAfter,
						rootCert.Subject.CommonName,
						verifyCert(rootCert),
						rootCertFile,
						rootCert.Issuer.CommonName,
						nil,
						nil,
						fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
					}}
			}
			if geneosCert != nil {
				results["global "+geneos.SigningCertBasename] = &instance.Response{
					Value: listCertLongType{
						"global",
						geneos.SigningCertBasename,
						geneos.LOCALHOST,
						time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
						geneosCert.NotAfter,
						geneosCert.Subject.CommonName,
						verifyCert(geneosCert),
						rootCertFile,
						geneosCert.Issuer.CommonName,
						nil,
						nil,
						fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
					}}
			}
		}
		results.Write(os.Stdout, instance.WriterIndent(listCmdIndent))
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
			"ChainFile",
			"Issuer",
			"SubjAltNames",
			"IPs",
			"Signature",
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
					fmt.Sprint(verifyCert(rootCert)),
					rootCertFile,
					rootCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
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
					fmt.Sprint(verifyCert(geneosCert)),
					rootCertFile,
					geneosCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
				})
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV).Write(listCSVWriter)
	default:
		listTabWriter := tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintln(listTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tValid\tChainFile\tIssuer\tSubjAltNames\tIPs\tFingerprint")
		if listCmdAll {
			if rootCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\t%q\t%q\t\t\t%X\n",
					geneos.RootCABasename,
					geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(),
					rootCert.NotAfter.Format(time.RFC3339),
					rootCert.Subject.CommonName,
					verifyCert(rootCert),
					rootCertFile,
					rootCert.Issuer.CommonName,
					sha1.Sum(rootCert.Raw))
			}
			if geneosCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\t%q\t%q\t\t\t%X\n",
					geneos.SigningCertBasename,
					geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(),
					geneosCert.NotAfter.Format(time.RFC3339),
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert),
					rootCertFile,
					geneosCert.Issuer.CommonName,
					sha1.Sum(geneosCert.Raw))
			}
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert).Write(listTabWriter)
		listTabWriter.Flush()
	}
	return
}

func listCmdInstanceCert(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cert, valid, chainfile, err := instance.ReadCert(i)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return
	}

	if cert == nil && err != nil {
		return
	}

	expires := cert.NotAfter
	resp.Line = fmt.Sprintf("%s\t%s\t%s\t%.f\t%q\t%q\t%v\t", i.Type(), i.Name(), i.Host(), time.Until(expires).Seconds(), expires.Format(time.RFC3339), cert.Subject.CommonName, valid)

	if listCmdLong {
		resp.Line += fmt.Sprintf("%q\t", chainfile)
		resp.Line += fmt.Sprintf("%q\t", cert.Issuer.CommonName)
		if len(cert.DNSNames) > 0 {
			resp.Line += fmt.Sprint(cert.DNSNames)
		}
		resp.Line += "\t"
		if len(cert.IPAddresses) > 0 {
			resp.Line += fmt.Sprint(cert.IPAddresses)
		}
		resp.Line += fmt.Sprintf("\t%X", sha1.Sum(cert.Raw))
	}
	return
}

func listCmdInstanceCertCSV(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cert, valid, chainfile, err := instance.ReadCert(i)
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
		cols = append(cols, chainfile)
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
	}

	resp.Rows = append(resp.Rows, cols)
	return
}

func listCmdInstanceCertJSON(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	cert, valid, chainfile, err := instance.ReadCert(i)
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
			chainfile,
			cert.Issuer.CommonName,
			cert.DNSNames,
			cert.IPAddresses,
			fmt.Sprintf("%X", sha1.Sum(cert.Raw)),
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

// verifyCert checks cert against the global rootCert and geneosCert
// (initialised in the main RunE()) and if that fails then against
// system certs. It also loads the Geneos global chain file and adds the
// certs to the verification pools after some basic validation.
func verifyCert(cert *x509.Certificate) bool {
	var rootCertPool, geneosCertPool *x509.CertPool

	if rootCert != nil {
		rootCertPool = x509.NewCertPool()
		rootCertPool.AddCert(rootCert)
	} else {
		rootCertPool, _ = x509.SystemCertPool()
	}

	if geneosCert != nil {
		geneosCertPool = x509.NewCertPool()
		geneosCertPool.AddCert(geneosCert)
	}

	// load chain, split into root and other
	chaincerts := config.ReadCertificates(geneos.LOCAL, geneos.LOCAL.PathTo("tls", geneos.ChainCertFile))
	for _, cert := range chaincerts {
		if bytes.Equal(cert.RawIssuer, cert.RawSubject) && cert.IsCA {
			// check root validity against itself
			selfRootPool := x509.NewCertPool()
			selfRootPool.AddCert(cert)
			if _, err := cert.Verify(x509.VerifyOptions{Roots: selfRootPool}); err != nil {
				log.Error().Err(err).Msg("root cert not valid")
				return false
			}
			rootCertPool.AddCert(cert)
		} else {
			if !cert.BasicConstraintsValid {
				continue
			}
			geneosCertPool.AddCert(cert)
		}
	}

	opts := x509.VerifyOptions{
		Roots:         rootCertPool,
		Intermediates: geneosCertPool,
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		log.Debug().Err(err).Msg("")
		return false
	}

	if len(chains) > 0 {
		log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
		return true
	}

	// if failed against internal certs, try system ones
	chains, err = cert.Verify(x509.VerifyOptions{})
	if err != nil {
		log.Debug().Err(err).Msg("")
		return false
	}
	if len(chains) > 0 {
		log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
		return true
	}

	log.Debug().Msgf("cert %q NOT verified", cert.Subject.CommonName)
	return false
}
