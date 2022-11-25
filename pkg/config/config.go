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
	"io/fs"
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

	if opts.setglobals {
		c = global
	} else {
		c = New()
	}

	defaults := viper.New()
	internalDefaults := viper.New()

	if opts.usedefaults && len(opts.defaults) > 0 {
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

	if opts.usedefaults {
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
