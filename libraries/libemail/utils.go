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

package main

import "C"
import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/go-mail/mail/v2"
	"github.com/itrs-group/cordial/pkg/config"
)

func debug(conf EMailConfig) (debug bool) {
	if d, ok := conf["_DEBUG"]; ok {
		if strings.EqualFold(d, "true") {
			debug = true
		}
	}
	return
}

func setupMail(conf EMailConfig) (m *mail.Message, err error) {
	m = mail.NewMessage()

	var from = getWithDefault("_FROM", conf, "geneos@localhost")
	var fromName = getWithDefault("_FROM_NAME", conf, "Geneos")

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

	if !debug(conf) {
		keys := []string{"_FROM", "_FROM_NAME",
			"_TO", "_TO_NAME", "_TO_INFO_TYPE",
			"_CC", "_CC_NAME", "_CC_INFO_TYPE",
			"_BCC", "_BCC_NAME", "_BCC_INFO_TYPE",
		}
		for _, key := range keys {
			delete(conf, key)
		}
	}

	return
}

// Set-up a Dialer using the _SMTP_* parameters
//
// All the parameters in the official docs are supported and have the same
// defaults
//
// Additional parameters are
// * _SMTP_TLS - default / force / none (case insensitive)
// * _SMTP_USERNAME - if defined authentication attempted
// * _SMTP_PASSWORD_FILE - plain text password in external file
// * XXX - _SMTP_REFERENCE - override the SMTP Reference header for conversations / threading
func dialServer(conf EMailConfig) (d *mail.Dialer, err error) {
	server := getWithDefault("_SMTP_SERVER", conf, "localhost")
	port := getWithDefaultInt("_SMTP_PORT", conf, 25)
	timeout := getWithDefaultInt("_SMTP_TIMEOUT", conf, 10)

	var tlsPolicy mail.StartTLSPolicy

	tlsEnabled := getWithDefault("_SMTP_TLS", conf, "default")
	tlsInsecure := getWithDefault("_SMTP_TLS_INSECURE", conf, "false")
	switch strings.ToLower(tlsEnabled) {
	case "force":
		tlsPolicy = mail.MandatoryStartTLS
	case "none":
		tlsPolicy = mail.NoStartTLS
	default:
		tlsPolicy = mail.OpportunisticStartTLS
	}

	username, ok := conf["_SMTP_USERNAME"]
	if ok {
		// get the password from the file given or continue with
		// an empty string
		password := config.ExpandString(getWithDefault("_SMTP_PASSWORD", conf, ""))
		if password == "" {
			pwfile := getWithDefault("_SMTP_PASSWORD_FILE", conf, "")
			if pwfile != "" {
				password, err = readFileString(pwfile)
				if err != nil {
					return nil, err
				}
			}
		}
		// the password can be empty at this point. this is valid, even if a bit dumb.

		d = mail.NewDialer(server, port, username, password)
		t, err := strconv.ParseBool(tlsInsecure)
		if err == nil && t {
			d.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	} else {
		// no auth - initialise Dialer directly
		d = &mail.Dialer{Host: server, Port: port}
	}
	d.Timeout = time.Duration(timeout) * time.Second
	d.StartTLSPolicy = tlsPolicy

	if !debug(conf) {
		keys := []string{"_SMTP_SERVER", "_SMTP_PORT", "_SMTP_TIMEOUT",
			"_SMTP_USERNAME", "_SMTP_PASSWORD_FILE",
			"_SMTP_TLS",
		}
		for _, key := range keys {
			delete(conf, key)
		}
	}

	return d, nil
}

// parse the C args - "n" of them - and return a map
// a value of empty string where there is no "=" or value
func parseArgs(n C.int, args **C.char) EMailConfig {
	conf := make(EMailConfig)

	// unsafe.Slice() requires Go 1.17+
	for _, s := range unsafe.Slice(args, n) {
		t := strings.SplitN(C.GoString(s), "=", 2)
		if len(t) > 1 {
			conf[t[0]] = t[1]
		} else {
			conf[t[0]] = ""
		}
	}
	return conf
}

// return the value of the key from conf or a default
func getWithDefault(key string, conf EMailConfig, def string) string {
	if val, ok := conf[key]; ok {
		return val
	}
	return def
}

// return the value of the key from conf or a default
func getWithDefaultInt(key string, conf EMailConfig, def int) int {
	if val, ok := conf[key]; ok {
		num, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return num
	}
	return def
}

// The Geneos libemail supports an optional text name per address and also the info type,
// if given, must match "email" or "e-mail" (case insensitive). If either names or info types
// are given they MUST have the same number of members otherwise it's a fatal error
func addAddresses(m *mail.Message, conf EMailConfig, header string) error {
	upperHeader := strings.ToUpper(header)
	addrs := splitCommaTrimSpace(conf[fmt.Sprintf("_%s", upperHeader)])
	names := splitCommaTrimSpace(conf[fmt.Sprintf("_%s_NAME", upperHeader)])
	infotypes := splitCommaTrimSpace(conf[fmt.Sprintf("_%s_INFO_TYPE", upperHeader)])

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

func readFileString(path string) (contents string, err error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return
	}
	contents = string(file)
	return
}
