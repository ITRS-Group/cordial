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

// # Expandable Formats
//
// The Expand family of functions return the input with all occurrences
// of the form ${name} replaced using an [os.Expand]-like function (but
// without support for $name in the input) for the built-in and optional
// formats (in the order of priority) below. The caller can use options
// to define additional expansion functions based on a "prefix:",
// disable external lookups and also to pass in lookup tables referred
// to as value maps.
//
//	${enc:keyfile[|keyfile...]:encodedvalue}
//
//	  "encodedvalue" is an AES256 ciphertext in Geneos format - or, if
//	  not prefixed with `+encs+` then it is processed as an expandable
//	  string itself and can be a reference to another configuration key
//	  or a file or remote url containing one - which will be decoded
//	  using the key file(s) given. Each "keyfile" must be one of either
//	  an absolute path, a path relative to the working directory of the
//	  program, or if prefixed with "~/" then relative to the home
//	  directory of the user running the program. The first valid decode
//	  (see below) is returned.
//
//	  To minimise (but not wholly eliminate) any false-positive decodes
//	  that occur in some circumstances when using the wrong key file,
//	  the decoded value is only returned if it is a valid UTF-8 string
//	  as per [utf8.Valid].
//
//	  This prefix can be disabled with the `config.NoDecode()` option.
//
//	  Note: The keyfile(s) can be in Windows file path format as the
//	  fields for keyfiles and the encoded value are split based on the
//	  last colon (`:`) in the string and not any contained in a drive
//	  letter path. On Windows paths may also contain normal backslashes,
//	  they are not considered escape characters once extracted from the
//	  underlying configuration syntax (which may impose it's own rules,
//	  e.g. JSON)
//
//	  Examples:
//
//	  - password: ${enc:~/.keyfile:+encs+9F2C3871E105EC21E4F0D5A7921A937D}
//	  - password: ${enc:/etc/geneos/keyfile.aes:env:ENCODED_PASSWORD}
//	  - password: ${enc:~/.config/geneos/keyfile1.aes:app.password}
//	  - password: ${enc:~/.keyfile.aes:config:mySecret}
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
//	${key}
//
//	  "key" will be substituted with the value of the first matching key
//	  from the tables set using config.LookupTable(), in the order passed
//	  to the function. If no lookup tables are set (as opposed to the
//	  key not being found in any of the tables) then name is looked up
//	  as an environment variable, as below.
//
//	${env:name}
//
//	  "name" will be substituted with the contents of the environment
//	  variable of the same name. If no environment variable with name
//	  exists then the value returned is an empty string.
//
//	The additional prefixes below are enabled by default. They can be
//	disabled using the config.ExternalLookups() option.
//
//	${.../path/to/file} or ${~/file} or ${file://path/to/file} or ${file:~/path/to/file}
//
//	  The contents of the referenced file will be read. Multiline files
//	  are used as-is; this can, for example, be used to read PEM
//	  certificate files or keys. If the path is prefixed with `~/` (or
//	  as an addition to a standard file url, if the first `/` is
//	  replaced with a tilde `~`) then the path is relative to the home
//	  directory of the user running the process.
//
//	  Any name that contains a `/` but not a `:` will be treated as a
//	  file, if file reading is enabled. File paths can be absolute or
//	  relative to the working directory (or relative to the home
//	  directory, as above)
//
//	  For Windows file paths you must use the URL style `${file:...}`
//	  formats, otherwise any drive identifier will be treated as a
//	  partial expansion type and cause unexpected behaviour.
//
//	  Examples:
//
//	  - certfile: ${file://etc/ssl/cert.pem}
//	  - template: ${file:~/templates/autogen.gotmpl}
//	  - relative: ${./file.txt}
//
//	${https://host/path} or ${http://host/path}
//
//	  The contents of the URL are fetched and used similarly as for
//	  local files above. The URL is passed to [http.Get] and supports
//	  proxies, embedded Basic Authentication and other features from
//	  that function.
//
//	The prefix below can be enabled with the config.Expressions() option.
//
//	${expr:EXPRESSION}
//
//	  EXPRESSION is evaluated using https://github.com/maja42/goval. Inside
//	  the expression all configuration items are available as variables
//	  with the top level map `env` set to the environment variables
//	  available. All results are returned as strings. An empty string
//	  may mean there was an error in evaluating the expression.
//
// Additional custom prefixes can be added with the config.Prefix()
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
// remote URLs) will result in an empty or corrupt string being
// returned. Error returns are intentionally discarded and an empty
// string substituted. Where a value contains multiple expandable items
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
package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/maja42/goval"
)

