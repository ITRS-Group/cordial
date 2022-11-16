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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/gurkankaymak/hocon"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/term"
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

// GetString functions like [viper.GetString] but additionally calls
// [ExpandString] with the configuration value, passing any "values" maps
func GetString(s string, values ...map[string]string) string {
	return global.GetString(s, values...)
}

// GetString functions like [viper.GetString] on a Config instance, but
// additionally calls [ExpandString] with the configuration value, passing
// any "values" maps
func (c *Config) GetString(s string, values ...map[string]string) string {
	return c.ExpandString(c.Viper.GetString(s), values...)
}

// GetByteSlice functions like [viper.GetString] but additionally calls
// [Expand] with the configuration value, passing any "values" maps and
// returning a byte slice
func GetByteSlice(s string, values ...map[string]string) []byte {
	return global.GetByteSlice(s, values...)
}

// GetByteSlice functions like [viper.GetString] on a Config instance, but
// additionally calls [Expand] with the configuration value, passing
// any "values" maps and returning a byte slice
func (c *Config) GetByteSlice(s string, values ...map[string]string) []byte {
	return c.Expand(c.Viper.GetString(s), values...)
}

// GetConfig returns the global Config instance
func GetConfig() *Config {
	return global
}

// New returns a Config instance initialised with a new viper instance
func New() *Config {
	return &Config{Viper: viper.New()}
}

// Sub returns a Config instance for the sub-key passed
func (c *Config) Sub(key string) *Config {
	return &Config{Viper: c.Viper.Sub(key)}
}

// GetStringSlice functions like [viper.GetStringSlice] but additionally calls
// [ExpandString] on each element of the slice, passing any "values" maps
func GetStringSlice(s string, values ...map[string]string) []string {
	return global.GetStringSlice(s, values...)
}

// GetStringSlice functions like [viper.GetStringSlice] on a Config
// instance but additionally calls [ExpandString] on each element of the
// slice, passing any "values" maps
func (c *Config) GetStringSlice(s string, values ...map[string]string) (slice []string) {
	r := c.Viper.GetStringSlice(s)
	for _, n := range r {
		slice = append(slice, c.ExpandString(n, values...))
	}
	return
}

// GetStringMapString functions like [viper.GetStringMapString] but additionally calls
// [ExpandString] on each value element of the map, passing any "values" maps
func GetStringMapString(s string, values ...map[string]string) map[string]string {
	return global.GetStringMapString(s, values...)
}

// GetStringMapString functions like [viper.GetStringMapString] on a
// Config instance but additionally calls [ExpandString] on each value
// element of the map, passing any "values" maps
func (c *Config) GetStringMapString(s string, values ...map[string]string) (m map[string]string) {
	m = make(map[string]string)
	r := c.Viper.GetStringMapString(s)
	for k, v := range r {
		m[k] = c.ExpandString(v, values...)
	}
	return m
}

// ExpandString returns the input with all occurrences of the form
// ${name} replaced using an [os.Expand]-like function (but without
// support for bare names) for the formats (in the order of priority)
// below. It operates on the global config instance, referencing any
// other configuration values in the global package context.
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
func ExpandString(input string, values ...map[string]string) (value string) {
	return global.ExpandString(input, values...)
}

// ExpandString works just like the package level [ExpandString] but on
// a specific config instance.
func (c *Config) ExpandString(input string, values ...map[string]string) (value string) {
	value = expand(input, func(s string) (r string) {
		if strings.HasPrefix(s, "enc:") {
			return c.expandEncodedString(s[4:], values...)
		}
		return c.expandString(s, values...)
	})
	return
}

// Expand behaves like [ExpandString] but returns a byte slice.
//
// This should be used where the return value may contain sensitive data
// and an immutable string cannot be destroyed after use.
func Expand(input string, values ...map[string]string) (value []byte) {
	return global.Expand(input, values...)
}

// Expand behaves like the [ExpandString] method but returns a byte
// slice.
func (c *Config) Expand(input string, values ...map[string]string) (value []byte) {
	value = expandBytes([]byte(input), func(s []byte) (r []byte) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			return c.expandEncodedBytes(s[4:], values...)
		}
		return []byte(c.expandString(string(s), values...))
	})
	return
}

// ExpandAllSettings returns all the settings from c applying
// ExpandString() to all string values and all string slice values.
// "values" maps are passed to ExpandString as-is. Further types may be
// added over time.
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

