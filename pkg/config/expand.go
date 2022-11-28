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

/*
Package config adds support for value expansion over viper based
configurations.

A number of the most common access methods from viper are replaced with
local versions that add support for [ExpandString]. Additionally, there
are a number of functions to simplify programs including [LoadConfig].
*/
package config

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
)

// ExpandString returns the input with all occurrences of the form
// ${name} replaced using an [os.Expand]-like function (but without
// support for bare names) for the built-in and optional formats (in the
// order of priority) below. The caller can use options to define
// additional expansion functions based on a "prefix:", disabled
// external lookups and also to pass in lookup tables referred to as
// value maps.
//
//	${enc:keyfile[|keyfile...]:encodedvalue}
//
//	   The item "encodedvalue" is an AES256 ciphertext in Geneos format
//	   - or a reference to one - which will be decoded using the key
//	   file(s) given. Each "keyfile" must be one of either an absolute
//	   path, a path relative to the working directory of the program, or
//	   if prefixed with "~/" then relative to the home directory of the
//	   user running the program. The first valid decode (see below) is
//	   returned.
//
//	   The "encodedvalue" must be either prefixed "+encs+" to align with
//	   Geneos or will otherwise be looked up using the forms of any of
//	   the other references below, but without the surrounding
//	   dollar-brackets "${...}".
//
//	   To minimise (but not wholly eliminate) any false-positive decodes
//	   that occur in some circumstances when using the wrong key file,
//	   the decoded value is only returned if it is a valid UTF-8 string
//	   as per [utf8.Valid].
//
//	   Examples:
//
//	   * password: ${enc:~/.keyfile:+encs+9F2C3871E105EC21E4F0D5A7921A937D}
//	   * password: ${enc:/etc/geneos/keyfile.aes:env:ENCODED_PASSWORD}
//	   * password: ${enc:~/.config/geneos/keyfile1.aes:app.password}
//	   * password: ${enc:~/.keyfile.aes:config:mySecret}
//
//	${config:key} or ${path.to.config}
//
//	   Fetch the "key" configuration value (for single layered
//	   configurations, where a sub-level dot cannot be used) or if any
//	   value containing one or more dots "." will be looked-up in the
//	   existing configuration that the method is called on. The
//	   underlying configuration is not changed and values are resolved
//	   each time ExpandString() is called. No locking of the
//	   configuration is done.
//
//	 ${key}
//
//	   "key" will be substituted with the value of the first matching
//	   key from the maps "values...", in the order passed to the
//	   function. If no "values" are passed (as opposed to the key not
//	   being found in any of the maps) then name is looked up
//	   as an environment variable, as 4. below.
//
//	 ${env:name}
//
//	   "name" will be substituted with the contents of the environment
//	   variable of the same name.
//
//	 The prefixes below are optional and enabled by default. They can be
//	 disabled using the option [config.NoExternalLookups]
//
//	 ${~/file} or ${/path/to/file} or ${file://path/to/file} or ${file:~/path/to/file}
//
//	   The contents of the referenced file will be read. Multiline files
//	   are used as-is; this can, for example, be used to read PEM
//	   certificate files or keys. If the path is prefixed with "~/" (or
//	   as an addition to a standard file url, if the first "/" is
//	   replaced with a tilde "~") then the path is relative to the home
//	   directory of the user running the process.
//
//	   Examples:
//
//	   * certfile ${file://etc/ssl/cert.pem}
//	   * template: ${file:~/templates/autogen.gotmpl}
//
//	 ${https://host/path} or ${http://host/path}
//
//	   The contents of the URL are fetched and used similarly as for
//	   local files above. The URL is passed to [http.Get] and supports
//	   proxies, embedded Basic Authentication and other features from
//	   that function.
//
// Additional custom lookup prefixes can be added with the [config.ExpandFunc]
// option.
//
// The bare form "$name" is NOT supported, unlike [os.Expand] as this
// can unexpectedly match values containing valid literal dollar signs.
//
// Expansion is not recursive. Configuration values are read and stored
// as literals and are expanded each time they are used. For each
// substitution any leading and trailing whitespace are removed.
// External sources are fetched each time they are used and so there may
// be a performance impact as well as the value unexpectedly changing
// during a process lifetime.
//
// Any errors (particularly from substitutions from external files or
// remote URLs) may result in an empty or corrupt string being returned.
// Error returns are intentionally discarded and an empty string
// substituted. Where a value contains multiple expandable items
// processing will continue even after an error for one of them.
//
// It is not currently possible to escape the syntax supported by
// ExpandString and if it is necessary to have a configuration value be
// a literal of the form "${name}" then you can set an otherwise unused
// item to the value and refer to it using the dotted syntax, e.g. for
// YAML
//
//	config:
//	  real: ${config.literal}
//	  literal: "${unchanged}"
//
// In the above a reference to ${config.real} will return the literal
// string ${unchanged} as there is no recursive lookups.
func ExpandString(input string, options ...ExpandOptions) (value string) {
	return global.ExpandString(input, options...)
}

