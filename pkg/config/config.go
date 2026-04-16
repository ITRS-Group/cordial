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

// Package config adds local extensions to viper as well as supporting Geneos
// encryption key files and basic encryption and decryption.
package config

import (
	"bytes"
	"errors"
	"os"
	"os/user"
	"path"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config structure
type Config struct {
	viper                *viper.Viper
	mutex                *sync.RWMutex // mutex to protect concurrent access to the above viper
	configType           string        // The type of configuration file loaded ("rc", "json", "yaml" etc) - this is not the same as the config type used for unmarshalling, which is determined by the file extension or SetConfigType() call and is stored in viper.ConfigType()
	defaultExpandOptions []ExpandOptions
	delimiter            string
	appUserConfDir       string
}

// global is the default configuration container for non-method callers
var global *Config

// globalMutex protects the global configuration object for concurrent access
var globalMutex sync.Mutex

func init() {
	global = New()
}

// Global returns the global Config instance
func Global() *Config {
	return global
}

// ResetConfig reinitialises the global configuration object. Existing
// settings will be copied over. This is primarily to be able to change
// the default delimiter after start-up.
func ResetConfig(options ...FileOptions) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	tmp := global.AllSettings()
	global = New(options...)
	global.MergeConfigMap(tmp)
}

// New returns a Config instance initialised with a new viper instance.
// Can be called with config.DefaultExpandOptions(...) to set defaults for
// future calls that use Expand.
func New(options ...FileOptions) *Config {
	var appUserConfDir string
	opts := evalFileOptions(options...)
	if userConfDir, err := UserConfigDir(); err == nil {
		// only set if no error, else ignore
		appUserConfDir = path.Join(userConfDir, opts.appName)
	}
	cf := &Config{
		viper: viper.NewWithOptions(
			viper.KeyDelimiter(opts.delimiter),
			viper.EnvKeyReplacer(strings.NewReplacer(opts.delimiter, opts.envDelimiter, "-", opts.envDelimiter))),
		mutex:                &sync.RWMutex{},
		delimiter:            opts.delimiter,
		appUserConfDir:       appUserConfDir,
		defaultExpandOptions: opts.expandOptions,
	}
	if opts.envPrefix != "" {
		cf.SetEnvPrefix(opts.envPrefix)
		cf.automaticEnv()
	}

	if len(opts.internalDefaults) > 0 {
		buf := bytes.NewBuffer(opts.internalDefaults)
		internalDefaults := &Config{
			viper: viper.New(),
			mutex: &sync.RWMutex{},
		}
		internalDefaults.setConfigType(opts.internalDefaultsFormat)
		if err := internalDefaults.readConfig(buf); err == nil || !opts.internalDefaultsCheckErrors {
			cf.MergeConfigMap(internalDefaults.AllSettings())
		}
	}

	if opts.defaultConfig != nil {
		for k, v := range opts.defaultConfig.AllSettings() {
			cf.SetDefault(k, v)
		}
	}
	return cf
}

var ErrNoUserConfigDir = errors.New("cannot resolve user config directory, check $USER and $HOME exist")

// AppConfigDir returns the application configuration directory for the
// global configuration
func AppConfigDir() string {
	return global.appUserConfDir
}

// AppConfigDir returns the application configuration directory for c
func (c *Config) AppConfigDir() string {
	return c.appUserConfDir
}

// Join returns a configuration key made up of parts joined with the
// default delimiter for the global configuration object.
func Join(parts ...string) string {
	return strings.Join(append([]string{}, parts...), global.delimiter)
}

// Join returns a configuration settings key joined with the delimiter
// for the c config object.
func (c *Config) Join(parts ...string) string {
	return strings.Join(append([]string{}, parts...), c.delimiter)
}

// Delimiter returns the global config key delimiter
func Delimiter() string {
	return global.delimiter
}

// Delimiter returns the config c key delimiter
func (c *Config) Delimiter() string {
	return c.delimiter
}

func (c *Config) ConfigType() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.configType
}

func (c *Config) SetConfigType(t string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.configType = t
}

// Sub returns a Config instance rooted at the key passed. If key does
// not exist then an empty config structure is returned, unlike viper
// which returns nil.
//
// Note that viper.Sub() does NOT merge defaults
func (c *Config) Sub(key string) *Config {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.sub(key)
}

// Sub returns a Config instance rooted at the key passed. If key does
// not exist then an empty config structure is returned, unlike viper
// which returns nil.
//
// Note that viper.Sub() does NOT merge defaults
func Sub(key string) *Config {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.sub(key)
}

