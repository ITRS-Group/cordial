/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// The email package provides Geneos specific email processing to
// support libemail and other email senders using Geneos formatted
// parameters.
package email

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-mail/mail/v2"

	"github.com/itrs-group/cordial/pkg/config"
)

// Params is a map of key/value pairs that conform to the Geneos
// libemail Email formats enhanced with changes to support SMTP
// Authentication and TLS
//
// The original libemail documentation is <https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Libemail>
//
// The additions/changes are:
//
//	_SMTP_USERNAME - Username for SMTP Authentication
//	_SMTP_PASSWORD - Password for SMTP Authentication. May be encrypted using [ExpandString] format.
//	_SMTP_TLS
//
// Other parameters

// Set-up a Dialer using the _SMTP_* parameters
//
// All the parameters in the official docs are supported and have the same
// defaults
//
// Additional parameters are
// * _SMTP_TLS - default / force / none (case insensitive)
// * _SMTP_USERNAME - if defined authentication attempted
// * _SMTP_PASSWORD - password in [ExpandString] format
// * XXX - _SMTP_REFERENCE - override the SMTP Reference header for conversations / threading
func Dial(conf *config.Config) (d *mail.Dialer, err error) {
	server := conf.GetString("_SMTP_SERVER", config.Default("localhost"))
	port := conf.GetInt("_SMTP_PORT", config.Default(25))
	timeout := conf.GetInt("_SMTP_TIMEOUT", config.Default(10))

	var tlsPolicy mail.StartTLSPolicy

	tls := conf.GetString("_SMTP_TLS", config.Default("default"))
	switch strings.ToLower(tls) {
	case "force":
		tlsPolicy = mail.MandatoryStartTLS
	case "none":
		tlsPolicy = mail.NoStartTLS
	default:
		tlsPolicy = mail.OpportunisticStartTLS
	}

	if conf.IsSet("_SMTP_USERNAME") {
		username := conf.GetString("_SMTP_USERNAME")
		// get the password from the file given or continue with
		// an empty string
		password := conf.GetString("_SMTP_PASSWORD")
		// the password can be empty at this point. this is valid, even if a bit dumb.

		d = mail.NewDialer(server, port, username, password)
	} else {
		// no auth - initialise Dialer directly
		d = &mail.Dialer{Host: server, Port: port}
	}
	d.Timeout = time.Duration(timeout) * time.Second
	d.StartTLSPolicy = tlsPolicy

	return d, nil
}

func Envelope(conf *config.Config) (m *mail.Message, err error) {
	m = mail.NewMessage()

	var from = conf.GetString("_FROM", config.Default("geneos@localhost"))
	var fromName = conf.GetString("_FROM_NAME", config.Default("Geneos"))

	m.SetAddressHeader("From", from, fromName)

	err = addAddresses(m, conf, "To")
	if err != nil {
		return
	}

	err = addAddresses(m, conf, "Cc")
	if err != nil {
		return
	}

	err = addAddresses(m, conf, "Bcc")
	if err != nil {
		return
	}

	return
}

// The Geneos libemail supports an optional text name per address and also the info type,
// if given, must match "email" or "e-mail" (case insensitive). If either names or info types
// are given they MUST have the same number of members otherwise it's a fatal error
func addAddresses(m *mail.Message, conf *config.Config, header string) error {
	upperHeader := strings.ToUpper(header)
	addrs := splitCommaTrimSpace(conf.GetString(fmt.Sprintf("_%s", upperHeader)))
	names := splitCommaTrimSpace(conf.GetString(fmt.Sprintf("_%s_NAME", upperHeader)))
	infotypes := splitCommaTrimSpace(conf.GetString(fmt.Sprintf("_%s_INFO_TYPE", upperHeader)))

	if len(names) > 0 && len(addrs) != len(names) {
		return fmt.Errorf("\"%s\" header items mismatch: addrs=%d != names=%d", header, len(addrs), len(names))
	}

	if len(infotypes) > 0 && len(addrs) != len(infotypes) {
		return fmt.Errorf("\"%s\" header items mismatch: addrs=%d != infotypes=%d", header, len(addrs), len(infotypes))
	}

	var addresses []string

	for i, to := range addrs {
		var name string
		if len(infotypes) > 0 {
			if !strings.EqualFold("email", infotypes[i]) && !strings.EqualFold("e-mail", infotypes[i]) {
				continue
			}
		}

		if len(names) > 0 {
			name = names[i]
		}
		addresses = append(addresses, m.FormatAddress(to, name))
	}
	m.SetHeader(header, addresses...)

	return nil
}

// split a string on commas and trim leading and trailing spaces
// an empty string results in an empty slice and NOT a slice
// with one empty value
func splitCommaTrimSpace(s string) []string {
	if s == "" {
		return []string{}
	}
	fields := strings.Split(s, ",")
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}
	return fields
}
