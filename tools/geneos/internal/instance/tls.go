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
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// CreateCert creates a new certificate for an instance.
//
// If the root and signing certs are readable then create an instance
// specific chain file, otherwise set the instance to point to the
// system chain file.
//
// this also creates a new private key
//
// skip if certificate exists and is valid
func CreateCert(i geneos.Instance, duration time.Duration) (resp *Response) {
	resp = NewResponse(i)

	// skip if we can load an existing and valid certificate
	if _, valid, _, err := ReadCert(i); err == nil && valid {
		resp.Line = "certificate already exists and is valid (use the `renew` command to overwrite)"
		return
	}

	confDir := config.AppConfigDir()
	if confDir == "" {
		resp.Err = config.ErrNoUserConfigDir
		return
	}

	signingCert, _, err := geneos.ReadSigningCert()
	if err != nil {
		resp.Err = err
		return
	}
	signingKey, err := config.ReadPrivateKey(geneos.LOCAL, path.Join(config.AppConfigDir(), geneos.SigningCertBasename+".key"))
	if err != nil {
		resp.Err = err
		return
	}

	hostname := i.Host().GetString("hostname")

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		resp.Err = err
		return
	}
	if duration == 0 {
		// default to one year
		duration = 365 * 24 * time.Hour
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

	cert, key, err := config.CreateCertificateAndKey(&template, signingCert, signingKey, nil)
	if err != nil {
		resp.Err = err
		return
	}

	if err = WriteCert(i, cert); err != nil {
		resp.Err = err
		return
	}

	if err = WriteKey(i, key); err != nil {
		resp.Err = err
		return
	}

	// optional root for instance specific chain
	rootCert, _, _ := geneos.ReadRootCert()
	if rootCert == nil {
		i.Config().SetString("certchain", i.Host().PathTo("tls", geneos.ChainCertFile))
	} else {
		chainfile := PathOf(i, "certchain")
		if chainfile == "" {
			chainfile = path.Join(i.Home(), "chain.pem")
			i.Config().SetString("certchain", chainfile, config.Replace("home"))
		}

		if err = config.WriteCertChain(i.Host(), chainfile, signingCert, rootCert); err != nil {
			resp.Err = err
			return
		}
	}

	if err = SaveConfig(i); err != nil {
		resp.Err = err
		return
	}

	// resp.Completed = append(resp.Completed, fmt.Sprintf("certificate created, expires %s", expires.UTC().Format(time.RFC3339)))
	resp.Line = fmt.Sprintf("certificate created for %s (expires %s)", i, expires.UTC().Format(time.RFC3339))
	return
}

// WriteCert writes the certificate in the instance i directory using
// standard file name of TYPE.pem and updates the `certificate`
// parameter. It does not write the instance configuration, expecting
// the caller to do so after any other updates.
func WriteCert(i geneos.Instance, cert *x509.Certificate) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certfile := ComponentFilepath(i, "pem")
	if err = config.WriteCert(i.Host(), certfile, cert); err != nil {
		return
	}
	if cf.GetString("certificate") == certfile {
		return
	}
	cf.SetString("certificate", certfile, config.Replace("home"))
	return
}

// WriteKey writes the private key in the instance i directory using
// standard file name of TYPE.key and updates the `privatekey` instance
// parameter. It does not write the instance configuration, expecting
// the caller to do so after any other updates.
func WriteKey(i geneos.Instance, key *memguard.Enclave) (err error) {
	cf := i.Config()

	if i.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := ComponentFilepath(i, "key")
	if err = config.WritePrivateKey(i.Host(), keyfile, key); err != nil {
		return
	}
	if cf.GetString("privatekey") == keyfile {
		return
	}
	cf.SetString("privatekey", keyfile, config.Replace("home"))
	return
}

// ReadCert reads the instance certificate for c. It verifies the
// certificate against any chain file and, if that fails, against system
// certificates.
func ReadCert(i geneos.Instance) (cert *x509.Certificate, valid bool, chainfile string, err error) {
	if i.Type() == nil {
		return nil, false, "", geneos.ErrInvalidArgs
	}

	if FileOf(i, "certificate") == "" {
		return nil, false, "", os.ErrNotExist
	}
	cert, err = config.ParseCertificate(i.Host(), PathOf(i, "certificate"))
	if err != nil {
		return
	}

	// first check if we have a valid private key
	c, err := config.ReadCertificateFile(i.Host(), PathOf(i, "certificate"))
	if err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	pk, err := config.ReadPrivateKey(i.Host(), PathOf(i, "privatekey"))
	if err != nil {
		log.Debug().Err(err).Msg("")
		return
	}
	k, err := pk.Open()
	if err != nil {
		log.Debug().Err(err).Msg("")
		return
	}
	pembytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: k.Bytes(),
	})
	defer k.Destroy()

	_, err = tls.X509KeyPair(c, pembytes)
	if err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	// validate against certificate chain file on the same host, expiry
	// etc.
	chainfile = PathOf(i, "certchain")
	if chainfile == "" {
		chainfile = config.MigrateFile(i.Host(), i.Host().PathTo("tls", geneos.ChainCertFile), i.Host().PathTo("tls", "chain.pem"))
	}

	if cp := config.ReadCertChain(i.Host(), chainfile); cp != nil {
		opts := x509.VerifyOptions{
			Roots:         cp,
			Intermediates: cp,
		}

		if _, err = cert.Verify(opts); err == nil { // return if no error
			valid = true
			log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
			return
		}
		log.Debug().Err(err).Msg("")
	}

	// if failed against internal certs, try system ones
	if _, err = cert.Verify(x509.VerifyOptions{}); err == nil { // return if no error
		valid = true
		log.Debug().Msgf("cert %q verified", cert.Subject.CommonName)
		return
	}

	log.Debug().Msgf("cert %q NOT verified: %s", cert.Subject.CommonName, err)
	return
}

// ReadKey reads the instance RSA private key
func ReadKey(i geneos.Instance) (key *memguard.Enclave, err error) {
	if i.Type() == nil || i.Config().GetString("privatekey") == "" {
		return nil, geneos.ErrInvalidArgs
	}

	return config.ReadPrivateKey(i.Host(), Abs(i, i.Config().GetString("privatekey")))
}
