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

// ExpandString returns it's input with any occurrences of the form
// ${name} or $name substituted using [os.Expand] for the formats
// in the order given below:
//
//      * ${path.to.config}
//      Any name containing one or more dots '.' will be looked up in the
//      running configuration (which can include existing settings outside
//      of any configuration file being read by the caller)
//      * ${name}
//      'name' will be substituted with the corresponding value from the map
//      'confmap'. If 'confmap' is empty (as opposed to a name not being found)
//      then name is looked up from the environment, as below
//      * ${env:name}
//      'name' with be substituted with the contents of the environment
//		variable of the same name.
//      * ${file://path/to/file} or ${file:~/path/to/file}
//      The contents of the referenced file will be read-in when the
//      configuration is loaded. Multiline files are used as-is so this
//		can, for example, be used to read PEM certificate files or keys.
//		As a change to a real file url, if the first '/' is replaced with
//		a tilde '~' then the path is relative to the home directory of
//		the user running the process.
//      * ${https://host/path} or ${http://host/path}
//      The contents of the URL are fetched and used similarly as for
//		local files above.
//
// The form $name is also supported, as per [os.Expand] but may be
// ambiguous and is not recommended.
//
// Expansion is not recursive. For each substitution any leading and
// trailing whitespace are removed.
//
// Any errors (particularly from substitutions from external files or
// remote URLs) may result in an empty or corrupt string being returned.
// Error returns are intentionally discarded.
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
