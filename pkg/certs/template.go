package certs

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"
)

// Template creates a basic x509 certificate template with the given
// parameters. The caller can modify the template as needed before using
// it to create a certificate. If duration is zero, a default of 365
// days is used.
func Template(cn string, sanDNSNames []string, duration time.Duration) (template *x509.Certificate) {
	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}
	if duration == 0 {
		duration = 365 * 24 * time.Hour
	}
	expires := time.Now().Add(duration)
	template = &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:      time.Now().Add(-60 * time.Second),
		NotAfter:       expires,
		KeyUsage:       x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLenZero: true,
		DNSNames:       sanDNSNames,
		// IPAddresses:    []net.IP{net.ParseIP("127.0.0.1")},
	}
	return
}