// ExpandString works just like the package level [ExpandString] but on
// a specific config instance.
func (c *Config) ExpandString(input string, options ...ExpandOptions) (value string) {
	value = expand(input, func(s string) (r string) {
		if strings.HasPrefix(s, "enc:") {
			return c.expandEncodedString(s[4:], options...)
		}
		return c.expandString(s, options...)
	})
	return
}

// Expand behaves like [ExpandString] but returns a byte slice.
//
// This should be used where the return value may contain sensitive data
// and an immutable string cannot be destroyed after use.
func Expand(input string, options ...ExpandOptions) (value []byte) {
	return global.Expand(input, options...)
}

// Expand behaves like the [ExpandString] method but returns a byte
// slice.
func (c *Config) Expand(input string, options ...ExpandOptions) (value []byte) {
	value = expandBytes([]byte(input), func(s []byte) (r []byte) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			return c.expandEncodedBytes(s[4:], options...)
		}
		return []byte(c.expandString(string(s), options...))
	})
	return
}

// ExpandAllSettings returns all the settings from c applying
// ExpandString() to all string values and all string slice values.
// Further types may be added over time.
func (c *Config) ExpandAllSettings(options ...ExpandOptions) (all map[string]interface{}) {
	as := c.AllSettings()
	all = make(map[string]interface{}, len(as))

	for k, v := range as {
		switch ev := v.(type) {
		case string:
			all[k] = c.ExpandString(ev, options...)
		case []string:
			ns := []string{}
			for _, s := range ev {
				ns = append(ns, c.ExpandString(s, options...))
			}
			all[k] = ns
		default:
			all[k] = ev
		}
	}
	return

}

