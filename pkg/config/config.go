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
	"strconv"
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
	Type                 string
	defaultExpandOptions []ExpandOptions
}

var global *Config

func init() {
	global = &Config{Viper: viper.New()}
}

// SetMap iterates over a map[string]string and sets each key to the
// value given. Viper's Set() doesn't support maps until the
// configuration is written to and read back from a file.
func (c *Config) SetStringMapString(m string, vals map[string]string) {
	for k, v := range vals {
		c.Set(m+"."+k, v)
	}
}

// SetMap iterates over a map[string]string and sets each key to the
// value given. Viper's Set() doesn't support maps until the
// configuration is written to and read back from a file.
func SetStringMapString(m string, vals map[string]string) {
	global.SetStringMapString(m, vals)
}

// XXX maybe later
// func (c *Config) Set(name string, value interface{}) {
// 	switch v := value.(type) {
// 	case string, int:
// 		c.Viper.Set(name, v)
// 	default:

// 	}
// }

// GetString functions like [viper.GetString] but additionally calls
// [ExpandString] with the configuration value, passing any "values" maps
func GetString(s string, options ...ExpandOptions) string {
	return global.GetString(s, options...)
}

// GetString functions like [viper.GetString] on a Config instance, but
// additionally calls [ExpandString] with the configuration value, passing
// any "values" maps
func (c *Config) GetString(s string, options ...ExpandOptions) string {
	return c.ExpandString(c.Viper.GetString(s), options...)
}

// GetInt functions like [viper.GetInt] but additionally calls
// [ExpandString] with the configuration value, passing any "values"
// maps. If the conversion fails then the value returned will be the one
// from [strconv.ParseInt] - typically 0 but can be the maximum integer
// value
func GetInt(s string, options ...ExpandOptions) int {
	return global.GetInt(s, options...)
}

// GetInt functions like [viper.GetInt] on a Config instance, but
// additionally calls [ExpandString] with the configuration value,
// passing any "values" maps, before converting the result to an int. If
// the conversion fails then the value returned will be the one from
// [strconv.ParseInt] - typically 0 but can be the maximum integer value
func (c *Config) GetInt(s string, options ...ExpandOptions) (i int) {
	value := c.ExpandString(c.Viper.GetString(s), options...)
	i, _ = strconv.Atoi(value)
	return
}

// GetInt64 functions like [viper.GetInt] but additionally calls
// [ExpandString] with the configuration value, passing any "values"
// maps. If the conversion fails then the value returned will be the one
// from [strconv.ParseInt] - typically 0 but can be the maximum integer
// value
func GetInt64(s string, options ...ExpandOptions) int64 {
	return global.GetInt64(s, options...)
}

// GetInt64 functions like [viper.GetInt] on a Config instance, but
// additionally calls [ExpandString] with the configuration value,
// passing any "values" maps, before converting the result to an int. If
// the conversion fails then the value returned will be the one from
// [strconv.ParseInt] - typically 0 but can be the maximum integer value
func (c *Config) GetInt64(s string, options ...ExpandOptions) (i int64) {
	value := c.ExpandString(c.Viper.GetString(s), options...)
	i, _ = strconv.ParseInt(value, 10, 64)
	return
}

// GetByteSlice functions like [viper.GetString] but additionally calls
// [Expand] with the configuration value, passing any "values" maps and
// returning a byte slice
func GetByteSlice(s string, options ...ExpandOptions) []byte {
	return global.GetByteSlice(s, options...)
}

