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
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type infoCmdConnectsType []string
type infoCmdRootsType []string

// var infoCmdJSON, infoCmdIndent, infoCmdCSV, infoCmdToolkit bool
var infoCmdFormat string
var infoCmdLong, infoCmdLeafOnly bool
var infoCmdPassword config.Secret
var infoCmdConnects infoCmdConnectsType
var infoCmdConnectsFile string
var infoCmdRoots infoCmdRootsType

func init() {
	tlsCmd.AddCommand(infoCmd)

	infoCmd.Flags().BoolVarP(&infoCmdLeafOnly, "leaf-only", "L", false,
		"Only output leaf certificates (i.e. skip CA certificates and any\ncertificate without a matching private key in any file)")
	infoCmd.Flags().BoolVarP(&infoCmdLong, "long", "l", false, "Output long format (more columns)")

	infoCmd.Flags().StringVarP(&infoCmdFormat, "format", "f", "column", "Output format (column, table, csv, toolkit)")
	infoCmd.Flags().VarP(&infoCmdPassword, "password", "p", "Password for PFX/PKCS#12 file(s), if needed. Defaults to prompting for each file. Use -p \"\" to specify empty password.")

	infoCmd.Flags().VarP(&infoCmdConnects, "connect", "c", "Connect to a URL or HOST[:PORT] to get TLS certificates. Can be specified multiple times for multiple instances.")
	infoCmd.Flags().StringVarP(&infoCmdConnectsFile, "connect-file", "", "", "Path to file containing list of URLs or HOST[:PORT] to connect to (one per line) to get TLS certificates.")
	infoCmd.Flags().VarP(&infoCmdRoots, "roots", "r", "Path to additional root certificates to use when verifying TLS connections. Can be specified multiple times for multiple files. If not specified, the system root CAs and the Geneos ca-bundle files will be used (and are always included).")

	infoCmd.Flags().SortFlags = false
}

//go:embed _docs/info.md
var infoCmdDescription string

type certInfo struct {
	Path               string
	Error              error
	ResponseTime       time.Duration
	Alias              []string
	CertChain          []*x509.Certificate
	CerificateVerified []bool
	PrivateKeys        []certs.PrivateKey
}

var columns = []string{
	"FileAndIndex",
	"Status",
	"CommonName",
	"IssuerCommonName",
	"NotAfter",
	"IsCA",
	"ExtKeyUsage",
	"PrivateKeyMatch",
	"Verified",
	"ResponseTime",
}

var columnsLong = []string{
	"PathAndIndex",
	"Status",
	"CommonName",
	"IssuerCommonName",
	"NotBefore",
	"NotAfter",
	"IsCA",
	"KeyUsage",
	"ExtKeyUsage",
	"PrivateKeyMatch",
	"Verified",
	"Serial",
	"SKID",
	"AKID",
	"SANDNSNames",
	"SANEmailAddresses",
	"SANIPAddresses",
	"SANURIs",
	"SHA1Fingerprint",
	"SHA256Fingerprint",
	"ResponseTime",
}

