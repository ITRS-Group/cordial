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

package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config embeds Viper and also exposes the config type used
type Config struct {
	*viper.Viper
	Type string
}

var global *Config

func init() {
	global = &Config{Viper: viper.New()}
}

// Returns the configuration item as a string with ExpandString() applied,
// passing the first "values" if given
func GetString(s string, values ...map[string]string) string {
	return global.GetString(s, values...)
}

// Returns the configuration item as a string with ExpandString() applied,
// passing the first "values" if given
func (c *Config) GetString(s string, values ...map[string]string) string {
	return c.ExpandString(c.Viper.GetString(s), values...)
}

func GetConfig() *Config {
	return global
}

func New() *Config {
	return &Config{Viper: viper.New()}
}

func (c *Config) Sub(key string) *Config {
	return &Config{Viper: c.Viper.Sub(key)}
}

func GetStringSlice(s string, values ...map[string]string) []string {
	return global.GetStringSlice(s, values...)
}

func (c *Config) GetStringSlice(s string, values ...map[string]string) (slice []string) {
	r := c.Viper.GetStringSlice(s)
	for _, n := range r {
		slice = append(slice, c.ExpandString(n, values...))
	}
	return
}

func GetStringMapString(s string, values ...map[string]string) map[string]string {
	return global.GetStringMapString(s, values...)
}

func (c *Config) GetStringMapString(s string, values ...map[string]string) (m map[string]string) {
	m = make(map[string]string)
	r := c.Viper.GetStringMapString(s)
	for k, v := range r {
		m[k] = c.ExpandString(v, values...)
	}
	return m
}

// ExpandString() returns the input with all occurrences of the form
// ${name} replaced using an [os.Expand]-like function (but without
// support for bare names) for the formats (in the order of priority)
// below:
//
//  1. "${enc:keyfile[|keyfile...]:encodedvalue}"
//
//     The item "encodedvalue" is an AES256 ciphertext in Geneos format
//     - or a reference to one - which will be decoded using the key
//     file(s) given. Each "keyfile" must be one of either an absolute
//     path, a path relative to the working directory of the program, or
//     if prefixed with "~/" then relative to the home directory of the
//     user running the program. The first valid decode (see below) is
//     returned.
//
//     The "encodedvalue" must be either prefixed "+encs+" to align with
//     Geneos or will otherwise be looked up using the forms of any of
//     the other references below, but without the surrounding
//     dollar-brackets "${...}".
//
//     To minimise (but not wholly eliminate) any false-positive decodes
//     that occur in some circumstances when using the wrong key file,
//     the decoded value is only returned if it is a valid UTF-8 string
//     as per [utf8.Valid].
//
//     Examples:
//
//     * password: ${enc:~/.keyfile:+encs+9F2C3871E105EC21E4F0D5A7921A937D}
//     * password: ${enc:/etc/geneos/keyfile.aes:env:ENCODED_PASSWORD}
//     * password: ${enc:~/.config/geneos/keyfile1.aes:env:app.password}
//     * password: ${enc:~/.keyfile.aes:env:config:mySecret}
//
//  2. "${config:key}" or "${path.to.config}"
//
//     Fetch the "key" configuration value (for single layered
//     configurations, where a sub-level dot cannot be used) or if any
//     value containing one or more dots "." will be looked-up in the
//     existing configuration that the method is called on. The
//     underlying configuration is not changed and values are resolved
//     each time ExpandString() is called. No locking of the
//     configuration is done.
//
//  3. "${key}"
//
//     "key" will be substituted with the value of the first matching
//     key from the maps "values...", in the order passed to the
//     function. If no "values" are passed (as opposed to the key not
//     being found in any of the maps) then name is looked up
//     as an environment variable, as 4. below.
//
//  4. "${env:name}"
//
//     "name" will be substituted with the contents of the environment
//     variable of the same name.
//
//  5. "${file://path/to/file}" or "${file:~/path/to/file}"
//
//     The contents of the referenced file will be read. Multiline
//     files are used as-is; this can, for example, be used to read
//     PEM certificate files or keys. As an addition to a standard
//     file url, if the first "/" is replaced with a tilde "~" then the
//     path is relative to the home directory of the user running the process.
//
//     Examples:
//
//     * certfile ${file://etc/ssl/cert.pem}
//     * template: ${file:~/templates/autogen.gotmpl}
//
//  6. "${https://host/path}" or "${http://host/path}"
//
//     The contents of the URL are fetched and used similarly as for
//     local files above. The URL is passed to [http.Get] and supports
//     proxies, embedded Basic Authentication and other features from
//     that function.
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
func (c *Config) ExpandString(input string, values ...map[string]string) (value string) {
	value = expand(input, func(s string) (r string) {
		if strings.HasPrefix(s, "enc:") {
			return c.expandEncodedString(s[4:], values...)
		}
		return c.expandString(s, values...)
	})
	return
}

