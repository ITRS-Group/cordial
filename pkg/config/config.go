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
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Config embeds Viper
type Config struct {
	Viper                *viper.Viper
	mutex                *sync.RWMutex // mutex to protect concurrent access to the above viper
	Type                 string        // The type of configuration file loaded
	defaultExpandOptions []ExpandOptions
	delimiter            string
	appUserConfDir       string
}

// global is the default configuration container for non-method callers
var global *Config

func init() {
	global = New()
}

// GetConfig returns the global Config instance
func GetConfig() *Config {
	return global
}

// ResetConfig reinitialises the global configuration object. Existing
// settings will be copied over. This is primarily to be able to change
// the default delimiter after start-up.
func ResetConfig(options ...FileOptions) {
	var mt sync.Mutex

	mt.Lock()
	tmp := global.AllSettings()
	global = New(options...)
	global.MergeConfigMap(tmp)
	mt.Unlock()
}

// New returns a Config instance initialised with a new viper instance.
// Can be called with config.DefaultExpandOptions(...) to set defaults for
// future calls that use Expand.
func New(options ...FileOptions) *Config {
	var appUserConfDir string
	opts := evalFileOptions(options...)
	if userConfDir, err := UserConfigDir(); err == nil {
		// only set of no error, else ignore
		appUserConfDir = path.Join(userConfDir, opts.appname)
	}
	cf := &Config{
		Viper: viper.NewWithOptions(
			viper.KeyDelimiter(opts.delimiter),
			viper.EnvKeyReplacer(strings.NewReplacer(opts.delimiter, opts.envdelimiter, "-", opts.envdelimiter))),
		mutex:          &sync.RWMutex{},
		delimiter:      opts.delimiter,
		appUserConfDir: appUserConfDir,
	}
	if opts.envprefix != "" {
		cf.SetEnvPrefix(opts.envprefix)
		cf.AutomaticEnv()
	}
	return cf
}

var ErrNoUserConfigDir = errors.New("cannot resolve user config directory, check $USER and $HOME exist")

// AppConfigDir returns the application configuration directory
func AppConfigDir() string {
	return global.appUserConfDir
}

// AppConfigDir returns the application configuration directory
func (c *Config) AppConfigDir() string {
	return c.appUserConfDir
}

// Join returns a configuration key made up of parts joined with the
// default delimiter for the global configuration object.
func Join(parts ...string) string {
	return global.Join(parts...)
	// elems := []string{}
	// return strings.Join(append(elems, parts...), global.delimiter)
}

// Join returns a configuration settings key joined with the delimiter
// for the c config object.
func (c *Config) Join(parts ...string) string {
	elems := []string{}
	return strings.Join(append(elems, parts...), c.delimiter)
}

// Delimiter returns the global config key delimiter
func Delimiter() string {
	return global.Delimiter()
}

// Delimiter returns the config c key delimiter
func (c *Config) Delimiter() string {
	return c.delimiter
}

// Sub returns a Config instance rooted at the key passed. If key does
// not exist then an empty config structure is returned, unlike viper
// which returns nil. It uses the mutex pointer from the caller so that
// locking of sub-config objects also applies to the original.
//
// Note that viper.Sub() does NOT merge defaults
func (c *Config) Sub(key string) *Config {
	c.mutex.RLock()
	vcf := c.Viper.Sub(key)
	c.mutex.RUnlock()

	if vcf == nil {
		vcf = viper.New()
	}
	return &Config{
		Viper:                vcf,
		mutex:                c.mutex,
		Type:                 c.Type,
		delimiter:            c.delimiter,
		defaultExpandOptions: c.defaultExpandOptions,
		appUserConfDir:       c.appUserConfDir,
	}
}

// Sub returns a Config instance rooted at the key passed. If key does
// not exist then an empty config structure is returned, unlike viper
// which returns nil. It uses the mutex pointer from the caller so that
// locking of sub-config objects also applies to the original.
//
// Note that viper.Sub() does NOT merge defaults
func Sub(key string) *Config {
	return global.Sub(key)
}

// Set sets the key to value
func Set(key string, value interface{}) {
	global.Set(key, value)
}

// Set sets the key to value
func (c *Config) Set(key string, value interface{}) {
	c.mutex.Lock()
	c.Viper.Set(key, value)
	c.mutex.Unlock()
}

// SetString sets the key to the string given after processing options.
// Options include replacing substrings with configuration items that
// match *at the time of the SetString call*. This allows the
// abstraction of a static string based on the other config values
// given. E.g.
//
//	cf.SetString("setup", "/path/to/myname/setup.json", config.Replace("name"))
//
// This would check the value of the "name" key in cf and do a global
// replace. Multiple Replace options are processed in order. If "name"
// was "myname" at the time of the call then the resulting value is
// `/path/to/${config:name}/setup.json`
//
// Existing expand options are left unchanged. All replacements are case
// sensitive.
func (c *Config) SetString(key, value string, options ...ExpandOptions) {
	value = c.replaceString(value, options...)
	c.Set(key, value)
}