var infoCmd = &cobra.Command{
	Use:          "info [PATH...]",
	Short:        "Info about certificates and keys",
	Long:         infoCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:      "false",
		cmd.CmdRequireHome: "false",
		cmd.CmdAllowRoot:   "true",
	},
	RunE: func(command *cobra.Command, paths []string) (err error) {
		log := cordial.Logger.With("command", "tls info")
		// gather cert info
		certInfos := make([]certInfo, len(paths))

		roots, err := x509.SystemCertPool()
		if err != nil {
			log.Error("unable to load system root CAs, skipping system roots", slog.Any("error", err))
			roots = x509.NewCertPool()
		}

		cabundle, err := os.ReadFile(geneos.PathToCABundlePEM(geneos.LOCAL))
		if err == nil {
			log.Debug("loaded Geneos CA bundle", slog.String("file", geneos.PathToCABundlePEM(geneos.LOCAL)))
			if ok := roots.AppendCertsFromPEM(cabundle); !ok {
				log.Error("unable to parse any certificates from Geneos CA bundle")
			}
		}

		if len(infoCmdRoots) > 0 {
			for _, r := range infoCmdRoots {
				contents, err := os.ReadFile(r)
				if err != nil {
					log.Error("unable to read roots file", slog.Any("error", err), slog.String("file", r))
					return err
				}
				if !roots.AppendCertsFromPEM(contents) {
					log.Error("unable to parse any certificates from roots file", slog.String("file", r))
					return fmt.Errorf("unable to parse any certificates from roots file: %s", r)
				}
			}
		}

		certInfos = readFiles(paths, roots)

		if len(infoCmdConnects) > 0 {
			var wg sync.WaitGroup
			ch := make(chan certInfo, len(infoCmdConnects))
			for _, addr := range infoCmdConnects {
				wg.Add(1)
				go func(ch chan certInfo, addr string) {
					defer wg.Done()
					ci := getCertificatesFromConnection(addr, roots)
					ch <- ci
				}(ch, addr)
			}
			wg.Wait()
			close(ch)

			for ci := range ch {
				certInfos = append(certInfos, ci)
			}
		}

		if infoCmdConnectsFile != "" {
			var r io.ReadCloser
			if infoCmdConnectsFile == "-" {
				r = os.Stdin
			} else {
				r, err = os.Open(infoCmdConnectsFile)
				if err != nil {
					log.Error("unable to open connects file", slog.Any("error", err), slog.String("file", infoCmdConnectsFile))
					return err
				}
				defer r.Close()
			}

			// first count the number of lines, so we can create a
			// channel of the right capacty
			var lines []string
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				addr := strings.TrimSpace(scanner.Text())
				if addr == "" || strings.HasPrefix(addr, "#") {
					continue
				}
				lines = append(lines, addr)
			}

			var wg sync.WaitGroup
			ch := make(chan certInfo, len(lines))

			for _, addr := range lines {
				wg.Add(1)
				if len(ch) == cap(ch) {
					log.Warn("channel buffer full, waiting for some connections to finish before starting new ones")
				}
				go func(ch chan certInfo, addr string) {
					defer wg.Done()
					ci := getCertificatesFromConnection(addr, roots)
					ch <- ci
				}(ch, addr)

				// ci := getCertificatesFromConnection(addr, roots)
				// certInfos = append(certInfos, ci)
			}
			wg.Wait()
			close(ch)

			for ci := range ch {
				certInfos = append(certInfos, ci)
			}
			if err := scanner.Err(); err != nil {
				log.Error("error reading connects file", slog.Any("error", err), slog.String("file", infoCmdConnectsFile))
				return err
			}
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

		var totalVerified, totalCerts, totalExpired, totalExpiring30days int64

		// sort output by path, allow lop below to order by index
		slices.SortFunc(certInfos, func(a, b certInfo) int {
			return strings.Compare(a.Path, b.Path)
		})

		for i, ci := range certInfos {
			if ci.Error != nil {
				lines = append(lines, []string{
					strings.TrimSuffix(path.Base(ci.Path), ":443"),
					"ERROR: " + ci.Error.Error(),
				})
				continue
			}
			for n, c := range ci.CertChain {
				var verified bool
				var fileandindex string
				status := "OK"

				privateKeyPresent := slices.ContainsFunc(certInfos, func(ci certInfo) bool {
					for _, pk := range ci.PrivateKeys {
						if certs.CheckKeyMatch(pk, c) {
							return true
						}
					}
					return false
				})

				if infoCmdLeafOnly {
					// skip if it's a CA cert or if it doesn't have a matching private key in any file
					if c.IsCA {
						continue
					}
					fileandindex = strings.TrimSuffix(path.Base(certInfos[i].Path), ":443")
				} else {
					name := strings.TrimSuffix(path.Base(certInfos[i].Path), ":443")
					if len(certInfos[i].Alias) > n && certInfos[i].Alias[n] != "" {
						fileandindex = fmt.Sprintf("%s/%s", name, certInfos[i].Alias[n])
					} else {
						fileandindex = fmt.Sprintf("%s/%d", name, n+1)
					}
				}

				if len(ci.CerificateVerified) > n {
					if ci.CerificateVerified[n] {
						verified = true
						totalVerified++
					} else {
						status = "Unverified"
					}
				}

				totalCerts++
				if c.NotAfter.Before(time.Now()) {
					status = "Expired"
					totalExpired++
				}
				if c.NotAfter.Before(time.Now().Add(30 * 24 * time.Hour)) {
					totalExpiring30days++
				}

				if !infoCmdLong {
					lines = append(lines, []string{
						fileandindex,
						status,
						c.Subject.CommonName,
						c.Issuer.CommonName,
						c.NotAfter.UTC().Format(time.RFC3339),
						strconv.FormatBool(c.IsCA),
						extKeyUsageToString(c.ExtKeyUsage),
						strconv.FormatBool(privateKeyPresent),
						strconv.FormatBool(verified),
						fmt.Sprintf("%dms", ci.ResponseTime.Milliseconds()),
					})
				} else {
					dnsNames := "None"
					if len(c.DNSNames) > 0 {
						dnsNames = strings.Join(c.DNSNames, " ")
					}
					emailAddresses := "None"
					if len(c.EmailAddresses) > 0 {
						emailAddresses = strings.Join(c.EmailAddresses, " ")
					}

					lines = append(lines, []string{
						fileandindex,
						status,
						c.Subject.CommonName,
						c.Issuer.CommonName,
						c.NotBefore.UTC().Format(time.RFC3339),
						c.NotAfter.UTC().Format(time.RFC3339),
						strconv.FormatBool(c.IsCA),
						keyUsageToString(c.KeyUsage),
						extKeyUsageToString(c.ExtKeyUsage),
						strconv.FormatBool(privateKeyPresent),
						strconv.FormatBool(verified),
						fmt.Sprintf("%X", c.SerialNumber),
						fmt.Sprintf("%X", c.SubjectKeyId),
						fmt.Sprintf("%X", c.AuthorityKeyId),
						dnsNames,
						emailAddresses,
						infoMapString(c.IPAddresses, func(ip net.IP) string { return ip.String() }),
						infoMapString(c.URIs, func(uri *url.URL) string { return uri.String() }),
						fmt.Sprintf("%X", sha1.Sum(c.Raw)),
						fmt.Sprintf("%X", sha256.Sum256(c.Raw)),
						fmt.Sprintf("%dms", ci.ResponseTime.Milliseconds()),
					})
				}
			}
		}

		if infoCmdLong {
			rp.UpdateTable(columnsLong, lines)
		} else {
			rp.UpdateTable(columns, lines)
		}

		if infoCmdFormat == "toolkit" {
			// rp.AddHeadline()
			rp.AddHeadline("totalCerts", strconv.FormatInt(totalCerts, 10))
			rp.AddHeadline("totalVerified", strconv.FormatInt(totalVerified, 10))
			rp.AddHeadline("totalExpired", strconv.FormatInt(totalExpired, 10))
			rp.AddHeadline("totalExpiring30days", strconv.FormatInt(totalExpiring30days, 10))
		}

		rp.Render()

		return
	},
}

