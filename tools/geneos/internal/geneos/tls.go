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
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// ReadRootCert reads the root certificate from the user's app config
// directory. It "promotes" old cert and key files from the previous tls
// directory if files do not already exist in the user app config
// directory. If verify is true then the certificate is verified against
// itself as a root and if it fails an error is returned.
func ReadRootCert(verify ...bool) (cert *x509.Certificate, file string, err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}
	file = config.PromoteFile(host.Localhost, confDir, LOCAL.PathTo("tls"), RootCABasename+".pem")
	if file == "" {
		err = ErrNotExist
		return
	}
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: root certificate file %s not found in %s", os.ErrNotExist, RootCABasename+".pem", confDir)
		return
	}
	config.PromoteFile(host.Localhost, confDir, LOCAL.PathTo("tls"), RootCABasename+".key")
	cert, err = config.ParseCertificate(LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
		if !cert.BasicConstraintsValid || !cert.IsCA {
			err = errors.New("root certificate not valid as a signing certificate")
			return
		}
		roots := x509.NewCertPool()
		roots.AddCert(cert)
		_, err = cert.Verify(x509.VerifyOptions{
			Roots: roots,
		})
	}
	return
}

// ReadSigningCert reads the signing certificate from the user's app
// config directory. It "promotes" old cert and key files from the
// previous tls directory if files do not already exist in the user app
// config directory. If verify is true then the signing certificate is
// checked and verified against the default root certificate.
func ReadSigningCert(verify ...bool) (cert *x509.Certificate, file string, err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}
	file = config.PromoteFile(host.Localhost, confDir, LOCAL.PathTo("tls", SigningCertBasename+".pem"))
	log.Debug().Msgf("reading %s", file)
	if file == "" {
		err = fmt.Errorf("%w: signing certificate file %s not found in %s", os.ErrNotExist, SigningCertBasename+".pem", confDir)
		return
	}
	config.PromoteFile(host.Localhost, confDir, LOCAL.PathTo("tls", SigningCertBasename+".key"))
	cert, err = config.ParseCertificate(LOCAL, file)
	if err != nil {
		return
	}
	if len(verify) > 0 && verify[0] {
		if !cert.BasicConstraintsValid || !cert.IsCA {
			err = errors.New("certificate not valid as a signing certificate")
			return
		}
		var root *x509.Certificate
		root, _, err = ReadRootCert(verify...)
		if err != nil {
			return
		}
		roots := x509.NewCertPool()
		roots.AddCert(root)
		_, err = cert.Verify(x509.VerifyOptions{
			Roots: roots,
		})
	}
	return
}