// SetString sets the key to the value with options applied. This
// applies to the global configuration struct. See the other SetString
// for detailed behaviour.
func SetString(key, value string, options ...ExpandOptions) {
	global.SetString(key, value, options...)
}

// replaceString does the string replacement for the Set* functions
func (c *Config) replaceString(value string, options ...ExpandOptions) string {
	opts := evalExpandOptions(c, options...)

	if len(opts.replacements) == 0 {
		return value
	}

	for _, r := range opts.replacements {
		sub := c.GetString(r)

		// simple case, no expand substrings
		if !strings.Contains(value, "${") {
			value = strings.ReplaceAll(value, sub, "${config:"+r+"}")
			continue
		}

		// iterate over value, skipping expand substrings
		var newval string
		for remval := value; ; {
			start := strings.Index(remval, "${")
			end := strings.Index(remval, "}")
			if start == -1 || end == -1 {
				// finished with expand options. any unterminated
				// substring is treated as ending the string, so just
				// return the concatenated string
				value = newval + remval
				break
			}
			// append substituted nonexpand substring
			newval += strings.ReplaceAll(remval[:start], sub, "${config:"+r+"}")
			// append expand substring
			newval += remval[start : end+1]
			// remove above from remaining
			remval = remval[end+1:]
		}
	}
	return value
}

// SetStringSlice sets the key to a slice of strings applying the
// replacement options as for SetString to each member of the slice
func (c *Config) SetStringSlice(key string, values []string, options ...ExpandOptions) {
	for i, v := range values {
		values[i] = c.replaceString(v, options...)
	}
	c.Set(key, values)
}

// SetStringSlice sets the key to a slice of strings applying the
// replacement options as for SetString to each member of the slice
func SetStringSlice(key string, values []string, options ...ExpandOptions) {
	global.SetStringSlice(key, values, options...)
}

// SetStringMapString iterates over a map[string]string and sets each
// key to the value given. Viper's Set() doesn't support maps until the
// configuration is written to and read back from a file.
func (c *Config) SetStringMapString(m string, vals map[string]string, options ...ExpandOptions) {
	for k, v := range vals {
		c.SetString(m+c.delimiter+k, v)
	}
}

// SetStringMapString iterates over a map[string]string and sets each key to the
// value given. Viper's Set() doesn't support maps until the
// configuration is written to and read back from a file.
func SetStringMapString(m string, vals map[string]string, options ...ExpandOptions) {
	global.SetStringMapString(m, vals, options...)
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
	c.mutex.RLock()
	str := c.Viper.GetString(s)
	c.mutex.RUnlock()

	return c.ExpandString(str, options...)
}

// GetPassword returns a sealed enclave containing the configuration item
// identified by key and expanded using the Expand function with the
// options supplied.
func GetPassword(s string, options ...ExpandOptions) *Plaintext {
	return global.GetPassword(s, options...)
}

// GetPassword returns a sealed enclave containing the configuration item
// identified by key and expanded using the Expand function with the
// options supplied.
func (c *Config) GetPassword(key string, options ...ExpandOptions) *Plaintext {
	c.mutex.RLock()
	str := c.Viper.GetString(key)
	c.mutex.RUnlock()

	return &Plaintext{c.ExpandToEnclave(str, options...)}
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
	c.mutex.RLock()
	str := c.Viper.GetString(s)
	c.mutex.RUnlock()

	value := c.ExpandString(str, options...)
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
	c.mutex.RLock()
	str := c.Viper.GetString(s)
	c.mutex.RUnlock()

	value := c.ExpandString(str, options...)
	i, _ = strconv.ParseInt(value, 10, 64)
	return
}

// GetBytes functions like [viper.GetString] but additionally calls
// [Expand] with the configuration value, passing any "values" maps and
// returning a byte slice
func GetBytes(s string, options ...ExpandOptions) []byte {
	return global.GetBytes(s, options...)
}

// GetBytes functions like [viper.GetString] on a Config instance, but
// additionally calls [Expand] with the configuration value, passing
// any "values" maps and returning a byte slice
func (c *Config) GetBytes(s string, options ...ExpandOptions) []byte {
	c.mutex.RLock()
	str := c.Viper.GetString(s)
	c.mutex.RUnlock()

	return c.Expand(str, options...)
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
	var result []string
	opts := evalExpandOptions(c, options...)

	c.mutex.RLock()
	is := c.Viper.IsSet(s)
	c.mutex.RUnlock()

	if is {
		c.mutex.RLock()
		result = c.Viper.GetStringSlice(s)
		c.mutex.RUnlock()
	} else if init, ok := opts.initialValue.([]string); ok {
		result = init
	}

	if len(result) == 0 {
		if def, ok := opts.defaultValue.([]string); ok {
			result = def
		}
	}

	for _, n := range result {
		slice = append(slice, c.ExpandString(n, options...))
	}
	return
}

