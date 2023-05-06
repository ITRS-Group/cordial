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
This package adds local extensions to viper as well as supporting Geneos
encryption key files and basic encryption and decryption.
*/
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/gurkankaymak/hocon"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// Config embeds Viper
type Config struct {
	*viper.Viper
	Type                 string // The type of configuration file loaded
	defaultExpandOptions []ExpandOptions
}

// global is the default configuration container for non-method callers
var global *Config

func init() {
	global = &Config{Viper: viper.New()}
}

// GetConfig returns the global Config instance
func GetConfig() *Config {
	return global
}

// New returns a Config instance initialised with a new viper instance
func New() *Config {
	return &Config{Viper: viper.New()}
}

// Sub returns a Config instance rooted at the key passed
func (c *Config) Sub(key string) *Config {
	return &Config{Viper: c.Viper.Sub(key)}
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

var discardRE = regexp.MustCompile(`(?m)^\s*#.*$`)
var shrinkBackSlashRE = regexp.MustCompile(`(?m)\\\\`)

// MergeHOCONConfig parses the HOCON configuration in conf and merges the
// results into the cf *config.Config object
func (c *Config) MergeHOCONConfig(conf string) (err error) {
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

	c.MergeConfigMap(vc.AllSettings())
	return
}

// MergeHOCONFile reads a HOCON configuration file in path and
// merges the settings into the cf *config.Config object
func (c *Config) MergeHOCONFile(path string) (err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	return c.MergeHOCONConfig(string(b))
}

// ReadHOCONFile loads a HOCON format configuration from file. To
// control behaviour and options like config.Load() use
// MergeHOCONConfig() or MergeHOCONFile() with an existing config.Config
// structure.
func ReadHOCONFile(file string) (c *Config, err error) {
	c = New()
	err = c.MergeHOCONFile(file)
	return
}

// ReadUserInput reads input from Stdin and returns the input unless
// there is an error. The prompt is shown to the user as-is.
func ReadUserInput(prompt string) (input string, err error) {
	var oldState *term.State
	// prompt for username
	if oldState, err = term.MakeRaw(int(os.Stdin.Fd())); err != nil {
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	t := term.NewTerminal(os.Stdin, prompt)
	return t.ReadLine()
}

// ReadPasswordInput prompts the user for a password without echoing the
// input. This is returned as a byte slice. If validate is true then the
// user is prompted twice and the two instances checked for a match. Up
// to maxtries attempts are allowed after which an error is returned. If
// maxtries is 0 then a default of 3 attempts is set.
//
// If prompt is given then it must either be one or two strings,
// depending on validate being false or true respectively. The prompt(s)
// are suffixed with ": " in both cases. The defaults are "Password" and
// "Re-enter Password".
func ReadPasswordInput(validate bool, maxtries int, prompt ...string) (pw []byte, err error) {
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

	return
}
