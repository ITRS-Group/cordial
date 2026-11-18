package certs

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

// Template creates a basic x509 certificate template with the given
// parameters. The caller can modify the template as needed before using
// it to create a certificate. If duration is zero, a default of 365
// days is used.
//
// If the options are invalid, for example an IP address or URI that
// cannot be parsed, or a serial number cannot be generated, then Template
// returns nil and [CreateCertificate] will reject it. Use [NewTemplate]
// to get the error.
func Template(cn string, options ...TemplateOption) (template *x509.Certificate) {
	template, _ = NewTemplate(cn, options...)
	return
}

// NewTemplate is like [Template] but returns an error if the options
// are invalid, including any IP addresses or URIs that cannot be
// parsed, or if a serial number cannot be generated.
//
// A template with the [IsCA] option always has valid basic constraints,
// and unless [KeyUsage] is also given is limited to signing certificates
// and CRLs (in addition to DigitalSignature).
func NewTemplate(cn string, options ...TemplateOption) (template *x509.Certificate, err error) {
	opts := evalTemplateOptions(options...)
	if opts.err != nil {
		return nil, opts.err
	}

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return nil, fmt.Errorf("generating certificate serial number: %w", err)
	}

	if opts.duration == 0 {
		opts.duration = 365 * 24 * time.Hour
	}
	if opts.keyUsage == -1 {
		if opts.isCA {
			opts.keyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign
		} else {
			opts.keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		}
	}
	if len(opts.extKeyUsage) == 1 && opts.extKeyUsage[0] == -1 {
		opts.extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}

	now := time.Now()
	template = &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: now.Add(-60 * time.Second),
		NotAfter:  now.Add(opts.duration),
		IsCA:      opts.isCA,
		// basic constraints are only written to a certificate if
		// valid, so a CA must always have them
		BasicConstraintsValid: opts.basicConstraintsValid || opts.isCA,
		KeyUsage:              opts.keyUsage,
		ExtKeyUsage:           opts.extKeyUsage,
		MaxPathLen:            opts.maxPathLen,
		MaxPathLenZero:        opts.maxPathLenZero,
		DNSNames:              opts.dnsNames,
		IPAddresses:           opts.ipAddresses,
		EmailAddresses:        opts.emailAddresses,
		URIs:                  opts.uris,
	}
	return
}

type templateOptions struct {
	duration              time.Duration
	keyUsage              x509.KeyUsage
	extKeyUsage           []x509.ExtKeyUsage
	isCA                  bool
	basicConstraintsValid bool
	maxPathLen            int
	maxPathLenZero        bool
	dnsNames              []string
	ipAddresses           []net.IP
	emailAddresses        []string
	uris                  []*url.URL
	err                   error
}

func evalTemplateOptions(options ...TemplateOption) (opts *templateOptions) {
	opts = &templateOptions{
		keyUsage:    -1,
		extKeyUsage: []x509.ExtKeyUsage{-1},
	}
	for _, option := range options {
		option(opts)
	}
	return
}

// TemplateOption TODO: Describe its purpose and intended use.
type TemplateOption func(*templateOptions)

// Days TODO: Document its behavior, inputs, and effects.
func Days(days int) TemplateOption {
	return func(opts *templateOptions) {
		if days == 0 {
			days = 365
		}
		opts.duration = time.Duration(days) * 24 * time.Hour
	}
}

// KeyUsage TODO: Document its behavior, inputs, and effects.
func KeyUsage(ku x509.KeyUsage) TemplateOption {
	return func(opts *templateOptions) {
		opts.keyUsage = ku
	}
}

// ExtKeyUsage TODO: Document its behavior, inputs, and effects.
func ExtKeyUsage(eku ...x509.ExtKeyUsage) TemplateOption {
	return func(opts *templateOptions) {
		opts.extKeyUsage = eku
	}
}

// IsCA marks the certificate as a CA. Valid basic constraints are always
// set for a CA.
func IsCA() TemplateOption {
	return func(opts *templateOptions) {
		opts.isCA = true
	}
}

// BasicConstraintsValid TODO: Document its behavior, inputs, and effects.
func BasicConstraintsValid() TemplateOption {
	return func(opts *templateOptions) {
		opts.basicConstraintsValid = true
	}
}

// MaxPathLen TODO: Document its behavior, inputs, and effects.
func MaxPathLen(len int) TemplateOption {
	return func(opts *templateOptions) {
		opts.maxPathLen = len
		if len == 0 {
			opts.maxPathLenZero = true
		} else {
			opts.maxPathLenZero = false
		}
	}
}

// SANsFromCert sets the Subject Alternative Names from an existing certificate
func SANsFromCert(cert *x509.Certificate) TemplateOption {
	return func(opts *templateOptions) {
		if cert == nil {
			return
		}
		if len(cert.DNSNames) > 0 {
			opts.dnsNames = cert.DNSNames
		}
		if len(cert.IPAddresses) > 0 {
			opts.ipAddresses = cert.IPAddresses
		}
		if len(cert.EmailAddresses) > 0 {
			opts.emailAddresses = cert.EmailAddresses
		}
		if len(cert.URIs) > 0 {
			opts.uris = cert.URIs
		}
	}
}

// DNSNames TODO: Document its behavior, inputs, and effects.
func DNSNames(names ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.dnsNames = names
	}
}

// IPAddresses TODO: Document its behavior, inputs, and effects.
func IPAddresses(ips ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.ipAddresses = make([]net.IP, 0, len(ips))
		for _, ip := range ips {
			parsed := net.ParseIP(ip)
			if parsed == nil {
				opts.err = errors.Join(opts.err, fmt.Errorf("invalid IP address %q", ip))
				continue
			}
			opts.ipAddresses = append(opts.ipAddresses, parsed)
		}
	}
}

// EmailAddresses TODO: Document its behavior, inputs, and effects.
func EmailAddresses(emails ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.emailAddresses = emails
	}
}

// URIs TODO: Document its behavior, inputs, and effects.
func URIs(uris ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.uris = make([]*url.URL, 0, len(uris))
		for _, u := range uris {
			parsed, err := url.Parse(u)
			if err != nil {
				opts.err = errors.Join(opts.err, fmt.Errorf("invalid URI %q: %w", u, err))
				continue
			}
			opts.uris = append(opts.uris, parsed)
		}
	}
}
