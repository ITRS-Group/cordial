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
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
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

	file = path.Join(confDir, RootCABasename+".pem")

	roots, err := certs.ReadCertificates(LOCAL, file)
	if err != nil {
		return
	}
	if len(roots) == 0 {
		return nil, file, fmt.Errorf("no certificates found in %q", file)
	}
	root = roots[0]
	if !certs.IsValidRootCA(root) {
		err = fmt.Errorf("certificate in %q is not valid as a root CA", file)
	}
	return
}

// ReadSignerCertificate reads the signing certificate from the user's
// app config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory. The signer certificate is verified against the
// default root certificate.
func ReadSignerCertificate() (signer *x509.Certificate, file string, err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}

	file = path.Join(confDir, SigningCertBasename+".pem")

	signers, err := certs.ReadCertificates(LOCAL, file)
	if err != nil {
		return
	}
	if len(signers) == 0 {
		return nil, file, fmt.Errorf("no certificates found in %q", file)
	}
	signer = signers[0]

	if !certs.IsValidSigningCA(signer) {
		err = fmt.Errorf("certificate in %q not valid as a signer certificate", file)
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

// TLSImportBundle processes a PEM formatted signingBundle from a file
// or an embedded string with either an included private key and chain
// or separately specified in the same way.
func TLSImportBundle(signingBundleSource, privateKeySource string) (err error) {
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

	signingBundle, err := config.ReadPEMBytes(signingBundleSource, "signing certificate(s)")
	if err != nil {
		return err
	}

	privateKey, err := config.ReadPEMBytes(privateKeySource, "signing key")
	if err != nil {
		return err
	}

	certBundle, err := certs.ParsePEM(signingBundle, privateKey)
	if err != nil {
		return err
	}

	if certBundle.Leaf == nil {
		return errors.New("no certificates found in signing bundle")
	}

	if !certBundle.Valid {
		return errors.New("signing bundle is not valid")
	}

	cert := certBundle.Leaf

	key := certBundle.Key

	// basic validation
	if !cert.BasicConstraintsValid || !cert.IsCA || key == nil {
		return errors.New("no signing certificate with matching private key found in bundle")

	}

	if certBundle.Root == nil {
		return errors.New("no root certificate found in signing bundle")
	}

	if err = certs.WriteCertificates(LOCAL, path.Join(confDir, SigningCertBasename+".pem"), cert); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+".pem"))

	if err = certs.WritePrivateKey(LOCAL, path.Join(confDir, SigningCertBasename+".key"), key); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate key written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+".key"))

	if err = certs.WriteCertificates(LOCAL, path.Join(confDir, RootCABasename+".pem"), certBundle.Root); err != nil {
		return err
	}
	fmt.Printf("root CA certificate written to %s\n", path.Join(confDir, RootCABasename+".pem"))

	if updated, err := certs.UpdatedCACertsFile(LOCAL, TrustedRootsPath(LOCAL), certBundle.Root); err != nil {
		return err
	} else if updated {
		fmt.Printf("trusted roots updated with root certificate\n")
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
		if _, _, err := ReadSignerCertificate(); err == nil {
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

	if _, err := certs.WriteNewRootCert(
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
	_, err = certs.UpdatedCACertsFile(LOCAL, TrustedRootsPath(LOCAL), rootCert)
	if err != nil {
		return err
	}

	if _, err := certs.WriteNewSignerCert(
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
		updated, err := certs.UpdatedCACertsFile(h, h.PathTo("tls", TrustedRootsFilename), allRoots...)
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