func (c *Config) expandEncodedBytes(s []byte, values ...map[string]string) (value []byte) {
	p := bytes.SplitN(s, []byte(":"), 2)
	if len(p) != 2 {
		return
	}
	keyfiles, encodedValue := p[0], p[1]

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		encodedValue = []byte(c.expandString(string(encodedValue), values...))
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
	case strings.HasPrefix(s, "~/"), strings.HasPrefix(s, "/"), strings.HasPrefix(s, "file:"):
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
// Internal defaults are passed as a byte slice - this is typically
// loaded from an embedded file but can be supplied from any source.
// External defaults and the main configuration file are passed as
// ordered slices of strings. The first match is loaded.
//
//	LoadConfig("geneos")
//
//	//go:embed somefile.json
//	var myDefaults []byte
//	LoadConfig("geneos", config.SetDefaults(myDefaults, "json"), config.SetConfigFile(configPath))
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

	defaults := viper.New()
	internalDefaults := viper.New()

	if len(opts.defaults) > 0 {
		buf := bytes.NewBuffer(opts.defaults)
		internalDefaults.SetConfigType(opts.defaultsFormat)
		// ignore errors ?
		internalDefaults.ReadConfig(buf)
	}

	// now set any internal default values as real defaults, cannot use Merge here
	for k, v := range internalDefaults.AllSettings() {
		defaults.SetDefault(k, v)
	}

	confDirs := opts.configDirs

	// add config directories in order - first match wins unless
	// MergeSettings() option is used
	if opts.workingdir != "" {
		confDirs = append(confDirs, opts.workingdir)
	}
	if opts.userconfdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.userconfdir, opts.appname))
	}
	if opts.systemdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.systemdir, opts.appname))
	}

	// if ew are merging, then we load in reverse order
	if opts.merge {
		for i := len(confDirs)/2 - 1; i >= 0; i-- {
			opp := len(confDirs) - 1 - i
			confDirs[i], confDirs[opp] = confDirs[opp], confDirs[i]
		}
	}
	log.Debug().Msgf("confDirs: %v", confDirs)

	// search directories for defaults. we do this even if the config
	// file itself is set using option SetConfigFile()
	if opts.merge {
		for _, dir := range confDirs {
			d := viper.New()
			d.AddConfigPath(dir)
			d.SetConfigName(configName + ".defaults")
			d.ReadInConfig()
			if err = d.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					// not found is fine
					continue
				} else {
					return c, fmt.Errorf("error reading defaults: %w", err)
				}
			}
			for k, v := range d.AllSettings() {
				defaults.SetDefault(k, v)
			}
		}
	} else if len(confDirs) > 0 {
		for _, dir := range confDirs {
			defaults.AddConfigPath(dir)
		}

		defaults.SetConfigName(configName + ".defaults")
		defaults.ReadInConfig()
		if err = defaults.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				// not found is fine
			} else {
				return c, fmt.Errorf("error reading defaults: %w", err)
			}
		}
	}

	// set defaults in real config based on collected defaults
	for k, v := range defaults.AllSettings() {
		c.Viper.SetDefault(k, v)
	}

	// fixed configuration file, skip directory search
	if opts.configFile != "" {
		c.Viper.SetConfigFile(opts.configFile)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				// not found is fine
			} else {
				return c, fmt.Errorf("error reading config: %w", err)
			}
		}
		return c, nil
	}

	// load configuration files from given directories, in order

	if opts.merge {
		for _, dir := range confDirs {
			d := viper.New()
			d.AddConfigPath(dir)
			d.SetConfigName(configName)
			if err = d.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
					// not found is fine
					continue
				} else {
					return c, fmt.Errorf("error reading config: %w", err)
				}
			}
			log.Debug().Msgf("merging %+v into %+v", d.AllSettings(), c.Viper.AllSettings())
			if err = c.Viper.MergeConfigMap(d.AllSettings()); err != nil {
				log.Debug().Err(err).Msgf("merge of %s/%s failed, continuing.", dir, configName)
			}
		}
		return c, nil
	}

	if len(confDirs) > 0 {
		for _, dir := range confDirs {
			c.Viper.AddConfigPath(dir)
		}
		c.Viper.SetConfigName(configName)
		if err = c.Viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, fs.ErrNotExist) {
				// not found is fine
			} else {
				return c, fmt.Errorf("error reading config: %w", err)
			}
		}
	}

	return c, nil
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

var discardRE = regexp.MustCompile(`(?m)^\s*#.*$`)
var shrinkBackSlashRE = regexp.MustCompile(`(?m)\\\\`)

// MergeHOCONConfig parses the HOCON configuration in conf and merges the
// results into the cf *config.Config object
func (cf *Config) MergeHOCONConfig(conf string) (err error) {
	conf = discardRE.ReplaceAllString(conf, "")
	hc, err := hocon.ParseString(conf)
	if err != nil {
		return
	}

	vc := viper.New()
	vc.SetConfigType("json")

	j, err := json.Marshal(hc.GetRoot())
	j = shrinkBackSlashRE.ReplaceAll(j, []byte{'\\'})
	if err != nil {
		return
	}
	cs := bytes.NewReader(j)
	if err := vc.ReadConfig(cs); err != nil {
		return err
	}

	cf.MergeConfigMap(vc.AllSettings())
	return
}

// MergeHOCONFile reads a HOCON configuration file in path and
// merges the settings into the cf *config.Config object
func (cf *Config) MergeHOCONFile(path string) (err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	return cf.MergeHOCONConfig(string(b))
}

func ReadHOCONFile(path string) (cf *Config, err error) {
	cf = New()
	err = cf.MergeHOCONFile(path)
	return
}

func ReadEncodePassword(keyfile string) (encpw string, err error) {
	var plaintext []byte
	var match bool
	for i := 0; i < 3; i++ {
		plaintext = ReadPasswordPrompt()
		plaintext2 := ReadPasswordPrompt("Re-enter Password")
		if bytes.Equal(plaintext, plaintext2) {
			match = true
			break
		}
		fmt.Println("Passwords do not match. Please try again.")
	}
	if !match {
		return "", fmt.Errorf("too many attempts, giving up")
	}
	return EncodePassword(plaintext, keyfile)
}

func ReadPasswordPrompt(prompt ...string) []byte {
	if len(prompt) == 0 {
		fmt.Printf("Password: ")
	} else {
		fmt.Printf("%s: ", strings.Join(prompt, " "))
	}
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting password")
	}
	fmt.Println()
	return bytes.TrimSpace(pw)
}

func ReadPasswordFile(path string) []byte {
	pw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading password from file")
	}
	return bytes.TrimSpace(pw)
}
