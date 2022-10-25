/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type lsCertType struct {
	Type       string
	Name       string
	Host       string
	Remaining  time.Duration
	Expires    time.Time
	CommonName string
}

type lsCertLongType struct {
	Type        string
	Name        string
	Host        string
	Remaining   time.Duration
	Expires     time.Time
	CommonName  string
	Issuer      string
	SubAltNames []string
	IPs         []net.IP
	Signature   string
}

var tlsCmdAll, tlsCmdCSV, tlsCmdJSON, tlsCmdIndent, tlsCmdLong bool

func init() {
	tlsCmd.AddCommand(tlsLsCmd)

	tlsLsCmd.Flags().BoolVarP(&tlsCmdAll, "all", "a", false, "Show all certs, including global and signing certs")
	tlsLsCmd.Flags().BoolVarP(&tlsCmdJSON, "json", "j", false, "Output JSON")
	tlsLsCmd.Flags().BoolVarP(&tlsCmdLong, "long", "l", false, "Long output")
	tlsLsCmd.Flags().BoolVarP(&tlsCmdIndent, "pretty", "i", false, "Indent / pretty print JSON")
	tlsLsCmd.Flags().BoolVarP(&tlsCmdCSV, "csv", "c", false, "Output CSV")

	tlsLsCmd.Flags().SortFlags = false
}

var tlsLsCmd = &cobra.Command{
	Use:   "ls [flags] [TYPE] [NAME...]",
	Short: "List certificates",
	Long: strings.ReplaceAll(`
List certificates and their details. The root and signing
certs are only shown in the |-a| flag is given. A list with more
details can be seen with the |-l| flag, otherwise options are the
same as for the main ls command.
`, "|", "`"),
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard": "true",
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		ct, args, params := cmdArgsParams(cmd)
		if tlsCmdLong {
			return listCertsLongCommand(ct, args, params)
		}
		return listCertsCommand(ct, args, params)
	},
}

func listCertsCommand(ct *geneos.Component, args []string, params []string) (err error) {
	rootCert, _ := instance.ReadRootCert()
	geneosCert, _ := instance.ReadSigningCert()

	switch {
	case tlsCmdJSON:
		jsonEncoder = json.NewEncoder(os.Stdout)
		if tlsCmdIndent {
			jsonEncoder.SetIndent("", "    ")
		}
		if tlsCmdAll {
			if rootCert != nil {
				jsonEncoder.Encode(lsCertType{
					"global",
					geneos.RootCAFile,
					string(host.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
				})
			}
			if geneosCert != nil {
				jsonEncoder.Encode(lsCertType{
					"global",
					geneos.SigningCertFile,
					string(host.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
				})
			}
		}
		err = instance.ForAll(ct, lsInstanceCertJSON, args, params)
	case tlsCmdCSV:
		csvWriter = csv.NewWriter(os.Stdout)
		csvWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
		})
		if tlsCmdAll {
			if rootCert != nil {
				csvWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					string(host.LOCALHOST),
					fmt.Sprintf("%0f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.String(),
					rootCert.Subject.CommonName,
				})
			}
			if geneosCert != nil {
				csvWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					string(host.LOCALHOST),
					fmt.Sprintf("%0f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.String(),
					geneosCert.Subject.CommonName,
				})
			}
		}
		err = instance.ForAll(ct, lsInstanceCertCSV, args, params)
		csvWriter.Flush()
	default:
		lsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(lsTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\n")
		if tlsCmdAll {
			if rootCert != nil {
				fmt.Fprintf(lsTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\n", geneos.RootCAFile, host.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(), rootCert.NotAfter,
					rootCert.Subject.CommonName)
			}
			if geneosCert != nil {
				fmt.Fprintf(lsTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\n", geneos.SigningCertFile, host.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(), geneosCert.NotAfter,
					geneosCert.Subject.CommonName)
			}
		}
		err = instance.ForAll(ct, lsInstanceCert, args, params)
		lsTabWriter.Flush()
	}
	return
}