// ExpandAllSettings returns all the settings from c applying
// ExpandString() to all string values and all string slice values.
// "values" maps are passed to ExpandString as-is.
// Futher types may be added over time.
func (c *Config) ExpandAllSettings(values ...map[string]string) (all map[string]interface{}) {
	as := c.AllSettings()
	all = make(map[string]interface{}, len(as))

	for k, v := range as {
		switch ev := v.(type) {
		case string:
			all[k] = c.ExpandString(ev, values...)
		case []string:
			ns := []string{}
			for _, s := range ev {
				ns = append(ns, c.ExpandString(s, values...))
			}
			all[k] = ns
		default:
			all[k] = ev
		}
	}
	return

}

func (c *Config) expandEncodedString(s string, values ...map[string]string) (value string) {
	p := strings.SplitN(s, ":", 2)
	if len(p) != 2 {
		return ""
	}
	keyfiles, encodedValue := p[0], p[1]

	if !strings.HasPrefix(encodedValue, "+encs+") {
		encodedValue = c.expandString(encodedValue, values...)
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

func (c *Config) expandString(s string, values ...map[string]string) (value string) {
	switch {
	case strings.HasPrefix(s, "config:"):
		fallthrough
	case !strings.Contains(s, ":"):
		if strings.HasPrefix(s, "config:") || strings.Contains(s, ".") {
			s = strings.TrimPrefix(s, "config:")
			// this call to GetString() must NOT be recursive
			return strings.TrimSpace(c.Viper.GetString(s))
		}
		// only lookup env if there are no values maps, NOT if lookups
		// fail in any given maps
		if len(values) == 0 {
			return strings.TrimSpace(mapEnv(s))
		}
		for _, v := range values {
			if n, ok := v[s]; ok {
				return strings.TrimSpace(n)
			}
		}
		return ""
	case strings.HasPrefix(s, "env:"):
		return strings.TrimSpace(mapEnv(strings.TrimPrefix(s, "env:")))
	case strings.HasPrefix(s, "file:"):
		path := strings.TrimPrefix(s, "file:")
		if strings.HasPrefix(path, "~/") {
			home, _ := os.UserHomeDir()
			path = strings.Replace(path, "~", home, 1)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return
		}
		return strings.TrimSpace(string(b))
	case strings.HasPrefix(s, "http:"), strings.HasPrefix(s, "https:"):
		resp, err := http.Get(s)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		return strings.TrimSpace(string(b))
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
		h, err := os.UserHomeDir()
		if err == nil {
			s = h
		}
	}
	return
}

// LoadConfig loads configuration files from internal defaults, external
// defaults and the given configuration file. The configuration file can
// be passed as an option. Each layer is only loaded once, if given.
// Internal defaults are passed as a []byte intended to be loaded from
// an embedded file. External defaults and the main configuration file
// are passed as ordered slices of strings. The first match is loaded.
//
//	LoadConfig("geneos")
//
//	//go:embed somefile.json
//	var myDefaults []byte
//	LoadConfig("geneos", config.SetDefaults(myDefaults, "json"), )
//
// Options can be passed to change the default behaviour and to pass any
// embedded defaults or an existing viper.
//
// for defaults see:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
// ... find windows equiv
func LoadConfig(configName string, options ...Options) (c *Config, err error) {
	opts := &configOptions{}
	evalOptions(configName, opts, options...)

	if opts.useglobal {
		c = global
	} else {
		c = New()
	}

	internalDefaults := make(map[string]interface{})
	if len(opts.defaults) > 0 {
		defaults := viper.New()
		buf := bytes.NewBuffer(opts.defaults)
		defaults.SetConfigType(opts.defaultsFormat)
		// ignore errors
		defaults.ReadConfig(buf)
		internalDefaults = defaults.AllSettings()
	}

	defaults := viper.New()
	for k, v := range internalDefaults {
		defaults.SetDefault(k, v)
	}

	confDirs := opts.configDirs

	if !opts.ignoreworkingdir {
		confDirs = append(confDirs, ".")
	}
	if !opts.ignoreuserconfdir {
		userConfDir, err := os.UserConfigDir()
		if err == nil {
			confDirs = append(confDirs, filepath.Join(userConfDir, opts.appname))
		}
	}
	systemConfDir := "/etc"
	if !opts.ignoresystemdir {
		confDirs = append(confDirs, filepath.Join(systemConfDir, opts.appname))
	}

	if len(confDirs) > 0 {
		for _, d := range confDirs {
			defaults.AddConfigPath(d)
		}

		defaults.SetConfigName(configName + ".defaults")
		defaults.ReadInConfig()
		if err = defaults.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// not found is fine
				err = nil
			} else {
				return c, fmt.Errorf("default config: %w", err)
			}
		}
		defaultSettings := defaults.AllSettings()

		for k, v := range defaultSettings {
			c.Viper.SetDefault(k, v)
		}
	}

	if opts.configFile != "" {
		c.Viper.SetConfigFile(opts.configFile)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// not found is fine
				err = nil
			} else {
				return c, fmt.Errorf("reading %s: %w", opts.configFile, err)
			}
		}
	} else if len(confDirs) > 0 {
		for _, d := range confDirs {
			c.Viper.AddConfigPath(d)
		}
		c.Viper.SetConfigName(configName)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// not found is fine
				err = nil
			} else {
				return c, fmt.Errorf("reading %s: %w", c.Viper.ConfigFileUsed(), err)
			}
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