// ExpandString returns the global configuration value for input as an
// expanded string. The returned string is always a freshly allocated
// value.
func ExpandString(input string, options ...ExpandOptions) (value string) {
	return global.ExpandString(input, options...)
}

// ExpandString returns the configuration c value for input as an
// expanded string. The returned string is always a freshly allocated
// value.
func (c *Config) ExpandString(input string, options ...ExpandOptions) (value string) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return strings.Clone(input)
		}
		// return a *copy* of the initialValue or defaultValue
		if opts.initialValue != nil {
			return fmt.Sprint(opts.initialValue)
		}
		return fmt.Sprint(opts.defaultValue)
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expand(input, func(s string) (r string) {
		if strings.HasPrefix(s, "enc:") {
			if opts.nodecode {
				// return string and restore containing ${...}
				return `${` + s + `}`
			}
			return c.expandEncodedString(s[4:], options...)
		}
		r, _ = c.ExpandRawString(s, options...)
		return
	})

	if opts.trimSpace {
		value = strings.TrimSpace(value)
	}

	if value == "" {
		value = fmt.Sprint(opts.defaultValue)
	}

	// return a clone
	return strings.Clone(value)
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func ExpandStringSlice(input []string, options ...ExpandOptions) []string {
	return global.ExpandStringSlice(input, options...)
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func (c *Config) ExpandStringSlice(input []string, options ...ExpandOptions) (vals []string) {
	for _, v := range input {
		vals = append(vals, c.ExpandString(v, options...))
	}
	return
}

// Expand behaves like ExpandString but returns a byte slice.
//
// This should be used where the return value may contain sensitive data
// and an immutable string cannot be destroyed after use.
func Expand(input string, options ...ExpandOptions) (value []byte) {
	return global.Expand(input, options...)
}

// Expand behaves like the ExpandString method but returns a byte
// slice.
func (c *Config) Expand(input string, options ...ExpandOptions) (value []byte) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return bytes.Clone([]byte(input))
		}
		if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return bytes.Clone(b)
			}
			return fmt.Append(value, opts.initialValue)
		}
		return fmt.Append(value, opts.defaultValue)
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandBytes([]byte(input), func(s []byte) (r []byte) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				// return string and restore containing ${...}
				return fmt.Append([]byte{}, `${`, s, `}`)
			}
			return c.expandEncodedBytes(s[4:], options...)
		}
		str, _ := c.ExpandRawString(string(s), options...)
		return []byte(str)
	})

	if opts.trimSpace {
		value = bytes.TrimSpace(value)
	}

	if len(value) == 0 {
		value = []byte(fmt.Sprint(opts.defaultValue))
	}

	return
}

// ExpandPassword expands the input string and returns a *Plaintext. The
// TrimSPace option is ignored.
func ExpandToPassword(input string, options ...ExpandOptions) *Plaintext {
	return &Plaintext{global.ExpandToEnclave(input, options...)}
}

// ExpandPassword expands the input string and returns a *Plaintext. The
// TrimSPace option is ignored.
func (c *Config) ExpandToPassword(input string, options ...ExpandOptions) *Plaintext {
	return &Plaintext{c.ExpandToEnclave(input, options...)}
}

// ExpandToEnclave expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func ExpandToEnclave(input string, options ...ExpandOptions) *memguard.Enclave {
	return global.ExpandToEnclave(input, options...)
}

// ExpandToEnclave expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) ExpandToEnclave(input string, options ...ExpandOptions) (value *memguard.Enclave) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return memguard.NewEnclave([]byte(input))
		}

		// fallback to any default value or, failing that, an initial value
		if opts.defaultValue != nil {
			return memguard.NewEnclave([]byte(fmt.Sprint(opts.defaultValue)))
		} else if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return memguard.NewEnclave(b)
			} else {
				return memguard.NewEnclave([]byte(fmt.Sprint(opts.initialValue)))
			}
		}
		return &memguard.Enclave{}
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandToEnclave([]byte(input), func(s []byte) (r *memguard.Enclave) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				return memguard.NewEnclave(fmt.Append([]byte{}, `${`, s, `}`))
			}
			return c.expandEncodedBytesEnclave(s[4:], options...)
		}
		str, _ := c.ExpandRawString(string(s), options...)
		return memguard.NewEnclave([]byte(str))
	})

	if value == nil || value.Size() == 0 {
		// return a *copy* of the defaultvalue, don't let memguard wipe it!
		return memguard.NewEnclave([]byte(fmt.Sprint(opts.defaultValue)))
	}

	return
}

