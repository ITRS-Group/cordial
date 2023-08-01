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
	"crypto/x509"
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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type listCertType struct {
	Type       string
	Name       string
	Host       string
	Remaining  time.Duration
	Expires    time.Time
	CommonName string
	Verified   bool
}

type listCertLongType struct {
	Type        string
	Name        string
	Host        string
	Remaining   time.Duration
	Expires     time.Time
	CommonName  string
	Verified    bool
	Issuer      string
	SubAltNames []string
	IPs         []net.IP
	Signature   string
}

var listCmdAll, listCmdCSV, listCmdJSON, listCmdIndent, listCmdLong bool
var listJSONEncoder *json.Encoder

var listTabWriter *tabwriter.Writer
var listCSVWriter *csv.Writer

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

var listCmd = &cobra.Command{
	Use:          "list [flags] [TYPE] [NAME...]",
	Short:        "List certificates",
	Long:         listCmdDescription,
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.AnnotationWildcard:  "true",
		cmd.AnnotationNeedsHome: "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		ct, names, params := cmd.TypeNamesParams(command)
		rootCert, _ = instance.ReadRootCert()
		geneosCert, _ = instance.ReadSigningCert()

		if listCmdLong {
			return listCertsLongCommand(ct, names, params)
		}
		return listCertsCommand(ct, names, params)
	},
}

func listCertsCommand(ct *geneos.Component, names []string, params []string) (err error) {
	switch {
	case listCmdJSON, listCmdIndent:
		var results []listCertType
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		if listCmdAll {
			if rootCert != nil {
				results = append(results, listCertType{
					"global",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
					verifyCert(rootCert),
				})
			}
			if geneosCert != nil {
				results = append(results, listCertType{
					"global",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert),
				})
			}
		}
		results2, _ := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		for _, r := range results2 {
			results = append(results, r.(listCertType))
		}
		listJSONEncoder.Encode(results)
	case listCmdCSV:
		listCSVWriter = csv.NewWriter(os.Stdout)
		listCSVWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
			"Verified",
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					fmt.Sprintf("%0f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.String(),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCert(rootCert)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					fmt.Sprintf("%0f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.String(),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCert(geneosCert)),
				})
			}
		}
		_, err = instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV)
		listCSVWriter.Flush()
	default:
		results, err := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert)
		if err != nil {
			return err
		}
		listTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintln(listTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tVerified")
		if listCmdAll {
			if rootCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\n",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
					verifyCert(rootCert))
			}
			if geneosCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\n",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert))
			}
		}
		for _, r := range results {
			fmt.Fprint(listTabWriter, r)
		}
		listTabWriter.Flush()
	}
	return
}

func listCertsLongCommand(ct *geneos.Component, names []string, params []string) (err error) {
	switch {
	case listCmdJSON:
		var results []listCertLongType
		listJSONEncoder = json.NewEncoder(os.Stdout)
		if listCmdIndent {
			listJSONEncoder.SetIndent("", "    ")
		}
		if listCmdAll {
			if rootCert != nil {
				results = append(results, listCertLongType{
					"global",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
					verifyCert(rootCert),
					rootCert.Issuer.CommonName,
					nil,
					nil,
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				results = append(results, listCertLongType{
					"global",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					time.Duration(time.Until(rootCert.NotAfter)).Truncate(time.Second),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert),
					geneosCert.Issuer.CommonName,
					nil,
					nil,
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
		}
		results2, _ := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertJSON)
		for _, r := range results2 {
			results = append(results, r.(listCertLongType))
		}
	case listCmdCSV:
		listCSVWriter = csv.NewWriter(os.Stdout)
		listCSVWriter.Write([]string{
			"Type",
			"Name",
			"Host",
			"Remaining",
			"Expires",
			"CommonName",
			"Verified",
			"Issuer",
			"SubjAltNames",
			"IPs",
			"Signature",
		})
		if listCmdAll {
			if rootCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(rootCert.NotAfter).Seconds()),
					rootCert.NotAfter.String(),
					rootCert.Subject.CommonName,
					fmt.Sprint(verifyCert(rootCert)),
					rootCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(rootCert.Raw)),
				})
			}
			if geneosCert != nil {
				listCSVWriter.Write([]string{
					"global",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					fmt.Sprintf("%.f", time.Until(geneosCert.NotAfter).Seconds()),
					geneosCert.NotAfter.String(),
					geneosCert.Subject.CommonName,
					fmt.Sprint(verifyCert(geneosCert)),
					geneosCert.Issuer.CommonName,
					"[]",
					"[]",
					fmt.Sprintf("%X", sha1.Sum(geneosCert.Raw)),
				})
			}
		}
		_, err = instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCertCSV)
		listCSVWriter.Flush()
	default:
		results, err := instance.Do(geneos.GetHost(cmd.Hostname), ct, names, listCmdInstanceCert)
		if err != nil {
			return err
		}
		listTabWriter = tabwriter.NewWriter(os.Stdout, 3, 8, 2, ' ', 0)
		fmt.Fprintln(listTabWriter, "Type\tName\tHost\tRemaining\tExpires\tCommonName\tVerified\tIssuer\tSubjAltNames\tIPs\tFingerprint")
		if listCmdAll {
			if rootCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\t%q\t\t\t%X\n",
					geneos.RootCAFile,
					geneos.LOCALHOST,
					time.Until(rootCert.NotAfter).Seconds(),
					rootCert.NotAfter,
					rootCert.Subject.CommonName,
					verifyCert(rootCert),
					rootCert.Issuer.CommonName,
					sha1.Sum(rootCert.Raw))
			}
			if geneosCert != nil {
				fmt.Fprintf(listTabWriter, "global\t%s\t%s\t%.f\t%q\t%q\t%v\t%q\t\t\t%X\n",
					geneos.SigningCertFile,
					geneos.LOCALHOST,
					time.Until(geneosCert.NotAfter).Seconds(),
					geneosCert.NotAfter,
					geneosCert.Subject.CommonName,
					verifyCert(geneosCert),
					geneosCert.Issuer.CommonName,
					sha1.Sum(geneosCert.Raw))
			}
		}
		for _, r := range results {
			fmt.Fprint(listTabWriter, r)
		}
		listTabWriter.Flush()
	}
	return
}

