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
	"strings"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
)

const (
	// RootCABasename is the file base name for the root certificate authority
	// created with the TLS commands
	RootCABasename = "rootCA"

	// CABundleFilename is the file name for the ca-bundle file used by
	// Geneos components to verify peer certificates. This file is
	// located in the geneos home directory on each hoist under `tls/`
	CABundleBasename string = "ca-bundle"

	// SigningCertLabel is the descriptive label for the signing
	// certificate created with the TLS commands. It is commonly
	// prefixed with the executable name and followed by a parenthesised
	// hostname to indicate where it is being used.
	SigningCertLabel = "signing certificate"
)

// SigningCertBasename is the file base name for the signing certificate
// created with the TLS commands. This is initialised to the executable
// name in the Init() function.
var SigningCertBasename string

// DeprecatedChainCertFile the is file name (including extension, as
// this does not need to be used for keys) for the consolidated chain
// file used to verify instance certificates. This is initialised to
// the executable name with "-chain.pem" suffix in the Init() function.
//
// This is deprecated in favour of using the ca-bundle file.
// Non-migrated instances may still require this file.
var DeprecatedChainCertFile string

// PathToCABundle returns the path to the ca-bundle file on the given
// host with extensions concatenated from ext. Without any ext parameter arguments
// the returned file will be "ca-bundle".
func PathToCABundle(h *Host, ext ...string) string {
	return h.PathTo("tls", strings.Join(append([]string{CABundleBasename}, ext...), ""))
}

// PathToCABundlePEM returns the path to the ca-bundle PEM file on the given
// host.
func PathToCABundlePEM(h *Host) string {
	return PathToCABundle(h, certs.PEMExtension)
}

// RootCertificatePath returns the path to the root CA certificate in
// the user's app config directory.
func RootCertificatePath() (string, error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return "", config.ErrNoUserConfigDir
	}
	return path.Join(confDir, RootCABasename+certs.PEMExtension), nil
}

// ReadRootCertificateAndKey reads the root certificate and private key
// from the user's app config directory. If the private key cannot be
// found then nil is returned but no error.
func ReadRootCertificateAndKey() (cert *x509.Certificate, key *memguard.Enclave, err error) {
	file, err := RootCertificatePath()
	if err != nil {
		return
	}

	roots, err := certs.ReadCertificates(LOCAL, file)
	if err != nil {
		return
	}

	if len(roots) != 1 {
		err = fmt.Errorf("only one certificate allowed in %q", file)
		return
	}

	cert = roots[0]
	if !certs.IsValidRootCA(cert) {
		err = fmt.Errorf("certificate in %q is not valid as a root CA", file)
	}

	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}

	if key, err = certs.ReadPrivateKey(LOCAL, path.Join(confDir, RootCABasename+certs.KEYExtension)); err != nil && errors.Is(err, os.ErrNotExist) {
		err = nil
	}

	return
}

// SigningCertificatePath returns the path to the signing certificate in
// the user's app config directory.
func SigningCertificatePath() (string, error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return "", config.ErrNoUserConfigDir
	}
	return path.Join(confDir, SigningCertBasename+certs.PEMExtension), nil
}

// SigningPrivateKeyPath returns the path to the signing certificate
// private key in the user's app config directory.
func SigningPrivateKeyPath() (string, error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return "", config.ErrNoUserConfigDir
	}
	return path.Join(confDir, SigningCertBasename+certs.KEYExtension), nil
}

// ReadSigningCertificateAndKey reads the signing certificate and private
// key from the user's app config directory.
func ReadSigningCertificateAndKey() (cert *x509.Certificate, key *memguard.Enclave, err error) {
	cert, err = readSigningCertificate()
	if err != nil {
		return
	}
	key, err = readSigningPrivateKey()
	return
}

