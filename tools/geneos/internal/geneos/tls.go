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

package geneos

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// RootCABasename is the file base name for the root certificate authority
// created with the TLS commands
var RootCABasename = "rootCA"

// SigningCertBasename is the file base name for the signing certificate
// created with the TLS commands
var SigningCertBasename string

// ChainCertFile the is file name (including extension, as this does not
// need to be used for keys) for the consolidated chain file used to
// verify instance certificates
var ChainCertFile string

const (
	// TrustedRootsFilename is the file name for the trusted roots file
	// used by Geneos components to verify peer certificates. This file
	// is located in the geneos home directory on each hoist under
	// `tls/`
	TrustedRootsFilename = "trusted-roots.pem"
)

func TrustedRootsPath(h *Host) string {
	return h.PathTo("tls", TrustedRootsFilename)
}

// ReadRootCertificate reads the root certificate from the user's app
// config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory. If verify is true then the certificate is verified
// against itself as a root and if it fails an error is returned.
func ReadRootCertificate() (root *x509.Certificate, file string, err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}

	// move the root certificate to the user app config directory
	log.Debug().Msgf("migrating root certificate from %s to %s", LOCAL.PathTo("tls", RootCABasename+".pem"), confDir)
	file = config.MigrateFile(host.Localhost, confDir, LOCAL.PathTo("tls", RootCABasename+".pem"))
	if file == "" {
		err = fmt.Errorf("%w: root certificate file %s not found in %s", os.ErrNotExist, RootCABasename+".pem", confDir)
		log.Debug().Err(err).Msgf("failed to migrate root certificate from %s", LOCAL.PathTo("tls", RootCABasename+".pem"))
		return
	}
	log.Debug().Msgf("reading root certificate %s", file)

	// speculatively promote the key file, but do not fail if it does
	// not exist. this is because the root certificate is self-signed and
	// does not need a key to verify itself.
	config.MigrateFile(host.Localhost, confDir, LOCAL.PathTo("tls", RootCABasename+".key"))
	root, err = certs.ReadCertificate(LOCAL, file)
	if err != nil {
		return
	}
	if !certs.ValidRootCA(root) {
		err = errors.New("certificate not valid as a root CA")
	}
	return
}

// ReadSigningCertificate reads the signing certificate from the user's app
// config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory. If verify is true then the signing certificate is
// checked and verified against the default root certificate.
func ReadSigningCertificate() (signer *x509.Certificate, file string, err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}
	// move the signing certificate to the user app config directory
	file = config.MigrateFile(host.Localhost, confDir, LOCAL.PathTo("tls", SigningCertBasename+".pem"))
	if file == "" {
		err = fmt.Errorf("%w: signing certificate file %s not found in %s", os.ErrNotExist, SigningCertBasename+".pem", confDir)
		return
	}
	log.Debug().Msgf("reading signing certificate %s", file)

	// speculatively promote the key file, but do not fail if it does
	// not exist.
	config.MigrateFile(host.Localhost, confDir, LOCAL.PathTo("tls", SigningCertBasename+".key"))
	signer, err = certs.ReadCertificate(LOCAL, file)
	if err != nil {
		return
	}

	if !certs.ValidSigningCA(signer) {
		err = errors.New("certificate not valid as a signing certificate")
		return
	}

	// verify against root CA
	var root *x509.Certificate
	root, _, err = ReadRootCertificate()
	if err != nil {
		return
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	_, err = signer.Verify(x509.VerifyOptions{
		Roots: roots,
	})
	return
}

