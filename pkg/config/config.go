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
	"path"
	"reflect"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
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
			cf.Default(k, v)
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

// BindPFlag binds a pflag.Flag to a key in the configuration.
func (c *Config) BindPFlag(key string, flag *pflag.Flag) (err error) {
	return c.viper.BindPFlag(key, flag)
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

func Set[T any](c *Config, key string, value T, options ...ExpandOptions) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	set(c, key, value, options...)
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
//   - *config.Secret
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

func Lookup[T any](c *Config, key string, options ...ExpandOptions) (value T, found bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return lookup[T](c, key, options...)
}

func Delete(c *Config, key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	deleteKey(c, key)
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

		return expand[string](global, str, opts...), nil
	}
}

func (c *Config) UnmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.unmarshalKey(key, rawVal, opts...)
}

// SetKeyValuePairs takes a list of `key=value` pairs as strings and applies
// them to the config object. Any item without an `=` is skipped.
//
// If the separator is either `+=` or `+` then the given value is
// appended to any existing setting. If the value is starts with a dash
// then it is considered a command line option and is appended with a
// space separator, otherwise it is simply concatenated.
func (c *Config) SetKeyValuePairs(items ...string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.setKeyValuePairs(items...)
}

func (c *Config) MergeConfigMap(vals map[string]any) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.mergeConfigMap(vals)
}

// AllKeys returns a slice of all the keys in the configuration,
// excluding any that have been marked as deleted. The keys are returned
// in sorted order.
func (c *Config) AllKeys() (keys []string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.allKeys()
}

// AllSettings returns a map of all the settings in the configuration,
// excluding any that have been marked as deleted. Only the top-level
// keys are returned, i.e. nested keys are not included and are in the
// value from the map.
//
// To return all the keys and values, use AllKeys to get the keys and
// then Get to get the values for each key.
func (c *Config) AllSettings() (value map[string]any) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.allSettings()
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

func (c *Config) Default(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.setDefault(key, value)
}

func (c *Config) IsSet(key string) (value bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isSet(key)
}

func (c *Config) ConfigFileUsed() (f string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.configFileUsed()
}

func (c *Config) RegisterAlias(alias, key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.registerAlias(alias, key)
}

func (c *Config) BindEnv(input ...string) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.bindEnv(input...)
}
