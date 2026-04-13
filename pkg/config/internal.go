package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awnumar/memguard"
	"github.com/go-viper/mapstructure/v2"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/maja42/goval"
	"github.com/spf13/viper"
)

// internal routines that don't lock the config structure and also, for
// now, call the Viper methods directly. These are used by the public
// methods that do the locking and also by other internal methods that
// need to call Viper methods without locking, such as Sub() and Save().

func (c *Config) allKeys() (keys []string) {
	return c.Viper.AllKeys()
}

func (c *Config) allSettings() (value map[string]any) {
	return c.Viper.AllSettings()
}

// expand behaves like the expandString method but returns a byte
// slice.
func (c *Config) expand(input string, options ...ExpandOptions) (value []byte) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return bytes.Clone([]byte(input))
		}
		if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return bytes.Clone(b)
			}
			return fmt.Append(value, opts.initialValue)
		}
		return fmt.Append(value, opts.defaultValue)
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandBytes([]byte(input), func(s []byte) (r []byte) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				// return string and restore containing ${...}
				return fmt.Append([]byte{}, `${`, s, `}`)
			}
			return c.expandEncodedBytes(s[4:], options...)
		}
		str, _ := c.expandRawString(string(s), options...)
		return []byte(str)
	})

	if opts.trimSpace {
		value = bytes.TrimSpace(value)
	}

	if len(value) == 0 {
		value = fmt.Append(nil, opts.defaultValue)
	}

	return
}

// ExpandAllSettings returns all the settings from config structure c
// applying ExpandString to all string values and all string slice
// values. Non-string types are left unchanged. Further types, e.g. maps
// of strings, may be added in future releases.
func (c *Config) expandAllSettings(options ...ExpandOptions) (all map[string]any) {
	as := c.allSettings()
	all = make(map[string]any, len(as))

	for k, v := range as {
		switch ev := v.(type) {
		case string:
			all[k] = c.expandString(ev, options...)
		case []string:
			ns := []string{}
			for _, s := range ev {
				ns = append(ns, c.expandString(s, options...))
			}
			all[k] = ns
		case map[string]any:
			nm := make(map[string]any, len(ev))
			for kk, vv := range ev {
				if vvs, ok := vv.(string); ok {
					nm[kk] = c.expandString(vvs, options...)
				} else {
					nm[kk] = vv
				}
			}
			all[k] = nm
		default:
			all[k] = ev
		}
	}
	return
}

