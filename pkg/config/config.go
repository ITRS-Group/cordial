package config

import (
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// The Config type acts as a container for local/extended viper instances
type Config struct {
	v *viper.Viper
}

var globalConfig *Config

func init() {
	globalConfig = &Config{v: viper.GetViper()}
}

// Returns the configuration item as a string with ExpandString() applied,
// passing the first "confmap" if given
func GetString(s string, confmap ...map[string]string) string {
	return globalConfig.GetString(s, confmap...)
}

// Returns the configuration item as a string with ExpandString() applied,
// passing the first "confmap" if given
func (c *Config) GetString(s string, confmap ...map[string]string) string {
	if len(confmap) > 0 {
		return c.ExpandString(c.v.GetString(s), confmap[0])
	}
	return c.ExpandString(c.v.GetString(s), nil)
}

func GetConfig() *Config {
	return globalConfig
}

func GetViper() *viper.Viper {
	return globalConfig.v
}

func (c *Config) GetViper() *viper.Viper {
	return c.v
}

func (c *Config) SetViper(v *viper.Viper) {
	c.v = v
}

func GetInt(s string) int {
	return globalConfig.GetInt(s)
}

func (c *Config) GetInt(s string) int {
	return c.v.GetInt(s)
}

func GetBool(s string) bool {
	return globalConfig.GetBool(s)
}

func (c *Config) GetBool(s string) bool {
	return c.v.GetBool(s)
}

func GetDuration(s string) time.Duration {
	return globalConfig.GetDuration(s)
}

func (c *Config) GetDuration(s string) time.Duration {
	return c.v.GetDuration(s)
}

func SetDefault(s string, i interface{}) {
	globalConfig.SetDefault(s, i)
}

func (c *Config) SetDefault(s string, i interface{}) {
	c.v.SetDefault(s, i)
}

func Set(s string, i interface{}) {
	globalConfig.Set(s, i)
}

func (c *Config) Set(s string, i interface{}) {
	c.v.Set(s, i)
}

func GetStringSlice(s string, confmap ...map[string]string) []string {
	return globalConfig.GetStringSlice(s, confmap...)
}

func (c *Config) GetStringSlice(s string, confmap ...map[string]string) (slice []string) {
	r := c.v.GetStringSlice(s)
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
	return globalConfig.GetStringMapString(s, confmap...)
}

func (c *Config) GetStringMapString(s string, confmap ...map[string]string) (m map[string]string) {
	var cfmap map[string]string
	m = make(map[string]string)
	r := c.v.GetStringMapString(s)
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
//		${env:VARNAME} - replace with the contents of the environment varaible VARNAME, trim whitespace
//		${path.to.config} - and var containing a '.' will be looked up in global viper config space - this is NOT recursive
//		${name} - replace with the contents of confmap["name"] - trim whitespace
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
		if !strings.Contains(s, ":") {
			if strings.Contains(s, ".") {
				// this call to GetString() must NOT be recursive
				return strings.TrimSpace(c.v.GetString(s))
			}
			if len(confmap) == 0 {
				return strings.TrimSpace(mapEnv(s))
			}
			return strings.TrimSpace(confmap[s])
		}
		if strings.HasPrefix(s, "env:") {
			return strings.TrimSpace(mapEnv(strings.TrimPrefix(s, "env:")))
		}
		if strings.HasPrefix(s, "file:") {
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
		}
		if strings.HasPrefix(s, "http:") || strings.HasPrefix(s, "https:") {
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
