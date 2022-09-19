package instance

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/host"
)

// create a new certificate for an instance
//
// this also creates a new private key
//
// skip if certificate exists (no expiry check)
func CreateCert(c geneos.Instance) (err error) {
	tlsDir := filepath.Join(host.Geneos(), "tls")

	// skip if we can load an existing certificate
	if _, err = ReadCert(c); err == nil {
		return
	}

	hostname, _ := os.Hostname()
	if c.Host() != host.LOCAL {
		hostname = c.Host().GetString("hostname")
	}

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	expires := time.Now().AddDate(1, 0, 0)
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("geneos %s %s", c.Type(), c.Name()),
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       []string{hostname},
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}

	intrCert, err := ReadSigningCert()
	if err != nil {
		return
	}
	intrKey, err := host.LOCAL.ReadKey(filepath.Join(tlsDir, geneos.SigningCertFile+".key"))
	if err != nil {
		return
	}

	cert, key, err := CreateCertKey(&template, intrCert, intrKey, nil)
	if err != nil {
		return
	}

	if err = WriteCert(c, cert); err != nil {
		return
	}

	if err = WriteKey(c, key); err != nil {
		return
	}

	fmt.Printf("certificate created for %s (expires %s)", c, expires)

	return
}

func WriteCert(c geneos.Instance, cert *x509.Certificate) (err error) {
	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}
	certfile := c.Type().String() + ".pem"
	if err = c.Host().WriteCert(filepath.Join(c.Home(), certfile), cert); err != nil {
		return
	}
	if c.Config().GetString("certificate") == certfile {
		return
	}
	c.Config().Set("certificate", certfile)

	return WriteConfig(c)
}

func WriteKey(c geneos.Instance, key *rsa.PrivateKey) (err error) {
	if c.Type() == nil {
		return geneos.ErrInvalidArgs
	}

	keyfile := c.Type().String() + ".key"
	if err = c.Host().WriteKey(filepath.Join(c.Home(), keyfile), key); err != nil {
		return
	}
	if c.Config().GetString("privatekey") == keyfile {
		return
	}
	c.Config().Set("privatekey", keyfile)
	return WriteConfig(c)
}

// read the rootCA certificate from the installation directory
func ReadRootCert() (cert *x509.Certificate, err error) {
	tlsDir := filepath.Join(host.Geneos(), "tls")
	return host.LOCAL.ReadCert(filepath.Join(tlsDir, geneos.RootCAFile+".pem"))
}

// read the signing certificate from the installation directory
func ReadSigningCert() (cert *x509.Certificate, err error) {
	tlsDir := filepath.Join(host.Geneos(), "tls")
	return host.LOCAL.ReadCert(filepath.Join(tlsDir, geneos.SigningCertFile+".pem"))
}

// read the instance certificate
func ReadCert(c geneos.Instance) (cert *x509.Certificate, err error) {
	if c.Type() == nil {
		return nil, geneos.ErrInvalidArgs
	}

	if Filename(c, "certificate") == "" {
		return nil, os.ErrNotExist
	}
	return c.Host().ReadCert(Filepath(c, "certificate"))
}

// read the instance RSA private key
func ReadKey(c geneos.Instance) (key *rsa.PrivateKey, err error) {
	if c.Type() == nil {
		return nil, geneos.ErrInvalidArgs
	}

	return c.Host().ReadKey(Abs(c, c.Config().GetString("privatekey")))
}

// wrapper to create a new certificate given the sign cert and private key and an optional private key to (re)use
// for the created certificate itself. returns a certificate and private key
func CreateCertKey(template, parent *x509.Certificate, parentKey *rsa.PrivateKey, existingKey *rsa.PrivateKey) (cert *x509.Certificate, key *rsa.PrivateKey, err error) {
	if existingKey != nil {
		key = existingKey
	} else {
		key, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return
		}
	}

	privKey := key
	if parentKey != nil {
		privKey = parentKey
	}

	var certBytes []byte
	if certBytes, err = x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, privKey); err == nil {
		cert, err = x509.ParseCertificate(certBytes)
	}

	return
}