// expandEncodedString accepts input of the form:
//
//	[enc:]keyfile[,keyfile...]:[+encs+HEX|external]
//
// Each keyfile is tried until the first that does not return a decoding
// error. `keyfile` may be prefixed `~/` in which case the file is
// relative to the user's home directory. If the encoded string is
// prefixed with `+encs+` (standard Geneos usage) then it is used
// directly, otherwise the value is looked-up using the normal
// conventions for external access, e.g. file or URL.
func (c *Config) expandEncodedString(s string, options ...ExpandOptions) (value string) {
	opts := evalExpandOptions(c, options...)
	keyfiles, encodedValue := splitEncFields(s)
	if opts.usekeyfile != "" {
		keyfiles = opts.usekeyfile
	}

	if !strings.HasPrefix(encodedValue, "+encs+") {
		encodedValue, _ = c.expandRawString(encodedValue, options...)
	}
	if encodedValue == "" {
		return
	}

	for k := range strings.SplitSeq(keyfiles, "|") {
		keyfile := KeyFile(ExpandHome(k))
		p, err := keyfile.DecodeString(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return ""
}

// expandEncodedBytes is the byte slice version of expandEncodedString.
// It accepts the same input but returns a byte slice instead of a
// string. The input is expected to be UTF-8 encoded and the output is
// also UTF-8 encoded.
func (c *Config) expandEncodedBytes(s []byte, options ...ExpandOptions) (value []byte) {
	opts := evalExpandOptions(c, options...)
	keyfiles, encodedValue := splitEncFieldsBytes(s)
	if opts.usekeyfile != "" {
		keyfiles = []byte(opts.usekeyfile)
	}

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.expandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for k := range bytes.SplitSeq(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.Decode(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return
}

// expandEncodedBytesEnclave is the enclave version of
// expandEncodedBytes. It accepts the same input but returns a
// memguard.Enclave instead of a byte slice. The input is expected to be
// UTF-8 encoded and the output is also UTF-8 encoded.
func (c *Config) expandEncodedBytesEnclave(s []byte, options ...ExpandOptions) (value *memguard.Enclave) {
	opts := evalExpandOptions(c, options...)
	keyfiles, encodedValue := splitEncFieldsBytes(s)
	if opts.usekeyfile != "" {
		keyfiles = []byte(opts.usekeyfile)
	}

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.expandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for k := range bytes.SplitSeq(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.DecodeEnclave(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return
}

// expandEncodedBytesLockedBuffer is the locked buffer version of
// expandEncodedBytesEnclave. It accepts the same input but returns a
// memguard.LockedBuffer instead of a memguard.Enclave. The input is
// expected to be UTF-8 encoded and the output is also UTF-8 encoded.
func (c *Config) expandEncodedBytesLockedBuffer(s []byte, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	opts := evalExpandOptions(c, options...)
	keyfiles, encodedValue := splitEncFieldsBytes(s)
	if opts.usekeyfile != "" {
		keyfiles = []byte(opts.usekeyfile)
	}

	if !bytes.HasPrefix(encodedValue, []byte("+encs+")) {
		str, _ := c.expandRawString(string(encodedValue), options...)
		encodedValue = []byte(str)
	}
	if len(encodedValue) == 0 {
		return
	}

	for k := range bytes.SplitSeq(keyfiles, []byte("|")) {
		keyfile := KeyFile(ExpandHomeBytes(k))
		p, err := keyfile.DecodeEnclave(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		value, _ = p.Open()
		return
	}
	return
}

// expandRawString is the internal function that expands the string s
// using the same rules and options as [ExpandString] but treats the
// whole of s as if it were wrapped in '${...}'. The function does most
// of the core work for configuration expansion but is also exported for
// use without the decoration required for configuration values,
// allowing use against command line flag values, for example.
//
// With the ExpandNonStringToCSV() or ExpandNonStringToJSON() options,
// if the item to be expanded is a configuration value (either `config:`
// prefixed or without a prefix but containing a `.`) and the value
// resolved is not a plain string then it is either, for a slice of
// values, returned as a string containing comma-separated strings found
// or as a JSON encoded representation of the value.
func (c *Config) expandRawString(s string, options ...ExpandOptions) (value string, err error) {
	opts := evalExpandOptions(c, options...)
	switch {
	case strings.Contains(s, "/") && !strings.Contains(s, ":"):
		// check if defaults disabled
		if _, ok := opts.funcMaps["file"]; ok {
			if _, err = os.Stat(s); err != nil {
				return
			}
			ci := c.allSettings()
			return fetchFile(ci, s, opts.trimSpace)
		}
		return
	case strings.HasPrefix(s, "config:"), !strings.Contains(s, ":"):
		// TODO: SHould this dot be a delimiter lookup?
		if strings.HasPrefix(s, "config:") || strings.Contains(s, ".") {
			s = strings.TrimPrefix(s, "config:")
			if !opts.expandNonString {
				// this call to GetString() must NOT be recursive
				value = c.Viper.GetString(s)
				if opts.trimSpace {
					value = strings.TrimSpace(value)
				}
				return
			}

			v := c.Viper.Get(s)

			switch w := v.(type) {
			case string:
				// strings still get returned "as-is"
				value = w
				if opts.trimSpace {
					value = strings.TrimSpace(value)
				}
			case []any:
				if opts.expandNonStringCSV {
					var u []string
					for _, i := range w {
						s, ok := i.(string)
						if ok {
							u = append(u, s)
						}
					}
					value = strings.Join(u, ",")
				} else {
					// type switches do not support fallthrough, so
					// duplicate code here
					u, err := json.Marshal(w)
					if err != nil {
						return "", err
					}
					value = string(u)
				}
			default:
				// if caller has asked for CSV then do not return JSON
				if opts.expandNonStringCSV {
					value = ""
					return
				}
				u, err := json.Marshal(w)
				if err != nil {
					return "", err
				}
				value = string(u)
			}

			return
		}

		for _, v := range opts.lookupTables {
			if n, ok := v[s]; ok {
				value = n
				if opts.trimSpace {
					value = strings.TrimSpace(value)
				}
				return
			}
		}

		value = mapEnv(s)
		if opts.trimSpace {
			value = strings.TrimSpace(value)
		}

		return
	case strings.HasPrefix(s, "env:"):
		value = mapEnv(strings.TrimPrefix(s, "env:"))
		if opts.trimSpace {
			value = strings.TrimSpace(value)
		}
		return
	default:
		// check for any registered functions and call that with the
		// whole of the config string. there must be a ":" here, else
		// the above test would have picked it up. it is up to the
		// function called to trim whitespace, if required.
		f := strings.SplitN(s, ":", 2)
		ci := c.allSettings()
		if fn, ok := opts.funcMaps[f[0]]; ok {
			if opts.trimPrefix {
				value, err = fn(ci, f[1], opts.trimSpace)
			} else {
				value, err = fn(ci, s, opts.trimSpace)
			}
			return
		}
	}

	return
}

// expandString is the internal version of ExpandString that doesn't
// acquire the lock when calling other functions. It returns the
// configuration c value for input as an expanded string. The returned
// string is always a freshly allocated value.
func (c *Config) expandString(input string, options ...ExpandOptions) (value string) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return strings.Clone(input)
		}
		// return a *copy* of the initialValue or defaultValue
		if opts.initialValue != nil {
			return fmt.Sprint(opts.initialValue)
		}
		return fmt.Sprint(opts.defaultValue)
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandString(input, func(s string) (r string) {
		if strings.HasPrefix(s, "enc:") {
			if opts.nodecode {
				// return string and restore containing ${...}
				return `${` + s + `}`
			}
			return c.expandEncodedString(s[4:], options...)
		}
		r, _ = c.expandRawString(s, options...)
		return
	})

	if opts.trimSpace {
		value = strings.TrimSpace(value)
	}

	if value == "" {
		value = fmt.Sprint(opts.defaultValue)
	}

	// return a clone
	return strings.Clone(value)
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func (c *Config) expandStringSlice(input []string, options ...ExpandOptions) (vals []string) {
	for _, v := range input {
		vals = append(vals, c.expandString(v, options...))
	}
	return
}

// expandToEnclave expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) expandToEnclave(input string, options ...ExpandOptions) (value *memguard.Enclave) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return memguard.NewEnclave([]byte(input))
		}

		// fallback to any default value or, failing that, an initial value
		if opts.defaultValue != nil {
			return memguard.NewEnclave(fmt.Append(nil, opts.defaultValue))
		} else if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return memguard.NewEnclave(b)
			} else {
				return memguard.NewEnclave(fmt.Append(nil, opts.initialValue))
			}
		}
		return &memguard.Enclave{}
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandToEnclave([]byte(input), func(s []byte) (r *memguard.Enclave) {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				return memguard.NewEnclave(fmt.Append(nil, `${`, s, `}`))
			}
			return c.expandEncodedBytesEnclave(s[4:], options...)
		}
		str, _ := c.expandRawString(string(s), options...)
		return memguard.NewEnclave([]byte(str))
	})

	if value == nil || value.Size() == 0 {
		// return a *copy* of the defaultValue, don't let memguard wipe it!
		return memguard.NewEnclave(fmt.Append(nil, opts.defaultValue))
	}

	return
}

