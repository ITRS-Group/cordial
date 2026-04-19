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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/awnumar/memguard"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

var migrateCmdCheck bool

func init() {
	migrateCmd.Flags().BoolVarP(&migrateCmdCheck, "check", "c", false, "check if instance is already migrated")

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
		cmd.CmdGlobal:                "true",
		cmd.CmdRequireHome:           "true",
		cmd.CmdWildcardNames:         "true",
		cmd.CmdNonInstanceArgsError:  "true",
		cmd.CmdAllInstancesMustMatch: "true",
	},
	RunE: func(command *cobra.Command, _ []string) (err error) {
		ct, names, _, err := cmd.FetchArgs(command)
		if err != nil {
			return
		}
		instance.Do(geneos.GetHost(cmd.Hostname), ct, names, migrateInstanceTLS).Report(os.Stdout)
		return
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
	var truststorePath, keystorePath string
	var truststorePassword, keystorePassword *config.Secret

	resp = responses.NewResponse(i)

	cf := i.Config()
	h := i.Host()

	// first check if already the instance already appears to have been
	// migrated
	if cf.IsSet(cf.Join("tls", "certificate")) {
		resp.Completed = append(resp.Completed, "already migrated ✔️")
		return
	}

	if migrateCmdCheck {
		resp.Completed = append(resp.Completed, "not migrated ❌")
		return
	}

	// For components that use Java keystore/truststores, use
	// those for migration and extraction of certs/keys
	switch {
	case i.Type().IsA("sso-agent"):
		ssoConf := config.New()
		if err := ssoConf.MergeHOCONFile(instance.Abs(i, "conf/sso-agent.conf")); err != nil {
			return
		}

		truststorePath = config.Get[string](ssoConf, ssoConf.Join("server", "trust_store", "location"))
		truststorePassword = config.Get[*config.Secret](ssoConf, ssoConf.Join("server", "trust_store", "password"))
		keystorePath = config.Get[string](ssoConf, ssoConf.Join("server", "key_store", "location"))
		keystorePassword = config.Get[*config.Secret](ssoConf, ssoConf.Join("server", "key_store", "password"))
	case i.Type().IsA("webserver"):
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
	default:
		// for other instance types, we don't expect to find
		// keystore/truststore parameters, do nothing
	}

	// extract trusted certs from truststore and update the global
	// ca-bundle files. if this fails, continue with the migration
	// as we may just need to rebuild the trust chain
	if truststorePath != "" {
		updated, err := certs.UpdateCACertsFileFromTrustStore(h, instance.Abs(i, truststorePath), truststorePassword, geneos.PathToCABundle(h))
		if err != nil {
			resp.Err = fmt.Errorf("updating ca-bundle from truststore: %w", err)
		} else if updated {
			resp.Completed = append(resp.Completed, "updated ca-bundle from truststore")
		}
	}

	// extract first private key entry and its cert chain from
	// keystore and write to instance cert and key files. If this
	// fails, continue with the migration as we may just need to
	// rebuild the keystore later (in instance Rebuild())
	if keystorePath != "" {
		k, err := certs.ReadKeystore(h, instance.Abs(i, keystorePath), keystorePassword)
		if err != nil {
			resp.Err = fmt.Errorf("reading keystore: %w", err)
			return
		} else {
			for _, alias := range k.Aliases() {
				var certChain []*x509.Certificate

				// leave the private key alone
				if i.Type().IsA("sso-agent") && alias == "ssokey" {
					continue
				}

				key, err := k.GetPrivateKeyEntry(alias, keystorePassword.Bytes())
				if err != nil {
					continue
				}

				for _, certData := range key.CertificateChain {
					cert, err := x509.ParseCertificate(certData.Content)
					if err != nil {
						continue
					}
					certChain = append(certChain, cert)
				}

				if len(certChain) == 0 {
					continue
				}

				if err = instance.WriteCertificateAndKey(i, memguard.NewEnclave(key.PrivateKey), certChain...); err != nil {
					resp.Err = fmt.Errorf("writing certificate chain and private key from keystore: %w", err)
					break
				}

				resp.Completed = append(resp.Completed, "extracted first certificate chain and private key from keystore")
				break // only first key entry
			}
		}
	}

	// some instances may already have multiple certificates in the
	// primary file, before migration, or they have just been created
	// from a Java keystore above
	instanceCertChain, err := instance.ReadCertificates(i)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		resp.Err = err
		return
	}
	if len(instanceCertChain) == 0 {
		resp.Err = fmt.Errorf("no valid instance certificate found")
		return
	}

	if cf.IsSet("certchain") {
		chain, err := certs.ReadCertificates(h, config.Get[string](cf, "certchain"))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			resp.Err = err
			return
		}
		instanceCertChain = append(instanceCertChain, chain...)
	}

	if !slices.ContainsFunc(instanceCertChain, certs.IsValidRootCA) {
		// try to make sure we have a full chain by adding root cert
		rootCert, _, err := geneos.ReadRootCertificateAndKey()
		if err != nil {
			resp.Err = fmt.Errorf("cannot read root certificate: %w", err)
			return
		}

		instanceCertChain = append(instanceCertChain, rootCert)
	}

	// parse the cert chain to get leaf, intermediates and root
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

	// write leaf and trust chain to certificate file - this updates
	// instance parameters for certificate
	err = instance.WriteCertificateAndKey(i, nil, append([]*x509.Certificate{leaf}, intermediates...)...)
	if err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "wrote fullchain to instance certificate file")

	// update instance parameters to new layout
	if pk := config.Get[string](cf, "privatekey"); pk != "" {
		// this may have already been done above in webserver/sso-agent
		config.Set(cf, "privatekey", "")
		config.Set(cf, cf.Join("tls", "privatekey"), pk)
	}
	config.Set(cf, cf.Join("tls", "ca-bundle"), geneos.PathToCABundlePEM(i.Host()))

	if cf.IsSet("use-chain") && !config.Get[bool](cf, "use-chain") {
		config.Set(cf, cf.Join("tls", "verify"), false)
	}

	config.Set(cf, "certchain", "")
	config.Set(cf, "use-chain", "")
	config.Set(cf, "truststore", "")
	config.Set(cf, "truststore-password", "")

	if err = instance.SaveConfig(i); err != nil {
		resp.Err = err
		return
	}
	resp.Completed = append(resp.Completed, "updated instance configuration")

	return
}
