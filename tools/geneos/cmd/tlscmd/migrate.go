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
	"crypto/x509"
	_ "embed"
	"fmt"
	"os"

	"github.com/awnumar/memguard"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

func init() {
	tlsCmd.AddCommand(migrateCmd)
}

//go:embed _docs/migrate.md
var migrateCmdDescription string

var migrateCmd = &cobra.Command{
	Use:          "migrate [TYPE] [NAME...]",
	Short:        "Migrate certificates to the new TLS layout",
	Long:         migrateCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:        "true",
		cmd.CmdRequireHome:   "true",
		cmd.CmdWildcardNames: "true",
	},
	Run: func(command *cobra.Command, _ []string) {
		ct, names := cmd.ParseTypeNames(command)
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, migrateInstanceTLS).Report(os.Stdout)
	},
}

// migrateInstanceTLS migrates the TLS configuration of an instance to
// the new layout
//
// If a `tls::certificate` parameter is already set then the instance is
// assumed to have already been migrated and no action is taken.
//
// For Java keystore/truststore based instances (sso-agent, webserver)
// with their own configurations files referring to keystores and
// truststores, the first private key entry and its certificate chain is
// extracted and written to the instance certificate and private key
// files. Trusted certificates from the truststore are added to the
// local ca-bundle file.
//
// For all instances, the existing instance certificate file is read
// (and certchain file if set) and the full certificate chain is built.
// If the root CA certificate is not present in the chain it is added
// from the local root certificate. The ca-bundle file is updated with
// the root and the full chain (minus root) is written to the instance
// certificate file.
//
// Finally, the instance configuration is updated to use the new TLS
// parameters and old parameters are cleared.
func migrateInstanceTLS(i geneos.Instance, _ ...any) (resp *responses.Response) {
	resp = responses.NewResponse(i)

	cf := i.Config()
	h := i.Host()

	// first check if already migrated
	if cf.IsSet(cf.Join("tls", "certificate")) {
		resp.Completed = append(resp.Completed, "instance already migrated")
		return
	}

	// Java keystore/truststore migration
	if i.Type().IsA("sso-agent", "webserver") {
		var truststorePath, keystorePath string
		var truststorePassword, keystorePassword *config.Plaintext

		if i.Type().IsA("sso-agent") {
			ssoconf := config.New()
			if err := ssoconf.MergeHOCONFile(instance.Abs(i, "conf/sso-agent.conf")); err != nil {
				return
			}

			truststorePath = ssoconf.GetString(config.Join("server", "trust_store", "location"))
			truststorePassword = ssoconf.GetPassword(config.Join("server", "trust_store", "password"))
			keystorePath = ssoconf.GetString(config.Join("server", "key_store", "location"))
			keystorePassword = ssoconf.GetPassword(config.Join("server", "key_store", "password"))
		} else if i.Type().IsA("webserver") {
			spPath := instance.Abs(i, "config/security.properties")

			// load the security.properties file, update the port and use the keystore values later
			sp, err := instance.ReadKVConfig(h, spPath)
			if err != nil {
				return nil
			}

			truststorePath = sp["trustStore"]
			truststorePassword = cf.ExpandToPassword(sp["trustStorePassword"])

			keystorePath = sp["keyStore"]
			keystorePassword = cf.ExpandToPassword(sp["keyStorePassword"])
		}

		// extract trusted certs from truststore
		if truststorePath != "" {
			k, err := certs.ReadKeystore(h, instance.Abs(i, truststorePath), truststorePassword)
			if err != nil {
				resp.Err = fmt.Errorf("reading truststore: %w", err)
				return
			}
			var roots []*x509.Certificate
			for _, alias := range k.Aliases() {
				if c, err := k.GetTrustedCertificateEntry(alias); err == nil {
					cert, err := x509.ParseCertificate(c.Certificate.Content)
					if err == nil {
						roots = append(roots, cert)
					}
				}
			}
			if updated, _ := certs.UpdateCACertsFiles(h, geneos.PathToCABundle(h), roots...); updated {
				resp.Completed = append(resp.Completed, "updated ca-bundle from truststore")
			}
		}

		if keystorePath != "" {
			k, err := certs.ReadKeystore(h, instance.Abs(i, keystorePath), keystorePassword)
			if err != nil {
				resp.Err = fmt.Errorf("reading keystore: %w", err)
				return
			}
			for _, alias := range k.Aliases() {
				// leave the private key alone
				if i.Type().IsA("sso-agent") && alias == "ssokey" {
					continue
				}

				privateKeyEntry, err := k.GetPrivateKeyEntry(alias, keystorePassword.Bytes())
				if err != nil {
					continue
				}
				var certChain []*x509.Certificate
				for _, certData := range privateKeyEntry.CertificateChain {
					cert, err := x509.ParseCertificate(certData.Content)
					if err != nil {
						continue
					}
					certChain = append(certChain, cert)
				}
				if len(certChain) == 0 {
					continue
				}
				// write private key
				if err = instance.WritePrivateKey(i, memguard.NewEnclave(privateKeyEntry.PrivateKey)); err != nil {
					resp.Err = fmt.Errorf("writing private key from keystore: %w", err)
					return
				}
				// write certificates
				if err = instance.WriteCertificates(i, certChain); err != nil {
					resp.Err = fmt.Errorf("writing certificates from keystore: %w", err)
					return
				}
				resp.Completed = append(resp.Completed, "extracted first certificate chain and private key from keystore")
				break // only first key entry
			}
		}
	}

	// some instances may already have multiple certificates in the
	// primary file, before migration
	instanceCertChain, err := instance.ReadCertificates(i)
	if err != nil && !os.IsNotExist(err) {
		resp.Err = err
		return
	}
	if len(instanceCertChain) == 0 {
		resp.Err = fmt.Errorf("no valid instance certificate found")
		return
	}

	if cf.IsSet("certchain") {
		chain, err := certs.ReadCertificates(h, cf.GetString("certchain"))
		if err != nil && !os.IsNotExist(err) {
			resp.Err = err
			return
		}
		instanceCertChain = append(instanceCertChain, chain...)
	}

	var haveRoot bool
	for _, c := range instanceCertChain {
		// prefer the root cert already in the chain
		if certs.IsValidRootCA(c) {
			haveRoot = true
			break
		}
	}

	if !haveRoot {
		// try to make sure we have a full chain by adding root cert
		rootCert, _, err := geneos.ReadRootCertificate()
		if err != nil {
			resp.Err = fmt.Errorf("cannot read root certificate: %w", err)
			return
		}

		instanceCertChain = append(instanceCertChain, rootCert)
	}

	leaf, intermediates, root, err := certs.ParseCertChain(instanceCertChain...)
	if err != nil {
		resp.Err = err
		return
	}

	// update ca-bundle file
	updated, err := certs.UpdateCACertsFiles(h, geneos.PathToCABundle(h), root)
	if err != nil {
		resp.Err = err
		return
	}
	if updated {
		resp.Completed = append(resp.Completed, "updated ca-bundle")
	}

	// write fullchain to certificate file - this updates instance parameters for certificate
	err = instance.WriteCertificates(i, append([]*x509.Certificate{leaf}, intermediates...))
	if err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "wrote fullchain to instance certificate file")

	// update instance parameters to new layout
	cf.Set(cf.Join("tls", "privatekey"), cf.GetString("privatekey"))
	cf.Set(cf.Join("tls", "ca-bundle"), geneos.PathToCABundlePEM(i.Host()))

	if !cf.GetBool("use-chain") {
		cf.Set(cf.Join("tls", "verify"), false)
	}

	// "certificate" is cleared by WriteCertificates above
	cf.Set("privatekey", "")
	cf.Set("certchain", "")
	cf.Set("use-chain", "")
	cf.Set("truststore", "")
	cf.Set("truststore-password", "")

	if err = instance.SaveConfig(i); err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "updated instance configuration")

	return
}