// DecomposePEM parses PEM formatted data and extracts the leaf
// certificate, any CA certs as a chain and a private key as a DER
// encoded *memguard.Enclave. The key is matched to the leaf
// certificate.
func DecomposePEM(data ...string) (cert *x509.Certificate, der *memguard.Enclave, chain []*x509.Certificate, err error) {
	var certs []*x509.Certificate
	var leaf *x509.Certificate
	var derkeys []*memguard.Enclave

	if len(data) == 0 {
		err = fmt.Errorf("no PEM data process")
		return
	}

	for _, pemstring := range data {
		pembytes := []byte(pemstring)
		for {
			block, rest := pem.Decode(pembytes)
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
				if !c.BasicConstraintsValid {
					err = ErrInvalidArgs
					return
				}
				if c.IsCA {
					certs = append(certs, c)
				} else if leaf == nil {
					// save first leaf
					leaf = c
				}
			case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
				// save all private keys for later matching
				derkeys = append(derkeys, memguard.NewEnclave(block.Bytes))
			default:
				err = fmt.Errorf("unsupported PEM type found: %s", block.Type)
				return
			}
			pembytes = rest
		}
	}

	if leaf == nil && len(certs) == 0 {
		err = fmt.Errorf("no certificates found")
		return
	}

	// if we got this far then we can start setting returns
	cert = leaf
	chain = certs

	// if we have no leaf certificate then user the first cert from the
	// chain BUT leave do not remove from the chain. order is not checked
	if cert == nil {
		cert = chain[0]
	}

	// are we good? check key and return a chain of valid CA certs
	if i := config.MatchKey(cert, derkeys); i != -1 {
		der = derkeys[i]
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
	signingBundle, err := config.ReadInputPEMString(signingBundleSource, "signing certificate(s)")
	if err != nil {
		return err
	}

	privateKey, err := config.ReadInputPEMString(privateKeySource, "signing key")
	if err != nil {
		return err
	}

	cert, key, chain, err := DecomposePEM(signingBundle, privateKey)
	if err != nil {
		return err
	}
	// basic validation
	if !(cert.BasicConstraintsValid && cert.IsCA) {
		return ErrInvalidArgs
	}

	if key == nil {
		return errors.New("no matching private key found")
	}

	if err = config.WriteCert(LOCAL, path.Join(confDir, SigningCertBasename+".pem"), cert); err != nil {
		return err
	}
	fmt.Printf("%s signing certificate written to %s\n", cordial.ExecutableName(), path.Join(confDir, SigningCertBasename+".pem"))

	if err = config.WritePrivateKey(LOCAL, path.Join(confDir, SigningCertBasename+".key"), key); err != nil {
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
	if err = config.WriteCertChain(LOCAL, path.Join(tlsPath, ChainCertFile), chain...); err != nil {
		return err
	}
	return
}

// create the tls/ directory in Geneos and a CA / DCA as required
//
// later options to allow import of a DCA
//
// This is also called from `init`
func TLSInit(overwrite bool, keytype string) (err error) {
	confDir := config.AppConfigDir()
	if confDir == "" {
		err = config.ErrNoUserConfigDir
		return
	}
	// directory permissions do not need to be restrictive
	err = LOCAL.MkdirAll(confDir, 0775)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	if err := config.CreateRootCert(
		LOCAL,
		path.Join(confDir, RootCABasename),
		cordial.ExecutableName()+" root certificate",
		overwrite,
		keytype); err != nil {
		if errors.Is(err, os.ErrExist) {
			// fmt.Println("root certificate already exists in", confDir)
			return nil
		}
		return err
	}
	fmt.Printf("CA created for %s\n", RootCABasename)

	if err := config.CreateSigningCert(
		LOCAL, path.Join(confDir, SigningCertBasename),
		path.Join(confDir, RootCABasename),
		cordial.ExecutableName()+" intermediate certificate",
		overwrite); err != nil {
		if errors.Is(err, os.ErrExist) {
			// fmt.Println("signing certificate already exists in", confDir)
			return nil
		}
		return err
	}
	fmt.Printf("Signing certificate created for %s\n", SigningCertBasename)

	// sync if geneos root exists
	if d, err := os.Stat(LocalRoot()); err == nil && d.IsDir() {
		return TLSSync()
	}
	return nil
}

// TLSSync creates and copies a certificate chain file to all remote
// hosts
//
// If a signing cert and/or a root cert exist, refresh the chain file
// from it, otherwise copy the chain file (using the configured name) to
// all remotes.
func TLSSync() (err error) {
	rootCert, _, err := ReadRootCert(true)
	if err != nil {
		rootCert = nil
	}
	geneosCert, _, err := ReadSigningCert()
	if err != nil {
		return os.ErrNotExist
	}

	if rootCert == nil && geneosCert == nil {
		tlsPath := LOCAL.PathTo("tls")
		chainpath := path.Join(tlsPath, ChainCertFile)
		if s, err := LOCAL.Stat(chainpath); err != nil && (s.Mode().IsRegular() || (s.Mode()&fs.ModeSymlink != 0)) {
			for _, r := range RemoteHosts(false) {
				host.CopyFile(LOCAL, tlsPath, r, r.PathTo("tls"))
			}
		}
		return
	}

	for _, r := range AllHosts() {
		tlsPath := r.PathTo("tls")
		if err = r.MkdirAll(tlsPath, 0775); err != nil {
			return
		}
		chainpath := path.Join(tlsPath, ChainCertFile)
		if err = config.WriteCertChain(r, chainpath, geneosCert, rootCert); err != nil {
			return
		}

		fmt.Printf("Updated certificate chain %s pem on %s\n", chainpath, r.String())
	}
	return
}