func Set[T any](c *Config, key string, value T, options ...ExpandOptions) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	set(c, key, value, options...)
}

// Set sets the key to value in the config structure c
func (c *Config) Set(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	set(c, key, value)
}

//
// Get functions
//

// Get returns the value associated with the key in the configuration
// structure c, applying the options given. The type T is the expected
// type of the value, which can be one of:
//
//   - bool
//   - int
//   - int64
//   - uint
//   - uint16
//   - float64
//   - string
//   - []string
//   - []byte
//   - map[string]any
//   - map[string]string
//   - []map[string]string
//   - time.Duration
//   - *config.Plaintext
//
// Other types are returned as-is and the caller is expected to do any
// necessary type assertion. Other specific types may be added in the
// future.
//
// If the option [`config.Default`] is used, then the type must be
// identical to T. If it is not, then the default value is the zero
// value for the type T.
func Get[T any](c *Config, key string, options ...ExpandOptions) (value T) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return get[T](c, key, options...)
}

// GetString functions like [viper.GetString] but additionally calls
// [ExpandString] with the configuration value, passing any "values" maps
func GetString(s string, options ...ExpandOptions) string {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return get[string](global, s, options...)
}

// GetString functions like [viper.GetString] on a Config instance, but
// additionally calls [ExpandString] with the configuration value, passing
// any "values" maps
func (c *Config) GetString(s string, options ...ExpandOptions) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return get[string](c, s, options...)
}

// ExpandFieldsHook returns a mapstructure.DecodeHookFunc that expands
// string fields using the config.ExpandString function with the options
// given. This is intended to be used in Unmarshal calls to allow for
// dynamic values in the configuration file. The hook will only expand
// string fields, and will leave other types unchanged.
var ExpandFieldsHook = func(opts ...ExpandOptions) mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		str := data.(string)

		return ExpandString(str, opts...), nil
	}
}

func (c *Config) UnmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.unmarshalKey(key, rawVal, opts...)
}

var itemRE = regexp.MustCompile(`^([\w\.\:-]+)([+=]=?)(.*)`)

// SetKeyValues takes a list of `key=value` pairs as strings and applies
// them to the config object. Any item without an `=` is skipped.
//
// If the separator is either `+=` or `+` then the given value is
// appended to any existing setting. If the value is starts with a dash
// then it is considered a command line option and is appended with a
// space separator, otherwise it is simply concatenated.
func (c *Config) SetKeyValues(items ...string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.setKeyValues(items...)
}

func (c *Config) MergeConfigMap(vals map[string]any) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.mergeConfigMap(vals)
}

func MergeConfigMap(vals map[string]any) error {
	global.mutex.Lock()
	defer global.mutex.Unlock()
	return global.mergeConfigMap(vals)
}

func (c *Config) AllKeys() (keys []string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.allKeys()
}

func AllKeys() []string {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.allKeys()
}

func (c *Config) AllSettings() (value map[string]any) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.allSettings()
}

func AllSettings() map[string]any {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.allSettings()
}

func (c *Config) SetEnvPrefix(prefix string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.setEnvPrefix(prefix)
}

func (c *Config) AutomaticEnv() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.automaticEnv()
}

func (c *Config) SetDefault(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.setDefault(key, value)
}

func (c *Config) IsSet(key string) (value bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isSet(key)
}

func IsSet(key string) bool {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.isSet(key)
}

func (c *Config) ConfigFileUsed() (f string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.configFileUsed()
}

func ConfigFileUsed() string {
	global.mutex.RLock()
	defer global.mutex.RUnlock()
	return global.configFileUsed()
}

func (c *Config) RegisterAlias(alias, key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.registerAlias(alias, key)
}

func RegisterAlias(alias, key string) {
	global.mutex.Lock()
	defer global.mutex.Unlock()
	global.registerAlias(alias, key)
}

func (c *Config) BindEnv(input ...string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.bindEnv(input...)
}

func BindEnv(input ...string) error {
	global.mutex.Lock()
	defer global.mutex.Unlock()
	return global.bindEnv(input...)
}

// UserHomeDir returns the home directory for username, or if none given
// then the current user. This works around empty environments by
// falling back to looking up the user.
func UserHomeDir(username ...string) (home string, err error) {
	if len(username) == 0 {
		if home, err = os.UserHomeDir(); err == nil { // all ok
			return
		}
		u, err := user.Current()
		if err != nil {
			return home, err
		}
		return u.HomeDir, nil
	}
	u, err := user.Lookup(username[0])
	if err != nil {
		return
	}
	return u.HomeDir, nil
}