// ExpandToLockedBuffer expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func ExpandToLockedBuffer(input string, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	return global.ExpandToLockedBuffer(input, options...)
}

// ExpandToLockedBuffer expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) ExpandToLockedBuffer(input string, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return memguard.NewBufferFromBytes([]byte(input))
		}
		if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return memguard.NewBufferFromBytes(b)
			}
			return memguard.NewBufferFromBytes([]byte(fmt.Sprint(opts.initialValue)))
		}
		return memguard.NewBufferFromBytes([]byte(fmt.Sprint(opts.defaultValue)))
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandToLockedBuffer([]byte(input), func(s []byte) *memguard.LockedBuffer {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				return memguard.NewBufferFromBytes(fmt.Append([]byte{}, `${`, s, `}`))
			}
			return c.expandEncodedBytesLockedBuffer(s[4:], options...)
		}
		str, _ := c.ExpandRawString(string(s), options...)
		return memguard.NewBufferFromBytes([]byte(str))
	})

	if value == nil || value.Size() == 0 {
		// return a *copy* of the defaultvalue, don't let memguard wipe it!
		return memguard.NewBufferFromBytes([]byte(fmt.Sprint(opts.defaultValue)))
	}

	return
}

// ExpandAllSettings returns all the settings from config structure c
// applying ExpandString to all string values and all string slice
// values. Non-string types are left unchanged. Further types, e.g. maps
// of strings, may be added in future releases.
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

// splitEncFields breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext. The split is done on the
// *last* colon and not the first, otherwise Windows drive letter paths
// would be considered the split point.
func splitEncFields(enc string) (keyfiles, ciphertext string) {
	c := strings.LastIndexByte(enc, ':')
	if c == -1 {
		return
	}
	keyfiles = enc[:c]
	if len(enc) > c+1 {
		ciphertext = enc[c+1:]
	}
	return
}

// splitEncFieldsBytes breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext. The split is done on the
// *last* colon and not the first, otherwise Windows drive letter paths
// would be considered the split point.
func splitEncFieldsBytes(enc []byte) (keyfiles, ciphertext []byte) {
	c := bytes.LastIndexByte(enc, ':')
	if c == -1 {
		return
	}
	keyfiles = enc[:c]
	if len(enc) > c+1 {
		ciphertext = enc[c+1:]
	}
	return
}