func ParseCertificatesAndKeys(data ...[]byte) (certificates []*x509.Certificate, keys []*memguard.Enclave, err error) {
	if len(data) == 0 {
		return
	}

	// store them unsorted for now
	var ks []*memguard.Enclave
	var leaves, intermediates, roots []*x509.Certificate

	for _, b := range data {
		pembytes := b
		for {
			p, rest := pem.Decode(pembytes)
			if p == nil {
				break
			}

			switch p.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(p.Bytes)
				if err != nil {
					return
				}
				if certs.ValidRootCA(c) {
					roots = append(roots, c)
				} else if certs.ValidSigningCA(c) {
					intermediates = append(intermediates, c)
				} else {
					leaves = append(leaves, c)
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save first private key found
				ks = append(ks, memguard.NewEnclave(p.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s", p.Type)
				return
			}
			pembytes = rest
		}
	}

	ip := x509.NewCertPool()
	for _, c := range intermediates {
		ip.AddCert(c)
	}
	rp := x509.NewCertPool()
	for _, c := range roots {
		rp.AddCert(c)
	}
	// sort certs then keys
	vopts := x509.VerifyOptions{
		Intermediates: ip,
		Roots:         rp,
	}
	for _, c := range leaves {
		_, err = c.Verify(vopts)
		if err != nil {
			continue
		}
		certificates = append(certificates, c)
	}
	for _, c := range intermediates {
		_, err = c.Verify(vopts)
		if err != nil {
			continue
		}
		certificates = append(certificates, c)
	}
	for _, c := range roots {
		_, err = c.Verify(vopts)
		if err != nil {
			continue
		}
		certificates = append(certificates, c)
	}

	return
}

// DecomposePEM parses PEM formatted data and extracts a leaf
// certificate, any CA certs as a chain and a private key as a DER
// encoded *memguard.Enclave.
//
// If no leaf certificate is found, but CA certificates are found, only
// return the chain.
//
// The key is matched either to the leaf certificate or, if no leaf,
// then the first member of the chain that matches and private key in
// the bundle.
func DecomposePEM(data ...string) (cert *x509.Certificate, key *memguard.Enclave, chain []*x509.Certificate, err error) {
	var certSlice []*x509.Certificate
	var leaf *x509.Certificate
	var keys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no PEM data process")
		return
	}

	for _, pemstring := range data {
		pembytes := []byte(pemstring)
		for {
			p, rest := pem.Decode(pembytes)
			if p == nil {
				break
			}

			switch p.Type {
			case "CERTIFICATE":
				var c *x509.Certificate
				c, err = x509.ParseCertificate(p.Bytes)
				if err != nil {
					return
				}
				if c.IsCA {
					certSlice = append(certSlice, c)
				} else if leaf == nil {
					// save first leaf
					leaf = c
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				keys = append(keys, memguard.NewEnclave(p.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s", p.Type)
				return
			}
			pembytes = rest
		}
	}

	if leaf == nil && len(certSlice) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}

	// if we got this far then we can start setting returns
	cert = leaf
	chain = certSlice

	// match key if we have a leaf cert
	if cert != nil {
		if i := certs.IndexPrivateKey(keys, cert); i != -1 {
			key = keys[i]
		}
	} else {
		// no leaf, try to match first cert in chain
		for _, c := range chain {
			if i := certs.IndexPrivateKey(keys, c); i != -1 {
				key = keys[i]
				break
			}
		}
	}

	err = nil
	return
}

// TLSImportBundle processes a PEM formatted signingBundle from a file
// or an embedded string with either an included private key and chain
// or separately specified in the same way.
func TLSImportBundle(signingBundleSource, privateKeySource, chainSource string) (err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}

	// speculatively create user config directory. permissions do not
	// need to be restrictive
	err = LOCAL.MkdirAll(confDir, 0775)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	signingBundle, err := config.ReadInputPEMString(signingBundleSource, "signing certificate(s)")
	if err != nil {
		return err
	}

	privateKey, err := config.ReadInputPEMString(privateKeySource, "signing key")
	if err != nil {
		return err
	}

	_, key, chain, err := DecomposePEM(signingBundle, privateKey)
	if err != nil {
		return err
	}
	cert := chain[0]

	// basic validation
	if !cert.BasicConstraintsValid || !cert.IsCA || key == nil {
		return errors.New("no signing certificate with private key found in bundle")

	}

	// write root cert, but only if it's the only other cert in the
	// chain (the chain will contain both the signing cert and root, as
	// there is no leaf cert) and it's self-signed. overwrite any
	// existing root.
	if len(chain) == 2 {
		root := chain[1]
		rootCA := path.Join(confDir, RootCABasename+".pem")

		if bytes.Equal(root.RawIssuer, root.RawSubject) && root.IsCA {
			// if st, err := os.Stat(rootCA); !errors.Is(err, os.ErrNotExist) {
			// 	return errors.New("rootCA.pem is already present in user config directory, will not overwrite")
			// }
			if err = certs.WriteCertificates(LOCAL, rootCA, root); err != nil {
				return err
			}
			fmt.Printf("%s root certificate written to %s\n", cordial.ExecutableName(), rootCA)
		}
	}

	if err = certs.WriteCertificates(LOCAL, path.Join(confDir, SigningCertBasename+".pem"), cert); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+".pem"))

	if err = certs.WritePrivateKey(LOCAL, path.Join(confDir, SigningCertBasename+".key"), key); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate key written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+".key"))

	if chainSource != "" {
		b, err := os.ReadFile(chainSource)
		if err != nil {
			log.Error().Err(err).Msg("")
			return err
		}
		_, _, chain, err = DecomposePEM(string(b))
		if err != nil {
			return err
		}
		if err = WriteChainLocal(chain); err != nil {
			return err
		}
	} else if len(chain) > 0 {
		if err = WriteChainLocal(chain); err != nil {
			return err
		}
	}
	fmt.Printf("%s certificate chain written to %s\n", cordial.ExecutableName(), path.Join(LOCAL.PathTo("tls"), ChainCertFile))
	return err
}

