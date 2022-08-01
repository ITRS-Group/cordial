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
// passing the first "confmap" if given
func GetString(s string, confmap ...map[string]string) string {
	return global.GetString(s, confmap...)
}

// Returns the configuration item as a string with ExpandString() applied,
// passing the first "confmap" if given
func (c *Config) GetString(s string, confmap ...map[string]string) string {
	if len(confmap) > 0 {
		return c.ExpandString(c.Viper.GetString(s), confmap[0])
	}
	return c.ExpandString(c.Viper.GetString(s), nil)
}

func GetConfig() *Config {
	return global
}

func New() *Config {
	return &Config{Viper: viper.New()}
}

func Sub(key string) *Config {
	return &Config{Viper: viper.Sub(key)}
}

func (c *Config) Sub(key string) *Config {
	return &Config{Viper: c.Viper.Sub(key)}
}

func GetStringSlice(s string, confmap ...map[string]string) []string {
	return global.GetStringSlice(s, confmap...)
}

func (c *Config) GetStringSlice(s string, confmap ...map[string]string) (slice []string) {
	r := c.Viper.GetStringSlice(s)
	for _, n := range r {
		if len(confmap) > 0 {
			slice = append(slice, c.ExpandString(n, confmap[0]))
		} else {
			slice = append(slice, c.ExpandString(n, nil))
		}
	}
	return
}

func GetStringMapString(s string, confmap ...map[string]string) map[string]string {
	return global.GetStringMapString(s, confmap...)
}

func (c *Config) GetStringMapString(s string, confmap ...map[string]string) (m map[string]string) {
	var cfmap map[string]string
	m = make(map[string]string)
	r := c.Viper.GetStringMapString(s)
	if len(confmap) > 0 {
		cfmap = confmap[0]
	}
	for k, v := range r {
		m[k] = c.ExpandString(v, cfmap)
	}
	return m
}

// Return a string that has all contents of the form ${var} or $var
// expanded according to the following rules:
//
//		url - read the contents of the url, which can be a local file, as below:
//			${file://path/to/file} - read the entire contents of the file, trim whitespace
//			${https://host/path} - fetch the remote contents, trim whitespace. "http:" also supported.
//		${env:VARNAME} - replace with the contents of the environment variable VARNAME, trim whitespace
//		${path.to.config} - and var containing a '.' will be looked up in global viper config space - this is NOT recursive
//		${name} - replace with the contents of confmap["name"] - trim whitespace - if confmap is empty then try the environment
//
// While the form $var is also supported but may be ambiguous and is not
// recommended.
//
// If confmap is not given then environment variables are used directly.
//
// Any errors result in an empty string being returned.
//
func (c *Config) ExpandString(input string, confmap map[string]string) (value string) {
	value = os.Expand(input, func(s string) (r string) {
		switch {
		case !strings.Contains(s, ":"):
			if strings.Contains(s, ".") {
				// this call to GetString() must NOT be recursive
				return strings.TrimSpace(c.Viper.GetString(s))
			}
			if len(confmap) == 0 {
				return strings.TrimSpace(mapEnv(s))
			}
			return strings.TrimSpace(confmap[s])
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
	})
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
//		LoadConfig("geneos")
//
//		//go:embed somefile.json
//		var myDefaults []byte
//		LoadConfig("geneos", config.SetDefaults(myDefaults, "json"), )
//
// Options can be passed to change the default behaviour and to pass any
// embedded defaults or an existing viper.
//
// for defaults see:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
// ... find windows equiv
func LoadConfig(configName string, options ...Options) (c *Config) {
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
		defaultSettings := defaults.AllSettings()

		for k, v := range defaultSettings {
			c.Viper.SetDefault(k, v)
		}
	}

	if opts.configFile != "" {
		c.Viper.SetConfigFile(opts.configFile)
		c.Viper.ReadInConfig()
	} else if len(confDirs) > 0 {
		for _, d := range confDirs {
			c.Viper.AddConfigPath(d)
		}
		c.Viper.SetConfigName(configName)
		c.Viper.ReadInConfig()
	}

	return
}
