package certs

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"net/url"
	"time"
)

// Template creates a basic x509 certificate template with the given
// parameters. The caller can modify the template as needed before using
// it to create a certificate. If duration is zero, a default of 365
// days is used.
func Template(cn string, options ...TemplateOption) (template *x509.Certificate) {
	opts := evalTemplateOptions(options...)

	serial, err := rand.Prime(rand.Reader, 64)
	if err != nil {
		return
	}

	if opts.duration == 0 {
		opts.duration = 365 * 24 * time.Hour
	}
	if opts.keyUsage == -1 {
		opts.keyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	}
	if len(opts.extKeyUsage) == 1 && opts.extKeyUsage[0] == -1 {
		opts.extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	}

	expires := time.Now().Add(opts.duration)
	template = &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now().Add(-60 * time.Second),
		NotAfter:              expires,
		IsCA:                  opts.isCA,
		BasicConstraintsValid: opts.basicConstraintsValid,
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

type TemplateOption func(*templateOptions)

func Days(days int) TemplateOption {
	return func(opts *templateOptions) {
		if days == 0 {
			days = 365
		}
		opts.duration = time.Duration(days) * 24 * time.Hour
	}
}

func KeyUsage(ku x509.KeyUsage) TemplateOption {
	return func(opts *templateOptions) {
		opts.keyUsage = ku
	}
}

func ExtKeyUsage(eku ...x509.ExtKeyUsage) TemplateOption {
	return func(opts *templateOptions) {
		opts.extKeyUsage = eku
	}
}

func IsCA() TemplateOption {
	return func(opts *templateOptions) {
		opts.isCA = true
	}
}

func BasicConstraintsValid() TemplateOption {
	return func(opts *templateOptions) {
		opts.basicConstraintsValid = true
	}
}

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

func DNSNames(names ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.dnsNames = names
	}
}

func IPAddresses(ips ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.ipAddresses = make([]net.IP, 0, len(ips))
		for _, ip := range ips {
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			opts.ipAddresses = append(opts.ipAddresses, parsed)
		}
	}
}

func EmailAddresses(emails ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.emailAddresses = emails
	}
}

func URIs(uris ...string) TemplateOption {
	return func(opts *templateOptions) {
		opts.uris = make([]*url.URL, 0, len(uris))
		for _, u := range uris {
			parsed, err := url.Parse(u)
			if err != nil {
				continue
			}
			opts.uris = append(opts.uris, parsed)
		}
	}
}
