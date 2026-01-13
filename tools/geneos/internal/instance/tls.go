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

package instance

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance/responses"
)

// NewCertificate creates a new certificate for an instance.
//
// If the root and signing certs are readable then create an instance
// specific chain file, otherwise set the instance to point to the
// system chain file.
//
// this also creates a new private key
//
// skip if certificate exists and is valid
func NewCertificate(i geneos.Instance, days int) (resp *responses.Response) {
	cf := i.Config()

	resp = responses.NewResponse(i)

	// skip if we can load an existing and valid certificate
	cert, err := ReadCertificate(i)
	if err == nil {
		if certs.IsValidLeafCert(cert) {
			resp.Summary = "certificate already exists and is valid (use the `renew` command to overwrite)"
			return
		}
	}

	confDir := config.AppConfigDir()
	if confDir == "" {
		resp.Err = config.ErrNoUserConfigDir
		return
	}

	signingCert, _, err := geneos.ReadSignerCertificate()
	if err != nil {
		resp.Err = err
		return
	}
	signingKey, err := certs.ReadPrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertBasename+".key"))
	if err != nil {
		resp.Err = err
		return
	}

	template := certs.Template("geneos "+i.Type().String()+" "+i.Name(),
		certs.DNSNames(i.Host().GetString("hostname")),
		certs.Days(days),
	)
	expires := template.NotAfter

	cert, key, err := certs.CreateCertificateAndKey(template, signingCert, signingKey)
	if err != nil {
		resp.Err = err
		return
	}

	certPath := ComponentFilepath(i, "pem")
	log.Debug().Msgf("writing certificate to %q", certPath)
	if err = certs.WriteCertificates(i.Host(), certPath, cert, signingCert); err != nil {
		resp.Err = err
		return
	}
	// remove old certificate entry
	cf.Set("certificate", "")
	cf.SetString(cf.Join("tls", "certificate"), certPath, config.Replace("home"))

	keyPath := ComponentFilepath(i, "key")
	log.Debug().Msgf("writing private key to %q", keyPath)
	if err = certs.WritePrivateKey(i.Host(), keyPath, key); err != nil {
		resp.Err = err
		return
	}
	// remove old private key entry
	cf.Set("privatekey", "")

	cf.SetString(cf.Join("tls", "privatekey"), keyPath, config.Replace("home"))

	if err = SaveConfig(i); err != nil {
		resp.Err = err
		return
	}

	resp.Summary = fmt.Sprintf("certificate created, expires %s", expires.UTC().Format(time.RFC3339))

	resp.Details = []string{
		fmt.Sprintf("certificate created for %s", i),
		fmt.Sprintf("            Expiry: %s", expires.UTC().Format(time.RFC3339)),
		fmt.Sprintf("  SHA1 Fingerprint: %X", sha1.Sum(cert.Raw)),
		fmt.Sprintf("SHA256 Fingerprint: %X", sha256.Sum256(cert.Raw)),
	}
	return
}

// WriteCertificate writes the certificate in the instance i directory
// using standard file name of TYPE.pem and updates the `certificate`
// parameter. It does not write the instance configuration, expecting
// the caller to do so after any other updates.
//
// If any extensions are passed (as ext), they are appended to the
// filename with dot separators, e.g. for temporary files and the
// instance config is not updated.
func WriteCertificate(i geneos.Instance, cert *x509.Certificate, ext ...string) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certFile := ComponentFilepath(i, append([]string{"pem"}, ext...)...)
	if err = certs.WriteCertificates(i.Host(), certFile, cert); err != nil {
		return
	}

	if len(ext) > 0 || cf.GetString(cf.Join("tls", "certificate")) == certFile {
		// do not update config if ext is given (used for temp files) or
		// if it's already set
		return
	}
	cf.Set("certificate", "")
	cf.SetString(cf.Join("tls", "certificate"), certFile, config.Replace("home"))
	if err = SaveConfig(i); err != nil {
		return
	}
	return
}