// ExpandToLockedBuffer expands the input string and returns a sealed
// enclave. The option TrimSpace is ignored.
func (c *Config) expandToLockedBuffer(input string, options ...ExpandOptions) (value *memguard.LockedBuffer) {
	opts := evalExpandOptions(c, options...)
	if opts.rawstring {
		if input != "" {
			return memguard.NewBufferFromBytes([]byte(input))
		}
		if opts.initialValue != nil {
			if b, ok := opts.initialValue.([]byte); ok {
				return memguard.NewBufferFromBytes(b)
			}
			return memguard.NewBufferFromBytes(fmt.Append(nil, opts.initialValue))
		}
		return memguard.NewBufferFromBytes(fmt.Append(nil, opts.defaultValue))
	}

	if input == "" && opts.initialValue != nil {
		input = fmt.Sprint(opts.initialValue)
	}

	value = expandToLockedBuffer([]byte(input), func(s []byte) *memguard.LockedBuffer {
		if bytes.HasPrefix(s, []byte("enc:")) {
			if opts.nodecode {
				return memguard.NewBufferFromBytes(fmt.Append([]byte{}, `${`, s, `}`))
			}
			return c.expandEncodedBytesLockedBuffer(s[4:], options...)
		}
		str, _ := c.expandRawString(string(s), options...)
		return memguard.NewBufferFromBytes([]byte(str))
	})

	if value == nil || value.Size() == 0 {
		// return a *copy* of the defaultvalue, don't let memguard wipe it!
		return memguard.NewBufferFromBytes(fmt.Append(nil, opts.defaultValue))
	}

	return
}