func (c *Config) expandEncodedString(s string, options ...ExpandOptions) (value string) {
	s = strings.TrimPrefix(s, "enc:")
	p := strings.SplitN(s, ":", 2)
	if len(p) != 2 {
		return ""
	}
	keyfiles, encodedValue := p[0], p[1]

	if !strings.HasPrefix(encodedValue, "+encs+") {
		encodedValue = c.expandString(encodedValue, options...)
	}
	if encodedValue == "" {
		return
	}
	encodedValue = strings.TrimPrefix(encodedValue, "+encs+")

	for _, keyfile := range strings.Split(keyfiles, "|") {
		if strings.HasPrefix(keyfile, "~/") {
			home, _ := os.UserHomeDir()
			keyfile = strings.Replace(keyfile, "~", home, 1)
		}
		a, err := ReadAESValuesFile(keyfile)
		if err != nil {
			continue
		}
		p, err := a.DecodeAESString(encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return ""
}

func (c *Config) expandEncodedBytes(s []byte, options ...ExpandOptions) (value []byte) {
	p := bytes.SplitN(s, []byte(":"), 2)
	if len(p) != 2 {
		return
	}
	keyfiles, encodedValue := p[0], p[1]

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		encodedValue = []byte(c.expandString(string(encodedValue), options...))
	}
	if len(encodedValue) == 0 {
		return
	}
	encodedBytes := bytes.TrimPrefix([]byte(encodedValue), []byte("+encs+"))

	for _, keyfile := range bytes.Split(keyfiles, []byte("|")) {
		if bytes.HasPrefix(keyfile, []byte("~/")) {
			home, _ := os.UserHomeDir()
			keyfile = bytes.Replace(keyfile, []byte("~"), []byte(home), 1)
		}
		a, err := ReadAESValuesFile(string(keyfile))
		if err != nil {
			continue
		}
		p, err := a.DecodeAES(encodedBytes)
		if err != nil {
			continue
		}
		return p
	}
	return
}

func (c *Config) expandString(s string, options ...ExpandOptions) (value string) {
	opts := evalExpandOptions(c, options...)
	switch {
	case strings.HasPrefix(s, "~/"), strings.HasPrefix(s, "/"):
		// check if defaults disabled
		if _, ok := opts.funcMaps["file"]; ok {
			return fetchFile(c, s)
		}
		return
	case strings.HasPrefix(s, "config:"):
		fallthrough
	case !strings.Contains(s, ":"):
		// note fallthrough from above, hence the check for prefix with
		// a "config:" in it.
		if strings.HasPrefix(s, "config:") || strings.Contains(s, ".") {
			s = strings.TrimPrefix(s, "config:")
			// this call to GetString() must NOT be recursive
			return strings.TrimSpace(c.Viper.GetString(s))
		}
		// only lookup env if there are no values maps, NOT if lookups
		// fail in any given maps
		if len(opts.lookupTables) == 0 {
			return strings.TrimSpace(mapEnv(s))
		}
		for _, v := range opts.lookupTables {
			if n, ok := v[s]; ok {
				return strings.TrimSpace(n)
			}
		}
		return
	case strings.HasPrefix(s, "env:"):
		return strings.TrimSpace(mapEnv(strings.TrimPrefix(s, "env:")))
	default:
		// check for any registered functions and call that with the
		// whole of the config string. there must be a ":" here, else
		// the above test would have picked it up. it is up to the
		// function called to trim whitespace, if required.
		f := strings.SplitN(s, ":", 2)
		if fn, ok := opts.funcMaps[f[0]]; ok {
			if opts.trimPrefix {
				return fn(c, f[1])
			}
			return fn(c, s)
		}
	}

	return
}

func fetchURL(cf *Config, url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func fetchFile(cf *Config, path string) string {
	path = strings.TrimPrefix(path, "file:")
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = strings.Replace(path, "~", home, 1)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// mapEnv is for special case mappings of environment variables across
// platforms. If a settings is not found via os.GetEnv() then defaults
// can be substituted. Currently only HOME is supported for Windows.
func mapEnv(e string) (s string) {
	if s = os.Getenv(e); s != "" {
		return
	}
	switch e {
	case "HOME":
		h, err := os.UserHomeDir()
		if err == nil {
			s = h
		}
	}
	return
}

// the below is copied from the Go source but modified to NOT support
// $val, only ${val}
//
// Copyright 2010 The Go Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// expand replaces ${var} in the string based on the mapping function.
func expand(s string, mapping func(string) string) string {
	var buf []byte
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getShellName(s[j+1:])
			if name == "" && w > 0 {
				// Encountered invalid syntax; eat the
				// characters.
			} else if name == "" {
				// Valid syntax, but $ was not followed by a
				// name. Leave the dollar character untouched.
				buf = append(buf, s[j])
			} else {
				buf = append(buf, mapping(name)...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return string(buf) + s[i:]
}

// as above but for byte slices directly
func expandBytes(s []byte, mapping func([]byte) []byte) []byte {
	var buf []byte
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getShellName(string(s[j+1:]))
			if name == "" && w > 0 {
				// Encountered invalid syntax; eat the
				// characters.
			} else if name == "" {
				// Valid syntax, but $ was not followed by a
				// name. Leave the dollar character untouched.
				buf = append(buf, s[j])
			} else {
				buf = append(buf, mapping([]byte(name))...)
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return append(buf, s[i:]...)
}

// isShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func isShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// getShellName returns the name that begins the string and the number of bytes
// consumed to extract it. If the name is enclosed in {}, it's part of a ${}
// expansion and two more bytes are needed than the length of the name.
//
// CHANGE: return if string does not start with an opening bracket
func getShellName(s string) (string, int) {
	if s[0] != '{' {
		// skip
		return "", 0
	}

	if len(s) > 2 && isShellSpecialVar(s[1]) && s[2] == '}' {
		return s[1:2], 3
	}

	// Scan to closing brace
	for i := 1; i < len(s); i++ {
		if s[i] == '}' {
			if i == 1 {
				return "", 2 // Bad syntax; eat "${}"
			}
			return s[1:i], i + 1
		}
	}
	return "", 1 // Bad syntax; eat "${"
}