func isStringMapInterface(val interface{}) bool {
	if val == nil {
		return false
	}
	vt := reflect.TypeOf(val)
	return vt.Kind() == reflect.Map &&
		vt.Key().Kind() == reflect.String &&
		vt.Elem().Kind() == reflect.Interface
}

// GetStringMapString functions like [viper.GetStringMapString] but additionally calls
// [ExpandString] on each value element of the map, passing any "values" maps
func GetStringMapString(s string, options ...ExpandOptions) map[string]string {
	return global.GetStringMapString(s, options...)
}

// GetStringMapString functions like [viper.GetStringMapString] on a
// Config instance but additionally calls [ExpandString] on each value
// element of the map, passing any "values" maps
//
// Use a version of https://github.com/spf13/viper/pull/1504 to fix viper bug #1106
func (c *Config) GetStringMapString(key string, options ...ExpandOptions) (m map[string]string) {
	m = make(map[string]string)

	key = strings.ToLower(key)
	prefix := key + c.delimiter

	c.mutex.RLock()
	i := c.Viper.Get(key)
	c.mutex.RUnlock()

	if !isStringMapInterface(i) {
		return
	}
	val := i.(map[string]interface{})
	keys := c.AllKeys()
	for _, k := range keys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		mk := strings.TrimPrefix(key, prefix)
		mk = strings.Split(mk, c.delimiter)[0]
		if _, exists := val[mk]; exists {
			continue
		}
		mv := c.Get(key + c.delimiter + mk)
		if mv == nil {
			continue
		}
		val[mk] = mv
	}

	for k, v := range val {
		m[k] = c.ExpandString(fmt.Sprint(v), options...)
	}

	return
}

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices
func defaultDecoderConfig(output interface{}, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input interface{}, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return global.UnmarshalKey(key, rawVal, opts...)
}

func (c *Config) UnmarshalKey(key string, rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	key = strings.ToLower(key)
	prefix := key + c.delimiter

	c.mutex.Lock()
	i := c.Viper.Get(key)
	c.mutex.Unlock()

	if isStringMapInterface(i) {
		val := i.(map[string]interface{})
		keys := c.AllKeys()
		for _, k := range keys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			mk := strings.TrimPrefix(k, prefix)
			mk = strings.Split(mk, c.delimiter)[0]
			if _, exists := val[mk]; exists {
				continue
			}
			mv := c.Get(key + c.delimiter + mk)
			if mv == nil {
				continue
			}
			val[mk] = mv
		}
		i = val
	}

	return decode(i, defaultDecoderConfig(rawVal, opts...))
}

// GetSliceStringMapString returns a slice of string maps for the key s,
// it iterates over all values in all maps and applies the ExpandString
// with the options given
func (c *Config) GetSliceStringMapString(s string, options ...ExpandOptions) (result []map[string]string) {
	if err := c.UnmarshalKey(s, &result); err != nil {
		return
	}
	for _, m := range result {
		for k, v := range m {
			m[k] = c.ExpandString(v, options...)
		}
	}
	return
}

// GetSliceStringMapString returns a slice of string maps for the key s,
// it iterates over all values in all maps and applies the ExpandString
// with the options given
func GetSliceStringMapString(s string, options ...ExpandOptions) (result []map[string]string) {
	return global.GetSliceStringMapString(s, options...)
}

var itemRE = regexp.MustCompile(`^(\w+)([+=]=?)(.*)`)

// SetKeyValues takes a list of `key=value` pairs as strings and applies
// them to the config object. Any item without an `=` is skipped.
//
// If the separator is either `+=` or `+` then the given value is
// appended to any existing setting with a space.
func (c *Config) SetKeyValues(items ...string) {
	for _, item := range items {
		fields := itemRE.FindStringSubmatch(item)
		if len(fields) != 4 {
			continue
		}
		switch fields[2] {
		case "=":
			c.Set(fields[1], fields[3])
		case "+=", "+":
			if c.IsSet(fields[1]) {
				c.Set(fields[1], c.GetString(fields[1])+" "+fields[3])
			} else {
				c.Set(fields[1], fields[3])
			}
		default:
			continue
		}
	}
}

// SetKeyValues takes a list of `key-value` pairs as strings and
// applies them to the global configuration object. Items without an `=`
// are skipped.
func SetKeyValues(items ...string) {
	global.SetKeyValues(items...)
}