func get[T any](c *Config, key string, options ...ExpandOptions) (value T) {
	switch any(*new(T)).(type) {
	case bool:
		return any(c.getBool(key, options...)).(T)
	case int:
		return any(c.getInt(key, options...)).(T)
	case int64:
		return any(c.getInt64(key, options...)).(T)
	case uint:
		return any(c.getUint(key)).(T)
	case uint16:
		return any(c.getUint16(key)).(T)
	case float64:
		return any(c.getFloat64(key)).(T)
	case string:
		return any(c.getString(key, options...)).(T)
	case []byte:
		return any(c.getBytes(key, options...)).(T)
	case []string:
		return any(c.getStringSlice(key, options...)).(T)
	case map[string]any:
		return any(c.getStringMap(key)).(T)
	case map[string]string:
		return any(c.getStringMapString(key, options...)).(T)
	case []map[string]string:
		return any(c.getSliceStringMapString(key, options...)).(T)
	case time.Duration:
		return any(c.getDuration(key, options...)).(T)
	default:
		return any(c.get(key)).(T)
	}
}

func (c *Config) get(key string) (value any) {
	return c.Viper.Get(key)
}

func (c *Config) getBool(key string, options ...ExpandOptions) (value bool) {
	s := c.getString(key, options...)
	value, _ = strconv.ParseBool(s)
	return
}

func (c *Config) getDuration(key string, options ...ExpandOptions) (value time.Duration) {
	s := c.getString(key, options...)
	value, _ = time.ParseDuration(s)
	return
}

func (c *Config) getInt(key string, options ...ExpandOptions) (value int) {
	s := c.getString(key, options...)
	value, _ = strconv.Atoi(s)
	return
}

func (c *Config) getInt64(key string, options ...ExpandOptions) (value int64) {
	s := c.getString(key, options...)
	value, _ = strconv.ParseInt(s, 10, 64)
	return
}

func (c *Config) getUint(key string) (value uint) {
	return c.Viper.GetUint(key)
}

func (c *Config) getUint16(key string) (value uint16) {
	return c.Viper.GetUint16(key)
}

func (c *Config) getFloat64(key string) (value float64) {
	return c.Viper.GetFloat64(key)
}

func (c *Config) getBytes(key string, options ...ExpandOptions) (value []byte) {
	str := c.Viper.GetString(key)
	return []byte(str)
}

// getString functions like [viper.GetString] on a Config instance, but
// additionally calls [ExpandString] with the configuration value, passing
// any "values" maps
func (c *Config) getString(s string, options ...ExpandOptions) string {
	str := c.Viper.GetString(s)

	return c.expandString(str, options...)
}

// getStringSlice is the internal function that does not lock the
// configuration structure and functions like [viper.GetStringSlice] on
// a Config instance but additionally calls [expandString] on each
// element of the slice, passing any "values" maps
func (c *Config) getStringSlice(key string, options ...ExpandOptions) (slice []string) {
	var result []string
	opts := evalExpandOptions(c, options...)

	if c.isSet(key) {
		result = c.Viper.GetStringSlice(key)
	} else if init, ok := opts.initialValue.([]string); ok {
		result = init
	}

	if len(result) == 0 {
		if def, ok := opts.defaultValue.([]string); ok {
			result = def
		}
	}

	for _, n := range result {
		slice = append(slice, c.expandString(n, options...))
	}
	return
}

// getStringMap functions like [viper.GetStringMap] on a Config instance
func (c *Config) getStringMap(key string) (value map[string]any) {
	value = c.Viper.GetStringMap(key)
	if value == nil {
		value = make(map[string]any)
	}
	return
}

// getStringMapString functions like [viper.GetStringMapString] on a
// Config instance but additionally calls [ExpandString] on each value
// element of the map, passing any "values" maps
//
// Use a version of https://github.com/spf13/viper/pull/1504 to fix viper bug #1106
func (c *Config) getStringMapString(key string, options ...ExpandOptions) (m map[string]string) {
	m = make(map[string]string)

	key = strings.ToLower(key)
	prefix := key + c.delimiter

	i := c.Viper.Get(key)

	if !isStringMapInterface(i) {
		return
	}
	val := i.(map[string]string)
	keys := c.allKeys()
	for _, k := range keys {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		mk := strings.TrimPrefix(key, prefix)
		mk = strings.Split(mk, c.delimiter)[0]
		if _, exists := val[mk]; exists {
			continue
		}
		mv := get[string](c, key+c.delimiter+mk)
		if mv == "" {
			continue
		}
		val[mk] = mv
	}

	for k, v := range val {
		m[k] = c.expandString(fmt.Sprint(v), options...)
	}

	return
}