func getCertificatesFromConnection(addr string, roots *x509.CertPool) (ci certInfo) {
	u, err := url.Parse(addr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		if host, port, found := strings.Cut(addr, ":"); found {
			u = &url.URL{
				Scheme: "https",
				Host:   net.JoinHostPort(host, port),
			}
		} else {
			u = &url.URL{
				Scheme: "https",
				Host:   net.JoinHostPort(addr, "443"),
			}
		}
	}
	if u.Port() == "" {
		u.Host = net.JoinHostPort(u.Host, "443")
	}

	ci = certInfo{
		Path: u.Host,
	}
	start := time.Now()

	var verifiedChains [][]*x509.Certificate
	conn, err := tls.Dial("tcp", u.Host, &tls.Config{
		// RootCAs: roots,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) (err error) {
			// Parse the root/leaf certificates
			log.Debug("verifying peer certificate", slog.Int("rawCerts", len(rawCerts)))

			certs := make([]*x509.Certificate, len(rawCerts))
			for i, asn1Data := range rawCerts {
				cert, err := x509.ParseCertificate(asn1Data)
				if err != nil {
					return fmt.Errorf("failed to parse certificate from server: %v", err)
				}
				certs[i] = cert
			}

			// 3. Setup verification options
			opts := x509.VerifyOptions{
				Roots:         roots,
				Intermediates: x509.NewCertPool(),
			}

			// Add intermediate certificates to the pool
			for _, cert := range certs[1:] {
				log.Debug("adding intermediate certificate to pool for verification", slog.String("subjectCN", cert.Subject.CommonName))
				opts.Intermediates.AddCert(cert)
			}

			// Perform verification (DNSName is empty, so hostname check is skipped)
			verifiedChains, err = certs[0].Verify(opts)
			return err
		},
	})

	if err != nil {
		// Check if the error is a certificate verification error
		_, ok := errors.AsType[*tls.CertificateVerificationError](err)
		_, ok2 := errors.AsType[x509.UnknownAuthorityError](err)
		if ok || ok2 {
			start = time.Now() // reset start time to exclude time taken by failed verification attempt
			log.Debug("TLS certificate verification failed, retrying with InsecureSkipVerify to get certificates anyway", slog.Any("error", err), slog.String("address", addr))
			// try again with skip verify
			conn, err = tls.Dial("tcp", u.Host, &tls.Config{
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Debug("unable to connect to address with TLS", slog.Any("error", err), slog.String("address", addr))
				ci.Error = err
				return
			}
		} else {
			log.Debug("unable to connect to address with TLS", slog.Any("error", err), slog.String("address", addr), slog.String("errorType", fmt.Sprintf("%T", err)))
			ci.Error = err
			return
		}
	}

	defer conn.Close()
	ci.ResponseTime = time.Since(start)

	var verified bool
	if len(verifiedChains) > 0 {
		log.Debug("TLS certificate chain successfully verified", slog.String("address", addr))
		verified = true
	}

	// count certs, if the leaf is verified then all certs are verified, otherwise none are
	certs := conn.ConnectionState().PeerCertificates

	ci.CertChain = certs
	ci.CerificateVerified = slices.Repeat([]bool{verified}, len(certs))

	return
}

