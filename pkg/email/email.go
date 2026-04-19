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

// The email package provides Geneos specific email processing to
// support libemail and other email senders using Geneos formatted
// parameters.
package email

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
)

// Geneos email params is a map of key/value pairs that conform to the
// libemail Email formats enhanced with changes to support SMTP
// Authentication and TLS
//
// The original libemail documentation is:
// <https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_rulesactionsalerts_tr.html#Libemail>
//
// The additions/changes are:
//
//  _SMTP_USERNAME - Username for SMTP Authentication
//  _SMTP_PASSWORD - Password for SMTP Authentication. May be encrypted using [ExpandString] format.
//  _SMTP_TLS
//
// Other parameters

// Dial sets up a Dialer using the Geneos _SMTP_* parameters
//
// All the parameters in the official docs are supported and have the same
// defaults
//
// Additional parameters are (which are case insensitive):
//
// * _SMTP_TLS - default / force / none (case insensitive)
// * _SMTP_USERNAME - if defined authentication attempted
// * _SMTP_PASSWORD - password in [ExpandString] format
func Dial(conf *config.Config) (d *mail.Client, err error) {
	var tlsPolicy mail.TLSPolicy

	switch strings.ToLower(config.Get[string](conf, "_smtp_tls", config.DefaultValue("default"))) {
	case "force":
		tlsPolicy = mail.TLSMandatory
	case "none":
		tlsPolicy = mail.NoTLS
	default:
		tlsPolicy = mail.TLSOpportunistic
	}

	mailOpts := []mail.Option{
		mail.WithTLSPortPolicy(tlsPolicy),
		mail.WithTimeout(time.Duration(config.Get[int](conf, "_smtp_timeout", config.DefaultValue(10))) * time.Second),
	}

	if config.Get[string](conf, "_smtp_username") != "" {
		mailOpts = append(mailOpts,
			mail.WithUsername(config.Get[string](conf, "_smtp_username")),
			mail.WithPassword(config.Get[string](conf, "_smtp_password")),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
		)
	} else {
		mailOpts = append(mailOpts,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		)
	}

	// override port policy if we are told to, but zero skips through
	// sometimes, so check that too
	if p, ok := config.Lookup[int](conf, "_smtp_port"); ok && p != 0 {
		mailOpts = append(mailOpts, mail.WithPort(p))
	}

	if config.Get[bool](conf, "_smtp_tls_insecure") {
		mailOpts = append(mailOpts, mail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}

	d, err = mail.NewClient(config.Get[string](conf, "_smtp_server", config.DefaultValue("localhost")), mailOpts...)
	if err != nil {
		return
	}

	d.SetTLSPolicy(tlsPolicy)

	return d, nil
}

// UpdateEnvelope processes the Geneos libemail parameters for _FROM, _TO, _CC
// and _BCC and their related parameters stored in conf and returns a
// populated mail.Message structure or an error.
func UpdateEnvelope(conf *config.Config, inlineCSS bool) (m *mail.Msg, err error) {
	m = mail.NewMsg()

	var from = config.Get[string](conf, "_from", config.DefaultValue("geneos@localhost"))
	var fromName = config.Get[string](conf, "_from_name", config.DefaultValue("Geneos"))

	m.FromFormat(fromName, from)

	err = addAddresses(m, conf, "to")
	if err != nil {
		return
	}

	err = addAddresses(m, conf, "cc")
	if err != nil {
		return
	}

	err = addAddresses(m, conf, "bcc")
	if err != nil {
		return
	}

	return
}

// The Geneos libemail supports an optional text name per address and also the info type,
// if given, must match "email" or "e-mail" (case insensitive). If either names or info types
// are given they MUST have the same number of members otherwise it's a fatal error
func addAddresses(m *mail.Msg, conf *config.Config, header string) (err error) {
	addrs := splitCommaTrimSpace(config.Get[string](conf, "_"+header))
	names := splitCommaTrimSpace(config.Get[string](conf, "_"+header+"_name"))
	infotypes := splitCommaTrimSpace(config.Get[string](conf, "_"+header+"_info_type"))

	if len(names) > 0 && len(addrs) != len(names) {
		return fmt.Errorf("\"%s\" header items mismatch: addrs=%d != names=%d", header, len(addrs), len(names))
	}

	if len(infotypes) > 0 && len(addrs) != len(infotypes) {
		return fmt.Errorf("\"%s\" header items mismatch: addrs=%d != infotypes=%d", header, len(addrs), len(infotypes))
	}

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
		switch header {
		case "to":
			err = m.AddToFormat(name, to)
		case "cc":
			err = m.AddCcFormat(name, to)
		case "bcc":
			err = m.AddBccFormat(name, to)
		}
		if err != nil {
			return
		}
	}

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

// NewEmailConfig sets up a config.Config containing Geneos specific email settings, ready to use with
func NewEmailConfig(cf *config.Config, toArg, ccArg, bccArg, subjectArg string) (em *config.Config) {
	em = config.New()
	// set default from yaml file, can be overridden from Geneos as
	// environment variables

	// creds can come from `geneos` credentials for the mail server
	// domain

	epassword := &config.Secret{}
	var eusername, smtpserver string

	if cf != nil {
		eusername = config.Get[string](cf, "email.username")
		smtpserver = config.Get[string](cf, "email.smtp")

		if eusername != "" {
			epassword = config.Get[*config.Secret](cf, "email.password")
		}

		if eusername == "" {
			creds := config.FindCreds(smtpserver, config.SetAppName("geneos"))
			if creds != nil {
				eusername = config.Get[string](creds, "username")
				epassword = config.Get[*config.Secret](creds, "password")
			}
		}
	}

	em.Default("_smtp_username", eusername)
	em.Default("_smtp_password", epassword.String())
	em.Default("_smtp_server", smtpserver)
	if cf != nil {
		em.Default("_smtp_tls", config.Get[string](cf, "email.use-tls"))
		em.Default("_smtp_tls_insecure", config.Get[bool](cf, "email.tls-skip-verify"))
		em.Default("_smtp_port", config.Get[int](cf, "email.port"))
		em.Default("_from", config.Get[string](cf, "email.from"))
		em.Default("_to", config.Get[string](cf, "email.to"))
		em.Default("_cc", config.Get[string](cf, "email.cc"))
		em.Default("_bcc", config.Get[string](cf, "email.bcc"))
		em.Default("_subject", config.Get[string](cf, "email.subject"))
	}

	for _, e := range os.Environ() {
		n := strings.SplitN(e, "=", 2)
		config.Set(em, n[0], n[1])
	}

	// override with args
	if toArg != "" {
		config.Set(em, "_to", toArg)
	}
	if ccArg != "" {
		config.Set(em, "_cc", ccArg)
	}
	if bccArg != "" {
		config.Set(em, "_bcc", bccArg)
	}
	if subjectArg != "" {
		config.Set(em, "_subject", subjectArg)
	}

	return
}