func listCmdInstanceCert(c geneos.Instance) (result any, err error) {
	var output string

	cert, valid, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK - instance.ReadCert() reports no configured cert this way
		return "", nil
	}
	if cert == nil && err != nil {
		return
	}

	expires := cert.NotAfter
	output = fmt.Sprintf("%s\t%s\t%s\t%.f\t%q\t%q\t%v\t", c.Type(), c.Name(), c.Host(), time.Until(expires).Seconds(), expires, cert.Subject.CommonName, valid)

	if listCmdLong {
		output += fmt.Sprintf("%q\t", cert.Issuer.CommonName)
		if len(cert.DNSNames) > 0 {
			output += fmt.Sprint(cert.DNSNames)
		}
		output += "\t"
		if len(cert.IPAddresses) > 0 {
			output += fmt.Sprint(cert.IPAddresses)
		}
		output += fmt.Sprintf("\t%X", sha1.Sum(cert.Raw))
	}
	output += "\n"
	return output, nil
}

func listCmdInstanceCertCSV(c geneos.Instance) (result any, err error) {
	cert, valid, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK
		err = nil
		return
	}
	if err != nil {
		return
	}
	expires := cert.NotAfter
	until := fmt.Sprintf("%.f", time.Until(expires).Seconds())
	cols := []string{c.Type().String(), c.Name(), c.Host().String(), until, expires.String(), cert.Subject.CommonName, fmt.Sprint(valid)}
	if listCmdLong {
		cols = append(cols, cert.Issuer.CommonName)
		cols = append(cols, fmt.Sprintf("%v", cert.DNSNames))
		cols = append(cols, fmt.Sprintf("%v", cert.IPAddresses))
		cols = append(cols, fmt.Sprintf("%X", sha1.Sum(cert.Raw)))
	}

	listCSVWriter.Write(cols)
	return
}

func listCmdInstanceCertJSON(c geneos.Instance) (result any, err error) {
	cert, valid, err := instance.ReadCert(c)
	if err == os.ErrNotExist {
		// this is OK
		err = nil
		return
	}
	if err != nil {
		return
	}
	if listCmdLong {
		result = listCertLongType{
			c.Type().String(),
			c.Name(),
			c.Host().String(),
			time.Duration(time.Until(cert.NotAfter).Seconds()),
			cert.NotAfter,
			cert.Subject.CommonName,
			valid,
			cert.Issuer.CommonName,
			cert.DNSNames,
			cert.IPAddresses,
			fmt.Sprintf("%X", sha1.Sum(cert.Raw)),
		}
		return
	}
	result = listCertType{
		c.Type().String(),
		c.Name(),
		c.Host().String(),
		time.Duration(time.Until(cert.NotAfter).Seconds()),
		cert.NotAfter,
		cert.Subject.CommonName,
		valid,
	}
	return
}

// verifyCert checks cert against the global rootCert and geneosCert
// (initialised in the main RunE()) and if that fails then against
// system certs.
func verifyCert(cert *x509.Certificate) bool {
	rootCertPool := x509.NewCertPool()
	rootCertPool.AddCert(rootCert)

	geneosCertPool := x509.NewCertPool()
	geneosCertPool.AddCert(geneosCert)

	opts := x509.VerifyOptions{
		Roots:         rootCertPool,
		Intermediates: geneosCertPool,
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		log.Debug().Err(err).Msg("")
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