// GetByteSlice functions like [viper.GetString] on a Config instance, but
// additionally calls [Expand] with the configuration value, passing
// any "values" maps and returning a byte slice
func (c *Config) GetByteSlice(s string, options ...ExpandOptions) []byte {
	return c.Expand(c.Viper.GetString(s), options...)
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
func GetStringSlice(s string, options ...ExpandOptions) []string {
	return global.GetStringSlice(s, options...)
}

// GetStringSlice functions like [viper.GetStringSlice] on a Config
// instance but additionally calls [ExpandString] on each element of the
// slice, passing any "values" maps
func (c *Config) GetStringSlice(s string, options ...ExpandOptions) (slice []string) {
	r := c.Viper.GetStringSlice(s)
	for _, n := range r {
		slice = append(slice, c.ExpandString(n, options...))
	}
	return
}

// GetStringMapString functions like [viper.GetStringMapString] but additionally calls
// [ExpandString] on each value element of the map, passing any "values" maps
func GetStringMapString(s string, options ...ExpandOptions) map[string]string {
	return global.GetStringMapString(s, options...)
}

// GetStringMapString functions like [viper.GetStringMapString] on a
// Config instance but additionally calls [ExpandString] on each value
// element of the map, passing any "values" maps
func (c *Config) GetStringMapString(s string, options ...ExpandOptions) (m map[string]string) {
	m = make(map[string]string)
	r := c.Viper.GetStringMapString(s)
	for k, v := range r {
		m[k] = c.ExpandString(v, options...)
	}
	return m
}

// LoadConfig loads configuration files from internal defaults, external
// defaults and the given configuration file(s). The configuration
// file(s) can be passed as an option. Each layer is only loaded once,
// if given. Internal defaults are passed as a byte slice - this is
// typically loaded from an embedded file but can be supplied from any
// source. External defaults, which have a `.defaults` suffix before the
// file extension, and the main configuration file are passed as ordered
// slices of strings. The first match is loaded unless the
// MergeSettings() option is passed, in which case all defaults are
// merged and then all non-defaults are merged in the order they were
// given.
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
//
// TBD: windows equiv of above
func LoadConfig(configName string, options ...Options) (c *Config, err error) {
	opts := evalOptions(configName, options...)

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

		// now set any internal default values as real defaults, cannot use Merge here
		for k, v := range internalDefaults.AllSettings() {
			defaults.SetDefault(k, v)
		}
	}

	// concatenate config directories in order - first match wins below,
	// unless MergeSettings() option is used. The order is:
	//
	// 1. Explicit directory arguments passed using the option AddConfigDirs()
	// 2. The working directory unless the option IgnoreWorkingDir() is used
	// 3. The user configuration directory plus `AppName`, unless IgnoreUserConfDir() is used
	// 4. The system configuration directory plus `AppName`, unless IgnoreSystemDir() is used
	confDirs := opts.configDirs
	if opts.workingdir != "" {
		confDirs = append(confDirs, opts.workingdir)
	}
	if opts.userconfdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.userconfdir, opts.appname))
	}
	if opts.systemdir != "" {
		confDirs = append(confDirs, filepath.Join(opts.systemdir, opts.appname))
	}

	// if we are merging, then we load in reverse order to ensure lower
	// priorities are overwritten
	if opts.merge {
		for i := len(confDirs)/2 - 1; i >= 0; i-- {
			opp := len(confDirs) - 1 - i
			confDirs[i], confDirs[opp] = confDirs[opp], confDirs[i]
		}
	}
	log.Debug().Msgf("confDirs: %v", confDirs)

	// search directories for defaults unless UseDefault(false) is
	// used as an option to LoadConfig(). we do this even if the
	// config file itself is set using option SetConfigFile()
	if opts.usedefaults {
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

		// set defaults in real config based on collected defaults,
		// following viper behaviour if the same default is set multiple
		// times.
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

// PasswordPrompt prompts the user for a password without echoing the input. This
// is returned as a whitespace trimmed byte slice. If validate is true then the
// user is prompted twice and the two instances checked for a match. Up to
// maxtries attempts are allowed after which an error is returned.
//
// If prompt is given then it must either be one or two strings, depending on
// validate being false or true respectively. The prompt(s) are suffixed with ":
// " in both cases. The defaults are "Password" and "Re-enter Password".
//
// if maxtries is 0 then it is set to the default of 3
func PasswordPrompt(validate bool, maxtries int, prompt ...string) (pw []byte, err error) {
	if validate {
		var match bool
		if len(prompt) != 2 {
			prompt = []string{}
		}

		if maxtries == 0 {
			maxtries = 3
		}

		for i := 0; i < maxtries; i++ {
			if len(prompt) == 0 {
				fmt.Printf("Password: ")
			} else {
				fmt.Printf("%s: ", prompt[0])
			}
			pw1, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // always move to new line even on error
			if err != nil {
				return pw, err
			}
			if len(prompt) == 0 {
				fmt.Printf("Re-enter Password: ")
			} else {
				fmt.Printf("%s: ", prompt[0])
			}
			pw2, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // always move to new line even on error
			if err != nil {
				return pw, err
			}
			if bytes.Equal(pw1, pw2) {
				pw = pw1
				match = true
				break
			}
			fmt.Println("Passwords do not match. Please try again.")
		}
		if !match {
			err = fmt.Errorf("too many attempts, giving up")
			return
		}
	} else {
		if len(prompt) == 0 {
			fmt.Printf("Password: ")
		} else {
			fmt.Printf("%s: ", strings.Join(prompt, " "))
		}
		pw, err = term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // always move to new line even on error
		if err != nil {
			return
		}
	}

	pw = bytes.TrimSpace(pw)
	return
}

func ReadPasswordFile(path string) []byte {
	pw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading password from file")
	}
	return bytes.TrimSpace(pw)
}