// readFiles reads the specified files and extracts certificate
// information, returning a slice of certInfo. Errors are stored in
// ci.Error for each file, and processing continues for all files even
// if some have errors. The roots parameter is used for verifying
// certificate chains.
func readFiles(paths []string, roots *x509.CertPool) (ci []certInfo) {
	ci = make([]certInfo, len(paths))

	// paths is a list of files to examine, pre-resolved by the
	// shell so we don't do any wildcard processing
	//
	// extensions are only checked for .db/.pfx/.p12 files, others are
	// assumed to be PEM and may contain certificates, private keys
	// or both
	//
	// each PEM file can contain multiple entries and they are
	// listed in order
	for i, p := range paths {
		if err := readFile(p, &ci[i]); err != nil {
			continue
		}
	}

	// verify cert chains
	for i := range ci {
		for n, c := range ci[i].CertChain {
			opts := x509.VerifyOptions{
				Roots:         roots, // nil means use system roots
				Intermediates: x509.NewCertPool(),
			}
			for j, ic := range ci[i].CertChain {
				if j == n {
					continue
				}
				opts.Intermediates.AddCert(ic)
			}
			if _, err := c.Verify(opts); err != nil {
				ci[i].CerificateVerified = append(ci[i].CerificateVerified, false)
			} else {
				ci[i].CerificateVerified = append(ci[i].CerificateVerified, true)
			}
		}
	}

	return
}

