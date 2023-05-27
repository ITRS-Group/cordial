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
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var tlsNewCmdNamed, tlsNewCmdDir string

func init() {
	tlsCmd.AddCommand(tlsNewCmd)

	tlsNewCmd.Flags().StringVarP(&tlsNewCmdNamed, "named", "n", "", "Create a named certificate and key in current directory. `CN` is the name used.")
	tlsNewCmd.Flags().StringVarP(&tlsNewCmdDir, "dir", "D", ".", "Write to directory `DIR` for named certificate and key.")
}

//go:embed _docs/new.md
var tlsNewCmdDescription string

var tlsNewCmd = &cobra.Command{
	Use:          "new [TYPE] [NAME...]",
	Short:        "Create new certificates",
	Long:         tlsNewCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		"wildcard":     "true",
		"needshomedir": "true",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		if tlsNewCmdNamed != "" {
			hostname, _ := os.Hostname()
			if cmd.Hostname != "all" {
				hostname = cmd.Hostname
			}
			return CreateCert(tlsNewCmdDir, tlsNewCmdNamed, hostname)
		}
		ct, args, params := cmd.CmdArgsParams(command)
		return instance.ForAll(ct, cmd.Hostname, newInstanceCert, args, params)
	},
}

func newInstanceCert(c geneos.Instance, _ []string) (err error) {
	return instance.CreateCert(c)
}

// CreateCert creates a new certificate
//
// this also creates a new private key
//
// skip if certificate exists and is valid
func CreateCert(dir string, cn string, hostname string) (err error) {
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	expires := time.Now().AddDate(1, 0, 0).Truncate(24 * time.Hour)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       []string{hostname},
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	intrCert, err := config.ReadCert(geneos.LOCAL, filepath.Join(config.AppConfigDir(), geneos.SigningCertFile+".pem"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	intrKey, err := config.ReadKey(geneos.LOCAL, filepath.Join(config.AppConfigDir(), geneos.SigningCertFile+".key"))
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	cert, key, err := config.CreateCertKey(&template, intrCert, intrKey, nil)
	if err != nil {
		return
	}

	c := filepath.Join(dir, strings.ReplaceAll(cn, " ", "-"))

	if err = config.WriteCert(geneos.LOCAL, c+".pem", cert); err != nil {
		return
	}

	if err = config.WriteKey(geneos.LOCAL, c+".key", key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s (expires %s)\n", c, expires.UTC())

	return
}