// expandEncodedString accepts input of the form:
//
//	[enc:]keyfile[,keyfile...]:[+encs+HEX|external]
//
// Each keyfile is tried until the first that does not return a decoding
// error. `keyfile` may be prefixed `~/` in which case the file is
// relative to the user's home directory. If the encoded string is
// prefixed with `+encs+` (standard Geneos usage) then it is used
// directly, otherwise the value is looked-up using the normal
// conventions for external access, e.g. file or URL.
func (c *Config) expandEncodedString(s string, options ...ExpandOptions) (value string) {
	keyfiles, encodedValue := splitEncFields(s)

	if !strings.HasPrefix(encodedValue, "+encs+") {
		encodedValue, _ = c.ExpandRawString(encodedValue, options...)
	}
	if encodedValue == "" {
		return
	}

	for _, k := range strings.Split(keyfiles, "|") {
		keyfile := KeyFile(ExpandHome(k))
		p, err := keyfile.DecodeString(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return ""
}

func (c *Config) expandEncodedBytes(s []byte, options ...ExpandOptions) (value []byte) {
	keyfiles, encodedValue := splitEncFieldsBytes(s)

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.ExpandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for _, k := range bytes.Split(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.Decode(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return
}

func (c *Config) expandEncodedBytesEnclave(s []byte, options ...ExpandOptions) (value *memguard.Enclave) {
	keyfiles, encodedValue := splitEncFieldsBytes(s)

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.ExpandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for _, k := range bytes.Split(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.DecodeEnclave(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return
}

func (c *Config) expandEncodedBytesLockedBuffer(s []byte, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	keyfiles, encodedValue := splitEncFieldsBytes(s)

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.ExpandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for _, k := range bytes.Split(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.DecodeEnclave(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		value, _ = p.Open()
		return
	}
	return
}

// ExpandRawString expands the string s using the same rules and options
// as [ExpandString] but treats the whole of s as if it were wrapped in
// '${...}'. The function does most of the core work for configuration
// expansion but is also exported for use without the decoration
// required for configuration values, allowing use against command line
// flag values, for example.
func (c *Config) ExpandRawString(s string, options ...ExpandOptions) (value string, err error) {
	opts := evalExpandOptions(c, options...)
	switch {
	case strings.Contains(s, "/") && !strings.Contains(s, ":"):
		// check if defaults disabled
		if _, ok := opts.funcMaps["file"]; ok {
			if _, err = os.Stat(s); err != nil {
				return
			}
			return fetchFile(c, s, opts.trimSpace)
		}
		return
	case strings.HasPrefix(s, "config:"), !strings.Contains(s, ":"):
		if strings.HasPrefix(s, "config:") || strings.Contains(s, ".") {
			s = strings.TrimPrefix(s, "config:")
			// this call to GetString() must NOT be recursive
			c.mutex.RLock()
			value = c.Viper.GetString(s)
			c.mutex.RUnlock()
			if opts.trimSpace {
				value = strings.TrimSpace(value)
			}
			return
		}

		// only lookup env if there are no values maps, NOT if lookups
		// fail in any given maps
		if len(opts.lookupTables) == 0 {
			value = mapEnv(s)
			if opts.trimSpace {
				value = strings.TrimSpace(value)
			}
			return
		}

		for _, v := range opts.lookupTables {
			if n, ok := v[s]; ok {
				value = n
				if opts.trimSpace {
					value = strings.TrimSpace(value)
				}
				return
			}
		}

		return
	case strings.HasPrefix(s, "env:"):
		value = mapEnv(strings.TrimPrefix(s, "env:"))
		if opts.trimSpace {
			value = strings.TrimSpace(value)
		}
		return
	default:
		// check for any registered functions and call that with the
		// whole of the config string. there must be a ":" here, else
		// the above test would have picked it up. it is up to the
		// function called to trim whitespace, if required.
		f := strings.SplitN(s, ":", 2)
		if fn, ok := opts.funcMaps[f[0]]; ok {
			if opts.trimPrefix {
				value, err = fn(c, f[1], opts.trimSpace)
			} else {
				value, err = fn(c, s, opts.trimSpace)
			}
			return
		}
	}

	return
}

func fetchURL(cf *Config, url string, trim bool) (s string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if !trim {
		s = string(b)
	} else {
		s = strings.TrimSpace(string(b))
	}
	return
}

func fetchFile(_ *Config, p string, trim bool) (s string, err error) {
	b, err := os.ReadFile(ExpandHome(strings.TrimPrefix(p, "file:")))
	if err != nil {
		return
	}
	if !trim {
		s = string(b)
	} else {
		s = strings.TrimSpace(string(b))
	}
	return
}

func expr(cf *Config, expression string, trim bool) (s string, err error) {
	eval := goval.NewEvaluator()
	vars := cf.AllSettings()
	env := make(map[string]string)
	for _, e := range os.Environ() {
		s := strings.SplitN(e, "=", 2)
		env[s[0]] = s[1]
	}
	vars["env"] = env
	result, err := eval.Evaluate(expression, vars, nil)
	if err != nil {
		return
	}
	if !trim {
		s = fmt.Sprint(result)
	} else {
		s = strings.TrimSpace(fmt.Sprint(result))
	}
	return
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
		h, err := UserHomeDir()
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
	// ${} is all UTF-8, so bytes are fine for this operation.
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

func expandToEnclave(s []byte, mapping func([]byte) *memguard.Enclave) *memguard.Enclave {
	var buf []byte
	// ${} is all UTF-8, so bytes are fine for this operation.
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
				e := mapping([]byte(name))
				if e != nil {
					l, _ := e.Open()
					buf = append(buf, l.Bytes()...)
					l.Destroy()
				}
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		// if no expansion, return as is in enclave
		return memguard.NewEnclave(s)
	}

	buf = append(buf, s[i:]...)
	return memguard.NewEnclave(buf)
}

func expandToLockedBuffer(s []byte, mapping func([]byte) *memguard.LockedBuffer) *memguard.LockedBuffer {
	var buf []byte
	// ${} is all UTF-8, so bytes are fine for this operation.
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
				e := mapping([]byte(name))
				if e != nil {
					buf = append(buf, e.Bytes()...)
					e.Destroy()
				}
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		// if no expansion, return as is in enclave
		return memguard.NewBufferFromBytes(s)
	}

	buf = append(buf, s[i:]...)
	return memguard.NewBufferFromBytes(buf)
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