// readFile reads the specified file and extracts certificate
// information into the provided certInfo struct. The file can be a PEM
// file containing certificates and/or private keys, a PKCS#12/PFX file,
// or a Java keystore. Errors are stored in ci.Error and returned by the
// function. The roots parameter is used for verifying certificate
// chains when reading from a connection, but not for files since we
// want to report the verification status of each certificate
// separately.
func readFile(p string, ci *certInfo) (err error) {
	ci.Path, err = filepath.Abs(p)
	if err != nil {
		log.Error("unable to get absolute path", slog.Any("error", err), slog.String("file", p))
		return
	}
	// ci.Contents = certContents{}

	contents, err2 := os.ReadFile(p)
	if err2 != nil {
		log.Error("unable to read file", slog.Any("error", err2), slog.String("file", p))
		return
	}

	// treat a cacerts file specially, setting the password to
	// "changeit" and only reading trusted certificate entries
	if path.Base(p) == "cacerts" {
		k, err := certs.ReadKeystore(geneos.LOCAL, p, config.Secret("changeit"))
		if err != nil {
			log.Error("unable to read Java keystore", slog.Any("error", err), slog.String("file", p))
			return err
		}
		for _, alias := range k.Aliases() {
			entry, err := k.GetTrustedCertificateEntry(alias)
			if err != nil {
				log.Error("unable to get certificate entry from Java truststore", slog.Any("error", err), slog.String("alias", alias))
				return err
			}
			cert, err := x509.ParseCertificate(entry.Certificate.Content)
			if err != nil {
				log.Error("unable to parse certificate from Java truststore", slog.Any("error", err), slog.String("alias", alias))
				return err
			}
			ci.Alias = append(ci.Alias, alias)
			ci.CertChain = append(ci.CertChain, cert)
		}
		return nil
	}

	r, err2 := os.Open(p)
	if err2 != nil {
		log.Error("unable to open file", slog.Any("error", err2), slog.String("file", p))
		return err2
	}

	magic := make([]byte, 4)
	_, err2 = r.Read(magic)
	r.Close() // close regardless of read success
	if err2 != nil && err2 != io.EOF {
		log.Error("unable to read file", slog.Any("error", err2), slog.String("file", p))
		return err2
	}

	if bytes.Equal(magic, []byte{0xFE, 0xED, 0xFE, 0xED}) {
		log.Debug("Java keystore magic number found", slog.String("file", p))
		if len(infoCmdPassword) == 0 {
			infoCmdPassword, err = config.ReadPasswordInput(false, 0, "Password for keystore file "+p)
			if err != nil {
				log.Error("Failed to read password", slog.Any("error", err))
				os.Exit(1)
				// return err
			}
			defer clear(infoCmdPassword)
		}
		k, err := certs.ReadKeystore(geneos.LOCAL, p, infoCmdPassword)
		if err != nil {
			log.Error("unable to read Java keystore", slog.Any("error", err), slog.String("file", p))
			return err
		}
		for _, alias := range k.Aliases() {
			switch {
			case k.IsPrivateKeyEntry(alias):
				pke, err := k.GetPrivateKeyEntry(alias, infoCmdPassword)
				if err != nil {
					log.Error("unable to get private key entry from Java keystore", slog.Any("error", err), slog.String("alias", alias))
					return err
				}
				ci.PrivateKeys = append(ci.PrivateKeys, pke.PrivateKey)

				chain := pke.CertificateChain
				for n, cert := range chain {
					parsedCert, err := x509.ParseCertificate(cert.Content)
					if err != nil {
						log.Error("unable to parse certificate from Java keystore", slog.Any("error", err), slog.String("alias", alias), slog.Int("cert", n))
						return err
					}
					ci.Alias = append(ci.Alias, alias+"["+strconv.FormatInt(int64(n+1), 10)+"]")
					ci.CertChain = append(ci.CertChain, parsedCert)
				}
			case k.IsTrustedCertificateEntry(alias):
				entry, err := k.GetTrustedCertificateEntry(alias)
				if err != nil {
					log.Error("unable to get CA certificate entry from Java keystore", slog.Any("error", err), slog.String("alias", alias))
					return err
				}
				cert, err := x509.ParseCertificate(entry.Certificate.Content)
				if err != nil {
					log.Error("unable to parse certificate from Java keystore", slog.Any("error", err), slog.String("alias", alias))
					return err
				}
				if slices.Contains(ci.Alias, alias) {
					return err
				}
				ci.Alias = append(ci.Alias, alias)
				ci.CertChain = append(ci.CertChain, cert)
			default:
				return err
			}
		}
		return err
	}

	ext := strings.ToLower(path.Ext(p))
	if ext == ".pfx" || ext == ".p12" {
		if len(infoCmdPassword) == 0 {
			infoCmdPassword, err = config.ReadPasswordInput(false, 0, "Password (for file "+p+")")
			if err != nil {
				return
			}
			defer clear(infoCmdPassword)
		}

		key, c, chain, err := pkcs12.DecodeChain(contents, string(infoCmdPassword))
		if err != nil {
			log.Error("unable to decode PKCS#12 file - is the password correct?", slog.Any("error", err), slog.String("file", p))
			return err
		}
		ci.CertChain = append(ci.CertChain, c)
		ci.CertChain = append(ci.CertChain, chain...)

		pk, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			log.Error("unable to marshal private key from PKCS#12 file", slog.Any("error", err), slog.String("file", p))
			return err
		}
		mpk := certs.PrivateKey(pk)
		if !certs.CheckKeyMatch(mpk, c) {
			log.Warn("private key does not match certificate in PKCS#12 file", slog.String("file", p))
		} else {
			log.Debug("added private key from PKCS#12 file to list for matching with certificates", slog.String("file", p))
		}
		ci.PrivateKeys = append(ci.PrivateKeys, mpk)
		return err
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
			ci.CertChain = append(ci.CertChain, c)
		case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
			// save all private keys for later matching
			ci.PrivateKeys = append(ci.PrivateKeys, block.Bytes)
		default:
			err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
		}
		contents = rest
	}

	return
}

