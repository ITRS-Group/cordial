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
	"slices"
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
// * _SMTP_TLS - default / force / none / ssl (case insensitive).
// "default" tries STARTTLS and continues without it if not offered,
// "force" requires STARTTLS, "none" never uses TLS and "ssl" uses
// implicit TLS from the start of the connection (SMTPS). Implicit TLS
// is also used when _SMTP_PORT is 465 and _SMTP_TLS is not "none"
// * _SMTP_USERNAME - if defined authentication attempted
// * _SMTP_PASSWORD - password in [ExpandString] format
// * _SMTP_AUTH - the authentication mechanism to use, such as "plain",
// "login", "cram-md5", "scram-sha-256" or "xoauth2", with "-noenc" added
// for the plain text mechanisms to allow them without TLS. The default
// is to select the strongest mechanism offered by the server
//
// If _SMTP_PORT is not set the port is selected by the TLS policy,
// as 587 with "default" and "force" (with "default" falling back to 25
// if 587 cannot be reached), 465 for "ssl" and 25 for "none". If it is
// set then only that port is used.
func Dial(conf *config.Config) (d *mail.Client, err error) {
	server, mailOpts, err := dialOptions(conf)
	if err != nil {
		return nil, err
	}

	return mail.NewClient(server, mailOpts...)
}

// authTypes are the _smtp_auth names accepted, in upper case
var authTypes = []mail.SMTPAuthType{
	mail.SMTPAuthPlain,
	mail.SMTPAuthPlainNoEnc,
	mail.SMTPAuthLogin,
	mail.SMTPAuthLoginNoEnc,
	mail.SMTPAuthCramMD5,
	mail.SMTPAuthXOAUTH2,
	mail.SMTPAuthSCRAMSHA1,
	mail.SMTPAuthSCRAMSHA1PLUS,
	mail.SMTPAuthSCRAMSHA256,
	mail.SMTPAuthSCRAMSHA256PLUS,
	mail.SMTPAuthNTLM,
	mail.SMTPAuthAutoDiscover,
}

// dialOptions returns the SMTP server name and the options for the
// mail client built from conf
func dialOptions(conf *config.Config) (server string, mailOpts []mail.Option, err error) {
	server = config.Get[string](conf, "_smtp_server", config.DefaultValue("localhost"))
	if server == "" {
		server = "localhost"
	}

	var tlsPolicy mail.TLSPolicy
	implicit := false

	switch strings.ToLower(config.Get[string](conf, "_smtp_tls", config.DefaultValue("default"))) {
	case "force":
		tlsPolicy = mail.TLSMandatory
	case "none":
		tlsPolicy = mail.NoTLS
	case "ssl", "implicit":
		implicit = true
	default:
		tlsPolicy = mail.TLSOpportunistic
	}

	// zero is not a valid port and is sometimes returned for unset
	port, _ := config.Lookup[int](conf, "_smtp_port")
	if port == 465 && tlsPolicy != mail.NoTLS {
		implicit = true
	}

	// The port must be set before the TLS policy, as go-mail only
	// selects a default port, and a fallback to plain text on port 25,
	// when no other port has already been set. If a port is given then
	// it is used alone: a failure to connect is an error and never
	// falls back to port 25 without TLS.
	switch {
	case port != 0 && implicit:
		mailOpts = append(mailOpts, mail.WithPort(port), mail.WithSSL())
	case port != 0:
		// the policy only, as the port policy would change port 25
		mailOpts = append(mailOpts, mail.WithPort(port), mail.WithTLSPolicy(tlsPolicy))
	case implicit:
		mailOpts = append(mailOpts, mail.WithSSLPort(false))
	default:
		mailOpts = append(mailOpts, mail.WithTLSPortPolicy(tlsPolicy))
	}

	mailOpts = append(mailOpts,
		mail.WithTimeout(time.Duration(config.Get[int](conf, "_smtp_timeout", config.DefaultValue(10)))*time.Second),
	)

	if config.Get[string](conf, "_smtp_username") != "" {
		auth := mail.SMTPAuthAutoDiscover
		if name := strings.ToUpper(strings.TrimSpace(config.Get[string](conf, "_smtp_auth"))); name != "" && name != "AUTO" {
			auth = mail.SMTPAuthType(name)
			if !slices.Contains(authTypes, auth) {
				return "", nil, fmt.Errorf("unknown _smtp_auth mechanism %q", name)
			}
		}
		mailOpts = append(mailOpts,
			mail.WithUsername(config.Get[string](conf, "_smtp_username")),
			mail.WithPassword(config.Get[string](conf, "_smtp_password")),
			mail.WithSMTPAuth(auth),
		)
	} else {
		mailOpts = append(mailOpts,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		)
	}

	if config.Get[bool](conf, "_smtp_tls_insecure") {
		mailOpts = append(mailOpts, mail.WithTLSConfig(&tls.Config{
			ServerName:         server,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}))
	}

	return server, mailOpts, nil
}

// UpdateEnvelope processes the Geneos libemail parameters for _FROM, _TO, _CC
// and _BCC and their related parameters stored in conf and returns a
// populated mail.Message structure or an error.
func UpdateEnvelope(conf *config.Config, inlineCSS bool) (m *mail.Msg, err error) {
	m = mail.NewMsg()

	from := config.Get[string](conf, "_from", config.DefaultValue("geneos@localhost"))
	if from == "" {
		from = "geneos@localhost"
	}
	fromName := config.Get[string](conf, "_from_name", config.DefaultValue("Geneos"))

	if err = m.FromFormat(fromName, from); err != nil {
		return nil, fmt.Errorf("invalid _from address %q: %w", from, err)
	}

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

	var epassword config.Secret
	var eusername, smtpserver string

	if cf != nil {
		eusername = config.Get[string](cf, cf.Join("email", "username"))
		smtpserver = config.Get[string](cf, cf.Join("email", "smtp"))

		if eusername != "" {
			epassword = config.Get[config.Secret](cf, cf.Join("email", "password"))
		}

		if eusername == "" {
			creds := config.FindCreds(smtpserver, config.AppName("geneos"))
			if creds != nil {
				eusername = config.Get[string](creds, "username")
				epassword = config.Get[config.Secret](creds, "password")
			}
		}
	}

	// do not set empty defaults for the settings that have their own
	// defaults, as an empty default would hide them
	setDefault := func(key, value string) {
		if value != "" {
			em.Default(key, value)
		}
	}

	em.Default("_smtp_username", eusername)
	em.Default("_smtp_password", string(epassword))
	setDefault("_smtp_server", smtpserver)
	if cf != nil {
		setDefault("_smtp_tls", config.Get[string](cf, cf.Join("email", "use-tls")))
		em.Default("_smtp_tls_insecure", config.Get[bool](cf, cf.Join("email", "tls-skip-verify")))
		em.Default("_smtp_port", config.Get[int](cf, cf.Join("email", "port")))
		setDefault("_from", config.Get[string](cf, cf.Join("email", "from")))
		em.Default("_to", config.Get[string](cf, cf.Join("email", "to")))
		em.Default("_cc", config.Get[string](cf, cf.Join("email", "cc")))
		em.Default("_bcc", config.Get[string](cf, cf.Join("email", "bcc")))
		em.Default("_subject", config.Get[string](cf, cf.Join("email", "subject")))
	}

	for _, e := range os.Environ() {
		if k, v, found := strings.Cut(e, "="); found {
			config.Set(em, k, v)
		}
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
