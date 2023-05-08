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
	"strconv"

	"github.com/spf13/viper"
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