func infoMapString[T, V any](ts []T, fn func(T) V) string {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	if len(result) == 0 {
		return "None"
	}
	stringsResult := make([]string, len(result))
	for i, r := range result {
		stringsResult[i] = fmt.Sprintf("%v", r)
	}
	return strings.Join(stringsResult, " ")
}

func keyUsageToString(ku x509.KeyUsage) string {
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
	keys := slices.Sorted(maps.Keys(usageMap))
	for _, k := range keys {
		if ku&k != 0 {
			usages = append(usages, usageMap[k])
		}
	}
	if len(usages) == 0 {
		return "None"
	}
	return strings.Join(usages, " ")
}

func extKeyUsageToString(eku []x509.ExtKeyUsage) string {
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
	keys := slices.Sorted(maps.Keys(usageMap))
	for _, k := range keys {
		if containsExtKeyUsage(eku, k) {
			usages = append(usages, usageMap[k])
		}
	}
	if len(usages) == 0 {
		return "None"
	}
	return strings.Join(usages, " ")
}

func containsExtKeyUsage(usages []x509.ExtKeyUsage, usage x509.ExtKeyUsage) bool {
	return slices.Contains(usages, usage)
}

func (c *infoCmdConnectsType) String() string {
	return strings.Join(*c, ", ")
}

func (c *infoCmdConnectsType) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func (c *infoCmdConnectsType) Type() string {
	return "connect"
}

func (r *infoCmdRootsType) String() string {
	return strings.Join(*r, ", ")
}

func (r *infoCmdRootsType) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func (r *infoCmdRootsType) Type() string {
	return "roots"
}