func WriteChainLocal(chain []*x509.Certificate) (err error) {
	if len(chain) == 0 {
		return
	}
	tlsPath := LOCAL.PathTo("tls")
	if err = LOCAL.MkdirAll(tlsPath, 0775); err != nil {
		return err
	}
	if err = certs.WriteCertificates(LOCAL, path.Join(tlsPath, ChainCertFile), chain...); err != nil {
		return err
	}
	return
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
//
// This is also called from `init`
func TLSInit(overwrite bool, keytype certs.KeyType) (err error) {
	if !overwrite {
		if _, _, err := ReadRootCertificate(); err == nil {
			// root cert already exists
			fmt.Printf("root certificate already exists, skipping TLS initialisation\n")
			return nil
		}
		if _, _, err := ReadSigningCertificate(); err == nil {
			// signing cert already exists
			fmt.Printf("signing certificate already exists, skipping TLS initialisation\n")
			return nil
		}
	}

	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
	}

	// directory permissions do not need to be restrictive
	if err = LOCAL.MkdirAll(confDir, 0775); err != nil {
		return err
	}

	if err := certs.CreateRootCert(
		path.Join(confDir, RootCABasename),
		cordial.ExecutableName()+" root certificate",
		keytype); err != nil {
		return err
	}
	fmt.Printf("CA certificate created for %s\n", RootCABasename)

	rootCert, _, err := ReadRootCertificate()
	if err != nil {
		return err
	}
	_, err = certs.UpdateRootCertsFile(LOCAL, TrustedRootsPath(LOCAL), rootCert)
	if err != nil {
		return err
	}

	if err := certs.CreateSigningCert(
		path.Join(confDir, SigningCertBasename),
		path.Join(confDir, RootCABasename),
		cordial.ExecutableName()+" intermediate certificate",
	); err != nil {
		return err
	}
	fmt.Printf("Signing certificate created for %s\n", SigningCertBasename)

	// sync if geneos root exists
	if d, err := os.Stat(LocalRoot()); err == nil && d.IsDir() {
		return TLSSync()
	}
	return nil
}

// TLSSync merges and updates the `TrustedRootsFilename` file on all remote hosts.
func TLSSync() (err error) {
	allRoots := []*x509.Certificate{}
	allHosts := append([]*Host{LOCAL}, RemoteHosts(false)...)
	for _, h := range allHosts {
		if certSlice, err := certs.ReadCertificates(h, h.PathTo("tls", TrustedRootsFilename)); err == nil {
			allRoots = append(allRoots, certSlice...)
		}
	}

	for _, h := range allHosts {
		hostname := h.Hostname()
		updated, err := certs.UpdateRootCertsFile(h, h.PathTo("tls", TrustedRootsFilename), allRoots...)
		if err != nil {
			log.Error().Err(err).Msgf("failed to update trusted roots on host %s", hostname)
			continue
		}
		if updated {
			fmt.Printf("trusted roots updated on host %s\n", hostname)
		}
	}

	return
}