// The methods below are to put locks around viper methods we use. Needs
// to be completed for any methods not covered in normal use

func (c *Config) Get(key string) (value any) {
	c.mutex.RLock()
	value = c.Viper.Get(key)
	c.mutex.RUnlock()
	return
}

func Get(key string) any {
	return global.Get(key)
}

func (c *Config) MergeConfigMap(vals map[string]any) (err error) {
	c.mutex.Lock()
	err = c.Viper.MergeConfigMap(vals)
	c.mutex.Unlock()
	return
}

func MergeConfigMap(vals map[string]any) error {
	return global.MergeConfigMap(vals)
}

func (c *Config) AllKeys() (keys []string) {
	c.mutex.RLock()
	keys = c.Viper.AllKeys()
	c.mutex.RUnlock()
	return
}

func AllKeys() []string {
	return global.AllKeys()
}

func (c *Config) AllSettings() (value map[string]any) {
	c.mutex.RLock()
	value = c.Viper.AllSettings()
	c.mutex.RUnlock()
	return
}

func AllSettings() map[string]any {
	return global.AllSettings()
}

func (c *Config) SetEnvPrefix(prefix string) {
	c.mutex.Lock()
	c.Viper.SetEnvPrefix(prefix)
	c.mutex.Unlock()
}

func SetEnvPrefix(prefix string) {
	global.SetEnvPrefix(prefix)
}

func (c *Config) GetStringMap(key string) (value map[string]any) {
	c.mutex.RLock()
	value = c.Viper.GetStringMap(key)
	c.mutex.RUnlock()
	return
}

func GetStringMap(key string) any {
	return global.GetStringMap(key)
}

func (c *Config) AutomaticEnv() {
	c.mutex.Lock()
	c.Viper.AutomaticEnv()
	c.mutex.Unlock()
}

func AutomaticEnv() {
	global.AutomaticEnv()
}

func (c *Config) SetDefault(key string, value any) {
	c.mutex.Lock()
	c.Viper.SetDefault(key, value)
	c.mutex.Unlock()
}

func SetDefault(key string, value any) {
	global.SetDefault(key, value)
}

func (c *Config) GetBool(key string) (value bool) {
	c.mutex.RLock()
	value = c.Viper.GetBool(key)
	c.mutex.RUnlock()
	return
}

func GetBool(key string) bool {
	return global.GetBool(key)
}

func (c *Config) GetUint16(key string) (value uint16) {
	c.mutex.RLock()
	value = c.Viper.GetUint16(key)
	c.mutex.RUnlock()
	return
}

func GetUint16(key string) uint16 {
	return global.GetUint16(key)
}

func (c *Config) GetUint(key string) (value uint) {
	c.mutex.RLock()
	value = c.Viper.GetUint(key)
	c.mutex.RUnlock()
	return
}

func GetUint(key string) uint {
	return global.GetUint(key)
}

func (c *Config) GetFloat64(key string) (value float64) {
	c.mutex.RLock()
	value = c.Viper.GetFloat64(key)
	c.mutex.RUnlock()
	return
}

func GetFloat64(key string) float64 {
	return global.GetFloat64(key)
}

func (c *Config) GetDuration(key string) (value time.Duration) {
	c.mutex.RLock()
	value = c.Viper.GetDuration(key)
	c.mutex.RUnlock()
	return
}

func GetDuration(key string) time.Duration {
	return global.GetDuration(key)
}

func (c *Config) IsSet(key string) (value bool) {
	c.mutex.RLock()
	value = c.Viper.IsSet(key)
	c.mutex.RUnlock()
	return
}

func IsSet(key string) bool {
	return global.IsSet(key)
}

func (c *Config) ConfigFileUsed() (f string) {
	c.mutex.RLock()
	f = c.Viper.ConfigFileUsed()
	c.mutex.RUnlock()
	return
}

func ConfigFileUsed() string {
	return global.ConfigFileUsed()
}

func (c *Config) RegisterAlias(alias, key string) {
	c.mutex.Lock()
	c.Viper.RegisterAlias(alias, key)
	c.mutex.Unlock()
}

func RegisterAlias(alias, key string) {
	global.RegisterAlias(alias, key)
}

func (c *Config) BindEnv(input ...string) (err error) {
	c.mutex.Lock()
	err = c.Viper.BindEnv(input...)
	c.mutex.Unlock()
	return
}

func BindEnv(input ...string) error {
	return global.BindEnv(input...)
}

func (c *Config) GetStringMapStringSlice(key string) (values map[string][]string) {
	c.mutex.RLock()
	values = c.Viper.GetStringMapStringSlice(key)
	c.mutex.RUnlock()
	return
}

func GetStringMapStringSlice(key string) map[string][]string {
	return global.GetStringMapStringSlice(key)
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