// WriteCertificates writes the certificates to a single file in the
// instance i directory using standard file name of TYPE.pem and updates
// the `certificate` parameter. It does not write the instance
// configuration, expecting the caller to do so after any other updates.
//
// If any extensions are passed (as ext), they are appended to the
// filename with dot separators, e.g. for temporary files and the
// instance config is not updated.
func WriteCertificates(i geneos.Instance, certSlice []*x509.Certificate, ext ...string) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certFile := ComponentFilepath(i, append([]string{"pem"}, ext...)...)
	if err = certs.WriteCertificates(i.Host(), certFile, certSlice...); err != nil {
		return
	}

	if len(ext) > 0 || cf.GetString(cf.Join("tls", "certificate")) == certFile {
		// do not update config if ext is given (used for temp files) or
		// if it's already set
		return
	}
	cf.Set("certificate", "")
	cf.SetString(cf.Join("tls", "certificate"), certFile, config.Replace("home"))
	if err = SaveConfig(i); err != nil {
		return
	}
	return
}

// WritePrivateKey writes the private key in the instance i directory using
// standard file name of TYPE.key and updates the `privatekey` instance
// parameter. It does not write the instance configuration, expecting
// the caller to do so after any other updates.
//
// If any extensions are passed (as ext), they are appended to the
// filename with dot separators, e.g. for temporary files and the
// instance config is not updated.
func WritePrivateKey(i geneos.Instance, key *memguard.Enclave, ext ...string) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := ComponentFilepath(i, append([]string{"key"}, ext...)...)
	if err = certs.WritePrivateKey(i.Host(), keyfile, key); err != nil {
		return
	}
	if len(ext) > 0 || cf.GetString(cf.Join("tls", "privatekey")) == keyfile {
		// do not update config if ext is given (used for temp files) or
		// if it's already set
		return
	}
	cf.Set("privatekey", "")
	cf.SetString(cf.Join("tls", "privatekey"), keyfile, config.Replace("home"))
	if err = SaveConfig(i); err != nil {
		return
	}
	return
}

func ReadCertificate(i geneos.Instance, ext ...string) (cert *x509.Certificate, err error) {
	var certPath string

	cf := i.Config()
	if cf.IsSet(cf.Join("tls", "certificate")) {
		certPath = cf.GetString(cf.Join("tls", "certificate"))
	} else if cf.IsSet("certificate") {
		certPath = cf.GetString("certificate")
	} else {
		return nil, geneos.ErrNotExist
	}

	certChain, err := certs.ReadCertificates(i.Host(), strings.Join(append([]string{certPath}, ext...), "."))
	if err != nil {
		return nil, err
	}
	if len(certChain) == 0 {
		return nil, geneos.ErrNotExist
	}
	return certChain[0], nil
}

// ReadCertificates reads the instance certificate and returns a slice
// of certificates.
//
// For older Geneos versions that do not support full chains, this will
// check for an also load a chain file. The chain file path is taken
// from the `certchain` parameter if set and no optional extensions are
// added.
func ReadCertificates(i geneos.Instance, ext ...string) (certChain []*x509.Certificate, err error) {
	var certPath, chainPath string

	cf := i.Config()
	if cf.IsSet(cf.Join("tls", "certificate")) {
		certPath = cf.GetString(cf.Join("tls", "certificate"))
	} else if cf.IsSet("certificate") {
		certPath = cf.GetString("certificate")
		chainPath = cf.GetString("certchain")
	} else {
		return nil, geneos.ErrNotExist
	}

	certChain, err = certs.ReadCertificates(i.Host(), strings.Join(append([]string{certPath}, ext...), "."))
	if err != nil {
		return nil, err
	}
	if chainPath != "" {
		chainCerts, err := certs.ReadCertificates(i.Host(), chainPath)
		if err != nil {
			return nil, err
		}
		certChain = append(certChain, chainCerts...)
	}

	return
}

// ReadPrivateKey reads the instance RSA private key
func ReadPrivateKey(i geneos.Instance, ext ...string) (key *memguard.Enclave, err error) {
	var keyPath string

	if i.Type() == nil {
		return nil, geneos.ErrInvalidArgs
	}

	cf := i.Config()
	if cf.IsSet(cf.Join("tls", "privatekey")) {
		keyPath = cf.GetString(cf.Join("tls", "privatekey"))
	} else if cf.IsSet("privatekey") {
		keyPath = cf.GetString("privatekey")
	} else {
		return nil, geneos.ErrNotExist
	}

	return certs.ReadPrivateKey(i.Host(), strings.Join(append([]string{keyPath}, ext...), "."))
}
