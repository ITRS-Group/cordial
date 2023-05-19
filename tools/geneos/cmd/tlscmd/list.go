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

package tlscmd

import (
	"crypto/sha1"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"text/tabwriter"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/spf13/cobra"
)

type tlsListCertType struct {
	Type       string
	Name       string
	Host       string
	Remaining  time.Duration
	Expires    time.Time
	CommonName string
}

type tlsListCertLongType struct {
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

var tlsListCmdAll, tlsListCmdCSV, tlsListCmdJSON, tlsListCmdIndent, tlsListCmdLong bool
var tlsJSONEncoder *json.Encoder

var tlsListTabWriter *tabwriter.Writer
var tlsListCSVWriter *csv.Writer

func init() {
	tlsCmd.AddCommand(tlsListCmd)

	tlsListCmd.Flags().BoolVarP(&tlsListCmdAll, "all", "a", false, "Show all certs, including global and signing certs")
	tlsListCmd.Flags().BoolVarP(&tlsListCmdLong, "long", "l", false, "Long output")
	tlsListCmd.Flags().BoolVarP(&tlsListCmdJSON, "json", "j", false, "Output JSON")
	tlsListCmd.Flags().BoolVarP(&tlsListCmdIndent, "pretty", "i", false, "Output indented JSON")
	tlsListCmd.Flags().BoolVarP(&tlsListCmdCSV, "csv", "c", false, "Output CSV")

	tlsListCmd.Flags().SortFlags = false
}

//go:embed _docs/list.md
var tlsListCmdDescription string

var tlsListCmd = &cobra.Command{
	Use:          "ls [flags] [TYPE] [NAME...]",
	Short:        "List certificates",
	Long:         tlsListCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, args, params := cmd.CmdArgsParams(command)
		if tlsListCmdLong {
			return listCertsLongCommand(ct, args, params)
		}
		return listCertsCommand(ct, args, params)
	},
}

func listCertsCommand(ct *geneos.Component, args []string, params []string) (err error) {
	rootCert, _ := instance.ReadRootCert()
	geneosCert, _ := instance.ReadSigningCert()

	switch {
	case tlsListCmdJSON, tlsListCmdIndent:
		tlsJSONEncoder = json.NewEncoder(os.Stdout)
		if tlsListCmdIndent {
			tlsJSONEncoder.SetIndent("", "    ")
		}
		if tlsListCmdAll {
			if rootCert != nil {
				tlsJSONEncoder.Encode(tlsListCertType{
					"global",
					geneos.RootCAFile,
					string(geneos.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
				})
			}
			if geneosCert != nil {
				tlsJSONEncoder.Encode(tlsListCertType{
					"global",
					geneos.SigningCertFile,
					string(geneos.LOCALHOST),
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
				})
			}
		}
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCertJSON, args, params)
	case tlsListCmdCSV:
		tlsListCSVWriter = csv.NewWriter(os.Stdout)
		tlsListCSVWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
		})
		if tlsListCmdAll {
			if rootCert != nil {
				tlsListCSVWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					string(geneos.LOCALHOST),
					fmt.Sprintf("%0f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.String(),
					rootCert.Subject.CommonName,
				})
			}
			if geneosCert != nil {
				tlsListCSVWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					string(geneos.LOCALHOST),
					fmt.Sprintf("%0f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.String(),
					geneosCert.Subject.CommonName,
				})
			}
		}
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCertCSV, args, params)
		tlsListCSVWriter.Flush()
	default:
		tlsListTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(tlsListTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\n")
		if tlsListCmdAll {
			if rootCert != nil {
				fmt.Fprintf(tlsListTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\n", geneos.RootCAFile, geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(), rootCert.NotAfter,
					rootCert.Subject.CommonName)
			}
			if geneosCert != nil {
				fmt.Fprintf(tlsListTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\n", geneos.SigningCertFile, geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(), geneosCert.NotAfter,
					geneosCert.Subject.CommonName)
			}
		}
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCert, args, params)
		tlsListTabWriter.Flush()
	}
	return
}

