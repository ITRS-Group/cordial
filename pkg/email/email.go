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

	switch strings.ToLower(conf.GetString("_smtp_tls", config.Default("default"))) {
	case "force":
		tlsPolicy = mail.TLSMandatory
	case "none":
		tlsPolicy = mail.NoTLS
	default:
		tlsPolicy = mail.TLSOpportunistic
	}

	mailOpts := []mail.Option{
		mail.WithTLSPortPolicy(tlsPolicy),
		mail.WithTimeout(time.Duration(conf.GetInt("_smtp_timeout", config.Default(10))) * time.Second),
	}

	if conf.GetString("_smtp_username") != "" {
		mailOpts = append(mailOpts,
			mail.WithUsername(conf.GetString("_smtp_username")),
			mail.WithPassword(conf.GetPassword("_smtp_password").String()),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
		)
	} else {
		mailOpts = append(mailOpts,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		)
	}

	// override port policy if we are told to, but zero skips through
	// sometimes, so check that too
	if conf.IsSet("_smtp_port") && conf.GetInt("_smtp_port") != 0 {
		mailOpts = append(mailOpts, mail.WithPort(conf.GetInt("_smtp_port")))
	}

	if conf.GetBool("_smtp_tls_insecure") {
		mailOpts = append(mailOpts, mail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}

	d, err = mail.NewClient(conf.GetString("_smtp_server", config.Default("localhost")), mailOpts...)
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

	var from = conf.GetString("_from", config.Default("geneos@localhost"))
	var fromName = conf.GetString("_from_name", config.Default("Geneos"))

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
	addrs := splitCommaTrimSpace(conf.GetString("_" + header))
	names := splitCommaTrimSpace(conf.GetString("_" + header + "_name"))
	infotypes := splitCommaTrimSpace(conf.GetString("_" + header + "_info_type"))

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

	epassword := &config.Plaintext{}
	var eusername, smtpserver string

	if cf != nil {
		eusername = cf.GetString("email.username")
		smtpserver = cf.GetString("email.smtp")

		if eusername != "" {
			epassword = cf.GetPassword("email.password")
		}

		if eusername == "" {
			creds := config.FindCreds(smtpserver, config.SetAppName("geneos"))
			if creds != nil {
				eusername = creds.GetString("username")
				epassword = creds.GetPassword("password")
			}
		}
	}

	em.SetDefault("_smtp_username", eusername)
	em.SetDefault("_smtp_password", epassword.String())
	em.SetDefault("_smtp_server", smtpserver)
	if cf != nil {
		em.SetDefault("_smtp_tls", cf.GetString("email.use-tls"))
		em.SetDefault("_smtp_tls_insecure", cf.GetBool("email.tls-skip-verify"))
		em.SetDefault("_smtp_port", cf.GetInt("email.port"))
		em.SetDefault("_from", cf.GetString("email.from"))
		em.SetDefault("_to", cf.GetString("email.to"))
		em.SetDefault("_cc", cf.GetString("email.cc"))
		em.SetDefault("_bcc", cf.GetString("email.bcc"))
		em.SetDefault("_subject", cf.GetString("email.subject"))
	}

	for _, e := range os.Environ() {
		n := strings.SplitN(e, "=", 2)
		em.Set(n[0], n[1])
	}

	// override with args
	if toArg != "" {
		em.Set("_to", toArg)
	}
	if ccArg != "" {
		em.Set("_cc", ccArg)
	}
	if bccArg != "" {
		em.Set("_bcc", bccArg)
	}
	if subjectArg != "" {
		em.Set("_subject", subjectArg)
	}

	return
}