func listCertsLongCommand(ct *geneos.Component, args []string, params []string) (err error) {
	rootCert, _ := instance.ReadRootCert()
	geneosCert, _ := instance.ReadSigningCert()

	switch {
	case tlsCmdJSON:
		jsonEncoder = json.NewEncoder(os.Stdout)
		if tlsCmdIndent {
			jsonEncoder.SetIndent("", "    ")
		}
		if tlsCmdAll {
			if rootCert != nil {
				jsonEncoder.Encode(lsCertLongType{
					"global",
					geneos.RootCAFile,
					string(host.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
					rootCert.Issuer.CommonName,
					nil,
					nil,
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				jsonEncoder.Encode(lsCertLongType{
					"global",
					geneos.SigningCertFile,
					string(host.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
					geneosCert.Issuer.CommonName,
					nil,
					nil,
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
		}
		err = instance.ForAll(ct, lsInstanceCertJSON, args, params)
	case tlsCmdCSV:
		csvWriter = csv.NewWriter(os.Stdout)
		csvWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
			"Issuer",
			"SubjAltNames",
			"IPs",
			"Signature",
		})
		if tlsCmdAll {
			if rootCert != nil {
				csvWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					string(host.LOCALHOST),
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.String(),
					rootCert.Subject.CommonName,
					rootCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				csvWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					string(host.LOCALHOST),
					fmt.Sprintf("%.f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.String(),
					geneosCert.Subject.CommonName,
					geneosCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
				})
			}
		}
		err = instance.ForAll(ct, lsInstanceCertCSV, args, params)
		csvWriter.Flush()
	default:
		lsTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(lsTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tIssuer\tSubjAltNames\tIPs\tFingerprint\n")
		if tlsCmdAll {
			if rootCert != nil {
				fmt.Fprintf(lsTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%q\t\t\t%X\n", geneos.RootCAFile, host.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(), rootCert.NotAfter,
					rootCert.Subject.CommonName, rootCert.Issuer.CommonName, sha1.Sum(rootCert.Raw))
			}
			if geneosCert != nil {
				fmt.Fprintf(lsTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%q\t\t\t%X\n", geneos.SigningCertFile, host.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(), geneosCert.NotAfter,
					geneosCert.Subject.CommonName, geneosCert.Issuer.CommonName, sha1.Sum(geneosCert.Raw))
			}
		}
		err = instance.ForAll(ct, lsInstanceCert, args, params)
		lsTabWriter.Flush()
	}
	return
}

func lsInstanceCert(c geneos.Instance, params []string) (err error) {
	cert, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return nil
	}
	if err != nil {
		return
	}
	expires := cert.NotAfter
	fmt.Fprintf(lsTabWriter, "%s\t%s\t%s\t%.f\t%q\t%q\t", c.Type(), c.Name(), c.Host(), time.Until(expires).Seconds(), expires, cert.Subject.CommonName)

	if tlsCmdLong {
		fmt.Fprintf(lsTabWriter, "%q\t", cert.Issuer.CommonName)
		if len(cert.DNSNames) > 0 {
			fmt.Fprintf(lsTabWriter, "%v", cert.DNSNames)
		}
		fmt.Fprintf(lsTabWriter, "\t")
		if len(cert.IPAddresses) > 0 {
			fmt.Fprintf(lsTabWriter, "%v", cert.IPAddresses)
		}
		fmt.Fprintf(lsTabWriter, "\t%X", sha1.Sum(cert.Raw))
	}
	fmt.Fprint(lsTabWriter, "\n")
	return
}

func lsInstanceCertCSV(c geneos.Instance, params []string) (err error) {
	cert, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK
		return nil
	}
	if err != nil {
		return
	}
	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())
	cols := []string{c.Type().String(), c.Name(), c.Host().String(), until, expires.String(), cert.Subject.CommonName}
	if tlsCmdLong {
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
	}

	csvWriter.Write(cols)
	return
}

func lsInstanceCertJSON(c geneos.Instance, params []string) (err error) {
	cert, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK
		return nil
	}
	if err != nil {
		return
	}
	if tlsCmdLong {
		jsonEncoder.Encode(lsCertLongType{c.Type().String(), c.Name(), c.Host().String(), time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter, cert.Subject.CommonName, cert.Issuer.CommonName, cert.DNSNames, cert.IPAddresses, fmt.Sprintf("%X", sha1.Sum(cert.Raw))})
	} else {
		jsonEncoder.Encode(lsCertType{c.Type().String(), c.Name(), c.Host().String(), time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter, cert.Subject.CommonName})
	}
	return
}