func listCertsLongCommand(ct *geneos.Component, args []string, params []string) (err error) {
	rootCert, _ := instance.ReadRootCert()
	geneosCert, _ := instance.ReadSigningCert()

	switch {
	case tlsListCmdJSON:
		tlsJSONEncoder = json.NewEncoder(os.Stdout)
		if tlsListCmdIndent {
			tlsJSONEncoder.SetIndent("", "    ")
		}
		if tlsListCmdAll {
			if rootCert != nil {
				tlsJSONEncoder.Encode(tlsListCertLongType{
					"global",
					geneos.RootCAFile,
					string(geneos.LOCALHOST),
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
				tlsJSONEncoder.Encode(tlsListCertLongType{
					"global",
					geneos.SigningCertFile,
					string(geneos.LOCALHOST),
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
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCertJSON, args, params)
	case tlsListCmdCSV:
		tlsListCSVWriter = csv.NewWriter(os.Stdout)
		tlsListCSVWriter.Write([]string{
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
		if tlsListCmdAll {
			if rootCert != nil {
				tlsListCSVWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					string(geneos.LOCALHOST),
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
				tlsListCSVWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					string(geneos.LOCALHOST),
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
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCertCSV, args, params)
		tlsListCSVWriter.Flush()
	default:
		tlsListTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintf(tlsListTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tIssuer\tSubjAltNames\tIPs\tFingerprint\n")
		if tlsListCmdAll {
			if rootCert != nil {
				fmt.Fprintf(tlsListTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%q\t\t\t%X\n", geneos.RootCAFile, geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(), rootCert.NotAfter,
					rootCert.Subject.CommonName, rootCert.Issuer.CommonName, sha1.Sum(rootCert.Raw))
			}
			if geneosCert != nil {
				fmt.Fprintf(tlsListTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%q\t\t\t%X\n", geneos.SigningCertFile, geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(), geneosCert.NotAfter,
					geneosCert.Subject.CommonName, geneosCert.Issuer.CommonName, sha1.Sum(geneosCert.Raw))
			}
		}
		err = instance.ForAll(ct, cmd.Hostname, lsInstanceCert, args, params)
		tlsListTabWriter.Flush()
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
	fmt.Fprintf(tlsListTabWriter, "%s\t%s\t%s\t%.f\t%q\t%q\t", c.Type(), c.Name(), c.Host(), time.Until(expires).Seconds(), expires, cert.Subject.CommonName)

	if tlsListCmdLong {
		fmt.Fprintf(tlsListTabWriter, "%q\t", cert.Issuer.CommonName)
		if len(cert.DNSNames) > 0 {
			fmt.Fprintf(tlsListTabWriter, "%v", cert.DNSNames)
		}
		fmt.Fprintf(tlsListTabWriter, "\t")
		if len(cert.IPAddresses) > 0 {
			fmt.Fprintf(tlsListTabWriter, "%v", cert.IPAddresses)
		}
		fmt.Fprintf(tlsListTabWriter, "\t%X", sha1.Sum(cert.Raw))
	}
	fmt.Fprint(tlsListTabWriter, "\n")
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
	if tlsListCmdLong {
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
	}

	tlsListCSVWriter.Write(cols)
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
	if tlsListCmdLong {
		tlsJSONEncoder.Encode(tlsListCertLongType{c.Type().String(), c.Name(), c.Host().String(), time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter, cert.Subject.CommonName, cert.Issuer.CommonName, cert.DNSNames, cert.IPAddresses, fmt.Sprintf("%X", sha1.Sum(cert.Raw))})
	} else {
		tlsJSONEncoder.Encode(tlsListCertType{c.Type().String(), c.Name(), c.Host().String(), time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter, cert.Subject.CommonName})
	}
	return
}