// GetSliceStringMapString returns a slice of string maps for the key s,
// it iterates over all values in all maps and applies the ExpandString
// with the options given
func (c *Config) getSliceStringMapString(s string, options ...ExpandOptions) (result []map[string]string) {
	if err := c.unmarshalKey(s, &result); err != nil {
		return
	}
	for _, m := range result {
		for k, v := range m {
			m[k] = c.expandString(v, options...)
		}
	}
	return
}

func (c *Config) getStringMapStringSlice(key string, options ...ExpandOptions) (m map[string][]string) {
	return c.Viper.GetStringMapStringSlice(key)
}

func (c *Config) isSet(key string) (value bool) {
	return c.Viper.IsSet(key)
}

func (c *Config) mergeConfigMap(vals map[string]any) (err error) {
	return c.Viper.MergeConfigMap(vals)
}

// set a value without locking. This is used by the public Set() method
// and also by Save() to set values in sub-configs without needing to
// acquire the lock multiple times. It assumes that the caller has
// already acquired the lock if needed.
func (c *Config) set(key string, value any) {
	c.Viper.Set(key, value)
}

func (c *Config) setDefault(key string, value any) {
	c.Viper.SetDefault(key, value)
}

func (c *Config) setEnvPrefix(prefix string) {
	c.Viper.SetEnvPrefix(prefix)
}

func (c *Config) automaticEnv() {
	c.Viper.AutomaticEnv()
}

func (c *Config) bindEnv(input ...string) error {
	return c.Viper.BindEnv(input...)
}

func (c *Config) registerAlias(alias, key string) {
	c.Viper.RegisterAlias(alias, key)
}

func (c *Config) configFileUsed() (f string) {
	return c.Viper.ConfigFileUsed()
}

// setString is the internal version of SetString that doesn't acquire
// the lock. It sets the given key in the configuration structure c to
// the string given after processing options. Options include replacing
// substrings with configuration items that match *at the time of the
// setString call*. This allows the abstraction of a static string based
// on the other config values given. E.g.
//
//	cf.setString("setup", "/path/to/myname/setup.json", config.Replace("name"))
//
// This would check the value of the "name" key in cf and do a global
// replace. Multiple Replace options are processed in order. If "name"
// was "myname" at the time of the call then the resulting value is
// `/path/to/${config:name}/setup.json`
//
// Existing expand options are left unchanged. All replacements are case
// sensitive.
func (c *Config) setString(key, value string, options ...ExpandOptions) {
	value = c.replaceString(value, options...)
	c.set(key, value)
}

// setStringSlice sets the key to a slice of strings applying the
// replacement options as for setString to each member of the slice
func (c *Config) setStringSlice(key string, values []string, options ...ExpandOptions) {
	for i, v := range values {
		values[i] = c.replaceString(v, options...)
	}
	c.set(key, values)
}

// SetStringMapString iterates over a map[string]string and sets each
// key to the value given. Viper's Set() doesn't support maps until the
// configuration is written to and read back from a file.
func (c *Config) setStringMapString(key string, vals map[string]string, options ...ExpandOptions) {
	for k, v := range vals {
		c.setString(key+c.delimiter+k, v, options...)
	}
}

// SetKeyValues takes a list of `key=value` pairs as strings and applies
// them to the config object. Any item without an `=` is skipped.
//
// If the separator is either `+=` or `+` then the given value is
// appended to any existing setting. If the value is starts with a dash
// then it is considered a command line option and is appended with a
// space separator, otherwise it is simply concatenated.
func (c *Config) setKeyValues(items ...string) (err error) {
	for _, item := range items {
		fields := itemRE.FindStringSubmatch(item)
		if len(fields) == 0 {
			return fmt.Errorf("item %q is not a valid setting", item)
		}

		switch fields[2] {
		case "=":
			c.set(fields[1], fields[3])
		case "+=", "+":
			if !c.isSet(fields[1]) {
				c.set(fields[1], fields[3])
				continue
			}

			if strings.HasPrefix(fields[3], "-") {
				c.set(fields[1], c.getString(fields[1], NoExpand())+" "+fields[3])
			} else {
				c.set(fields[1], c.getString(fields[1], NoExpand())+fields[3])
			}
		default:
			continue
		}
	}
	return
}

