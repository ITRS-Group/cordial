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
	"github.com/awnumar/memguard"
)

// ExpandString returns the global configuration value for input as an
// expanded string. The returned string is always a freshly allocated
// value.
func ExpandString(input string, options ...ExpandOptions) (value string) {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.expandString(input, options...)
}

// ExpandString returns the configuration c value for input as an
// expanded string. The returned string is always a freshly allocated
// value.
func (c *Config) ExpandString(input string, options ...ExpandOptions) (value string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expandString(input, options...)
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func ExpandStringSlice(input []string, options ...ExpandOptions) []string {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.expandStringSlice(input, options...)
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func (c *Config) ExpandStringSlice(input []string, options ...ExpandOptions) (vals []string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expandStringSlice(input, options...)
}

// Expand behaves like ExpandString but returns a byte slice.
//
// This should be used where the return value may contain sensitive data
// and an immutable string cannot be destroyed after use.
func Expand(input string, options ...ExpandOptions) (value []byte) {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.expand(input, options...)
}

// Expand behaves like the ExpandString method but returns a byte
// slice.
func (c *Config) Expand(input string, options ...ExpandOptions) (value []byte) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expand(input, options...)
}

// ExpandPassword expands the input string and returns a *Plaintext. The
// TrimSPace option is ignored.
func ExpandToPassword(input string, options ...ExpandOptions) *Plaintext {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return &Plaintext{global.expandToEnclave(input, options...)}
}

// ExpandPassword expands the input string and returns a *Plaintext. The
// TrimSPace option is ignored.
func (c *Config) ExpandToPassword(input string, options ...ExpandOptions) *Plaintext {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return &Plaintext{c.expandToEnclave(input, options...)}
}

// ExpandToEnclave expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func ExpandToEnclave(input string, options ...ExpandOptions) *memguard.Enclave {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.expandToEnclave(input, options...)
}

// ExpandToEnclave expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) ExpandToEnclave(input string, options ...ExpandOptions) (value *memguard.Enclave) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expandToEnclave(input, options...)
}

// ExpandToLockedBuffer expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func ExpandToLockedBuffer(input string, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.expandToLockedBuffer(input, options...)
}

// ExpandToLockedBuffer expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) ExpandToLockedBuffer(input string, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expandToLockedBuffer(input, options...)
}

// ExpandAllSettings returns all the settings from config structure c
// applying ExpandString to all string values and all string slice
// values. Non-string types are left unchanged. Further types, e.g. maps
// of strings, may be added in future releases.
func (c *Config) ExpandAllSettings(options ...ExpandOptions) (all map[string]any) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.expandAllSettings(options...)
}

// the functions below are based on code copied from the Go sources but
// modified to NOT support $val, only ${val}
//
// Copyright 2010 The Go Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in
// the LICENSE file.
//
// _expandString replaces ${var} in the string based on the mapping function.
func _expandString(s string, mapping func(string) string) string {
	var buf []byte
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getContents(s[j+1:])
			if name == "" {
				if w == 1 {
					// if invalid after opening `${` then return them
					// unchanged
					buf = append(buf, s[j:j+2]...)
				} else if w > 0 {
					// Encountered invalid syntax; eat the
					// characters.
				} else {
					// Valid syntax, but $ was not followed by a
					// name. Leave the dollar character untouched.
					buf = append(buf, s[j])
				}
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
func _expandBytes(s []byte, mapping func([]byte) []byte) []byte {
	var buf []byte
	// ${} is all UTF-8, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getContents(string(s[j+1:]))
			if name == "" {
				if w == 1 {
					// if invalid after opening `${` then return them
					// unchanged
					buf = append(buf, s[j:j+2]...)
				} else if w > 0 {
					// Encountered invalid syntax; eat the
					// characters.
				} else {
					// Valid syntax, but $ was not followed by a
					// name. Leave the dollar character untouched.
					buf = append(buf, s[j])
				}
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

func _expandToEnclave(s []byte, mapping func([]byte) *memguard.Enclave) *memguard.Enclave {
	var buf []byte
	// ${} is all UTF-8, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getContents(string(s[j+1:]))
			if name == "" {
				if w == 1 {
					// if invalid after opening `${` then return them
					// unchanged
					buf = append(buf, s[j:j+2]...)
				} else if w > 0 {
					// Encountered invalid syntax; eat the
					// characters.
				} else {
					// Valid syntax, but $ was not followed by a
					// name. Leave the dollar character untouched.
					buf = append(buf, s[j])
				}
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

func _expandToLockedBuffer(s []byte, mapping func([]byte) *memguard.LockedBuffer) *memguard.LockedBuffer {
	var buf []byte
	// ${} is all UTF-8, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getContents(string(s[j+1:]))
			if name == "" {
				if w == 1 {
					// if invalid after opening `${` then return them
					// unchanged
					buf = append(buf, s[j:j+2]...)
				} else if w > 0 {
					// Encountered invalid syntax; eat the
					// characters.
				} else {
					// Valid syntax, but $ was not followed by a
					// name. Leave the dollar character untouched.
					buf = append(buf, s[j])
				}
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

// getContents returns the string inside braces, checking for embedded
// braces and the number of bytes consumed to extract it. The contents
// must be enclosed in {} and two more bytes are needed than the length
// of the name.
//
// CHANGE: return if string does not start with an opening bracket
//
// CHANGE: skip any character after a backslash, including closing
// braces
//
// CHANGE: match embedded opening braces and closing ones inside the
// string.
func getContents(s string) (string, int) {
	// must start with an opening brace
	if s[0] != '{' {
		// skip
		return "", 0
	}

	// Scan to closing brace, skipping backslash+next and stacking opening braces
	var depth int
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
				continue
			}
			if i == 1 {
				return "", 2 // Bad syntax; eat "${}"
			}
			return s[1:i], i + 1
		default:
		}
	}
	return "", 1 // Bad syntax; eat "${"
}