// readSigningCertificate reads the signing certificate from the user's
// app config directory. The signing certificate is verified against the
// default root certificate.
func readSigningCertificate() (signing *x509.Certificate, err error) {
	file, err := SigningCertificatePath()
	if err != nil {
		return
	}

	signings, err := certs.ReadCertificates(LOCAL, file)
	if err != nil {
		return
	}
	if len(signings) == 0 {
		return nil, fmt.Errorf("no certificates found in %q", file)
	}
	signing = signings[0]

	if !certs.IsValidSigningCA(signing) {
		err = fmt.Errorf("certificate in %q not valid as a signing certificate", file)
		return
	}

	// verify against root CA
	var root *x509.Certificate
	root, _, err = ReadRootCertificateAndKey()
	if err != nil {
		return
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	_, err = signing.Verify(x509.VerifyOptions{
		Roots: roots,
	})
	return
}

// readSigningPrivateKey reads the signing certificate private key from the
// user's app config directory.
func readSigningPrivateKey() (key *memguard.Enclave, err error) {
	file, err := SigningPrivateKeyPath()
	if err != nil {
		return
	}

	key, err = certs.ReadPrivateKey(LOCAL, file)
	return
}

// TLSImportBundle processes a PEM formatted signingBundle from a file
// or an embedded string with either an included private key and chain
// or separately specified in the same way.
//
// If the bundle contains only root certificates then these are added to
// the ca-bundle only. In this case any privateKeySource parameter is
func TLSImportBundle(signingBundleSource, privateKeySource string) (err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		return config.ErrNoUserConfigDir
		// ignored.
	}

	signingBundle, err := config.ReadPEMBytes(signingBundleSource, "signing certificate(s)")
	if err != nil {
		return err
	}

	leaf, chain, roots, _, err := certs.DecodePEM(signingBundle)
	if leaf == nil && len(chain) == 0 && len(roots) > 0 {
		// import roots to ca-bundle only
		updated, err := certs.UpdateCACertsFiles(LOCAL, PathToCABundle(LOCAL), roots...)
		if err != nil {
			return err
		} else if updated {
			fmt.Printf("ca-bundle updated with root certificate(s)\n")
		} else {
			fmt.Printf("ca-bundle is already up to date\n")
		}
		return nil
	}

	privateKey, err := config.ReadPEMBytes(privateKeySource, "signing key")
	if err != nil {
		return err
	}

	certBundle, err := certs.ParsePEM(signingBundle, privateKey)
	if err != nil {
		return err
	}

	if !certBundle.Valid {
		return errors.New("signing bundle is not valid")
	}

	if certBundle.Leaf == nil {
		return errors.New("no certificates found in signing bundle")
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

	if err = certs.WriteCertificates(LOCAL, path.Join(confDir, SigningCertBasename+certs.PEMExtension), cert); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+certs.PEMExtension))

	if err = certs.WritePrivateKey(LOCAL, path.Join(confDir, SigningCertBasename+certs.KEYExtension), key); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate key written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+certs.KEYExtension))

	if err = certs.WriteCertificates(LOCAL, path.Join(confDir, RootCABasename+certs.PEMExtension), certBundle.Root); err != nil {
		return err
	}
	fmt.Printf("root CA certificate written to %s\n", path.Join(confDir, RootCABasename+certs.PEMExtension))

	if updated, err := certs.UpdateCACertsFiles(LOCAL, PathToCABundle(LOCAL), certBundle.Root); err != nil {
		return err
	} else if updated {
		fmt.Printf("ca-bundle updated with root certificate\n")
	}

	return
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
//
// This is also called from `init`
func TLSInit(hostname string, overwrite bool, keytype certs.KeyType) (err error) {
	if !overwrite {
		if _, _, err := ReadRootCertificateAndKey(); err == nil {
			// root cert already exists
			log.Debug().Msg("root certificate already exists, skipping TLS initialisation")
			return nil
		}
		if _, err := readSigningCertificate(); err == nil {
			// signing cert already exists
			log.Debug().Msg("signing certificate already exists, skipping TLS initialisation")
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

	root, err := certs.WriteNewRootCert(
		path.Join(confDir, RootCABasename),
		cordial.ExecutableName()+" root certificate",
		keytype)
	if err != nil {
		return err
	}
	fmt.Printf("CA certificate created for %s\n", RootCABasename)
	fmt.Print(string(certs.CertificateComments(root)))

	rootCert, rootKey, err := ReadRootCertificateAndKey()
	if err != nil {
		return err
	}
	if rootKey == nil {
		return fmt.Errorf("no root private key found")
	}

	_, err = certs.UpdateCACertsFiles(LOCAL, PathToCABundle(LOCAL), rootCert)
	if err != nil {
		return err
	}

	signing, err := certs.WriteNewSigningCert(path.Join(confDir, SigningCertBasename), rootCert, rootKey,
		cordial.ExecutableName()+" "+SigningCertLabel+" ("+hostname+")",
	)
	if err != nil {
		return err
	}
	fmt.Printf("signing certificate created for %s\n", SigningCertBasename)
	fmt.Print(string(certs.CertificateComments(signing)))

	// sync if geneos root exists
	if d, err := os.Stat(LocalRoot()); err == nil && d.IsDir() {
		return TLSSync()
	}
	return nil
}

// TLSSync merges and updates the `CABundleFilename` file on all remote hosts.
func TLSSync() (err error) {
	allRoots := []*x509.Certificate{}
	allHosts := append([]*Host{LOCAL}, RemoteHosts(false)...)
	for _, h := range allHosts {
		if certSlice, err := certs.ReadCertificates(h, PathToCABundle(h, certs.PEMExtension)); err == nil {
			allRoots = append(allRoots, certSlice...)
		}
	}

	log.Debug().Msgf("found %d root certificates to sync", len(allRoots))

	for _, h := range allHosts {
		hostname := h.Hostname()
		updated, err := certs.UpdateCACertsFiles(h, PathToCABundle(h), allRoots...)
		if err != nil {
			log.Error().Err(err).Msgf("failed to update ca-bundle on host %s", hostname)
			continue
		}
		if updated {
			fmt.Printf("ca-bundle updated on host %s\n", hostname)
		} else {
			fmt.Printf("ca-bundle on host %s is already up to date\n", hostname)
		}
	}

	return
}