// sub is the internal version of Sub that doesn't acquire the read
// lock. This is used by Sub itself and also by Save to avoid acquiring
// the lock multiple times when iterating over keys and creating
// sub-configs for saving. It assumes that the caller has already
// acquired the read lock if needed.
func (c *Config) sub(key string) *Config {
	vcf := c.Viper.Sub(key)

	if vcf == nil {
		vcf = viper.New()
	}
	return &Config{
		Viper:                vcf,
		mutex:                &sync.RWMutex{}, // never copy mutex, always create a new one
		Type:                 c.Type,
		delimiter:            c.delimiter,
		defaultExpandOptions: c.defaultExpandOptions,
		appUserConfDir:       c.appUserConfDir,
	}
}

func (c *Config) unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.Unmarshal(rawVal, opts...)
}

func (c *Config) unmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	key = strings.ToLower(key)
	prefix := key + c.delimiter

	i := c.Viper.Get(key)

	if isStringMapInterface(i) {
		val := i.(map[string]any)
		keys := c.allKeys()
		for _, k := range keys {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			mk := strings.TrimPrefix(k, prefix)
			mk = strings.Split(mk, c.delimiter)[0]
			if _, exists := val[mk]; exists {
				continue
			}
			mv := c.get(key + c.delimiter + mk)
			if mv == nil {
				continue
			}
			val[mk] = mv
		}
		i = val
	}

	return decode(i, defaultDecoderConfig(rawVal, opts...))
}

// internal routines

// replaceString does the string replacement for the Set* functions
func (c *Config) replaceString(value string, options ...ExpandOptions) string {
	opts := evalExpandOptions(c, options...)

	if len(opts.replacements) == 0 {
		return value
	}

	for _, r := range opts.replacements {
		sub := c.getString(r)

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

// splitEncFields breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext. The split is done on the
// *last* colon and not the first, otherwise Windows drive letter paths
// would be considered the split point.
func splitEncFields(enc string) (keyfiles, ciphertext string) {
	c := strings.LastIndexByte(enc, ':')
	if c == -1 {
		return
	}
	keyfiles = enc[:c]
	if len(enc) > c+1 {
		ciphertext = enc[c+1:]
	}
	return
}

// splitEncFieldsBytes breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext. The split is done on the
// *last* colon and not the first, otherwise Windows drive letter paths
// would be considered the split point.
func splitEncFieldsBytes(enc []byte) (keyfiles, ciphertext []byte) {
	c := bytes.LastIndexByte(enc, ':')
	if c == -1 {
		return
	}
	keyfiles = enc[:c]
	if len(enc) > c+1 {
		ciphertext = enc[c+1:]
	}
	return
}

func fetchURL(_ map[string]any, url string, trim bool) (s string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if !trim {
		s = string(b)
	} else {
		s = strings.TrimSpace(string(b))
	}
	return
}

func fetchFile(_ map[string]any, p string, trim bool) (s string, err error) {
	b, err := os.ReadFile(ExpandHome(strings.TrimPrefix(p, "file:")))
	if err != nil {
		return
	}
	if !trim {
		s = string(b)
	} else {
		s = strings.TrimSpace(string(b))
	}
	return
}

func expr(configItems map[string]any, expression string, trim bool) (s string, err error) {
	vars := maps.Clone(configItems)
	eval := goval.NewEvaluator()
	env := make(map[string]string)
	for _, e := range os.Environ() {
		s := strings.SplitN(e, "=", 2)
		env[s[0]] = s[1]
	}
	vars["env"] = env
	result, err := eval.Evaluate(expression, vars, nil)
	if err != nil {
		return
	}
	if !trim {
		s = fmt.Sprint(result)
	} else {
		s = strings.TrimSpace(fmt.Sprint(result))
	}
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
		h, err := UserHomeDir()
		if err == nil {
			s = h
		}
	}
	return
}

func isStringMapInterface(val any) bool {
	if val == nil {
		return false
	}
	vt := reflect.TypeOf(val)
	return vt.Kind() == reflect.Map &&
		vt.Key().Kind() == reflect.String &&
		vt.Elem().Kind() == reflect.Interface
}

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices
func defaultDecoderConfig(output any, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
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
func decode(input any, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
