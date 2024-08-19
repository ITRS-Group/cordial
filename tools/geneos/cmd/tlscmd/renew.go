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
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

var renewCmdDays int

func init() {
	tlsCmd.AddCommand(renewCmd)

	renewCmd.Flags().IntVarP(&renewCmdDays, "days", "D", 365, "Certificate duration in days")
}

//go:embed _docs/renew.md
var renewCmdDescription string

var renewCmd = &cobra.Command{
	Use:          "renew [TYPE] [NAME...]",
	Short:        "Renew instance certificates",
	Long:         renewCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, renewInstanceCert).Write(os.Stdout)
	},
}

// renew an instance certificate, use private key if it exists
func renewInstanceCert(i geneos.Instance, _ ...any) (resp *instance.Response) {
	resp = instance.NewResponse(i)

	confDir := config.AppConfigDir()
	if confDir == "" {
		resp.Err = config.ErrNoUserConfigDir
		return
	}

	hostname, _ := os.Hostname()
	if !i.Host().IsLocal() {
		hostname = i.Host().GetString("hostname")
	}

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	duration := 365 * 24 * time.Hour
	if renewCmdDays != 0 {
		duration = 24 * time.Hour * time.Duration(renewCmdDays)
	}
	expires := time.Now().Add(duration)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("geneos %s %s", i.Type(), i.Name()),
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       []string{hostname},
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	signingCert, _, err := geneos.ReadSigningCert()
	resp.Err = err
	if resp.Err != nil {
		return
	}
	signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(confDir, geneos.SigningCertBasename+".key"))
	resp.Err = err
	if resp.Err != nil {
		return
	}

	// read existing key or create a new one
	existingKey, _ := instance.ReadKey(i)
	cert, key, err := config.CreateCertificateAndKey(&template, signingCert, signingKey, existingKey)
	resp.Err = err
	if resp.Err != nil {
		return
	}

	if resp.Err = instance.WriteCert(i, cert); resp.Err != nil {
		return
	}

	if existingKey == nil {
		if resp.Err = instance.WriteKey(i, key); resp.Err != nil {
			return
		}
	}

	// root cert optional to create instance specific chain file
	rootCert, _, _ := geneos.ReadRootCert()
	if rootCert == nil {
		i.Config().SetString("certchain", i.Host().PathTo("tls", geneos.ChainCertFile))
	} else {
		chainfile := instance.PathOf(i, "certchain")
		if chainfile == "" {
			chainfile = path.Join(i.Home(), "chain.pem")
			i.Config().SetString("certchain", chainfile, config.Replace("home"))
		}

		if resp.Err = config.WriteCertChain(i.Host(), chainfile, signingCert, rootCert); resp.Err != nil {
			return
		}
	}

	if resp.Err = instance.SaveConfig(i); resp.Err != nil {
		return
	}

	resp.Completed = append(resp.Completed, fmt.Sprintf("certificate renewed (expires %s)", expires.UTC().Format(time.RFC3339)))
	return
}
