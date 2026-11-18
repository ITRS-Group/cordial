package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-viper/mapstructure/v2"
	"github.com/maja42/goval"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// SupportedExts lists the file extensions supported by the underlying config
// loader.
var SupportedExts = viper.SupportedExts

type config struct {
	*viper.Viper
}

func newConfig() *config {
	return &config{Viper: viper.New()}
}

func newWithOptions(opts *fileOptions) *config {
	envDelimiter := opts.envDelimiter
	if envDelimiter == "" {
		envDelimiter = "_"
	}
	return &config{Viper: viper.NewWithOptions(viper.KeyDelimiter(opts.delimiter),
		viper.EnvKeyReplacer(strings.NewReplacer(opts.delimiter, envDelimiter, "-", envDelimiter)))}
}

// internal routines that don't lock the config structure and also, for
// now, call the Viper methods directly. These are used by the public
// methods that do the locking and also by other internal methods that
// need to call Viper methods without locking, such as Sub() and Save().

// internal placeholders, for now

// allKeys returns all the keys in the config, excluding any that have
// been marked as deleted. The keys are returned in sorted order.
func (c *Config) allKeys() (keys []string) {
	k := c.config.AllKeys()
	keys = make([]string, 0, len(k))
	for _, key := range k {
		if _, deleted := c.config.Get(key).(deletedKey); !deleted {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func (c *Config) allSettings() (value map[string]any) {
	s := c.config.AllSettings()
	removeDeleted(s)
	return s
}

// removeDeleted removes, in place, all the keys marked as deleted from
// m and any maps nested in it. It is only for the maps created by
// viper for each call to AllSettings, which are not shared.
func removeDeleted(m map[string]any) {
	maps.DeleteFunc(m, func(_ string, v any) bool {
		// delete any keys that have been marked as deleted
		_, deleted := v.(deletedKey)
		return deleted
	})
	for _, v := range m {
		if nm, ok := v.(map[string]any); ok {
			removeDeleted(nm)
		}
	}
}

// withoutDeleted returns a copy of v without any keys marked as
// deleted, so that they are never seen by callers decoding values.
func withoutDeleted(v any) any {
	switch t := v.(type) {
	case deletedKey:
		return nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			if _, deleted := e.(deletedKey); !deleted {
				out[k] = withoutDeleted(e)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = withoutDeleted(e)
		}
		return out
	}
	return v
}

func (c *Config) setFs(fs afero.Fs) {
	c.config.SetFs(fs)
}

func (c *Config) setConfigType(t string) {
	c.config.SetConfigType(t)
}

func (c *Config) setConfigFile(f string) {
	c.config.SetConfigFile(f)
}

func (c *Config) readInConfig() (err error) {
	err = c.config.ReadInConfig()
	if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
		return fs.ErrNotExist
	}
	return
}

func (c *Config) readConfig(r io.Reader) (err error) {
	return c.config.ReadConfig(r)
}

func (c *Config) writeConfigAs(path string) (err error) {
	return c.config.WriteConfigAs(path)
}

func (c *Config) writeConfigTo(w io.Writer) (err error) {
	return c.config.WriteConfigTo(w)
}

func (c *Config) configFileUsed() (f string) {
	return c.config.ConfigFileUsed()
}

func (c *Config) onConfigChange(run func(in fsnotify.Event)) {
	c.config.OnConfigChange(run)
}

func (c *Config) watchConfig() {
	c.config.WatchConfig()
}

// automaticEnv enables lookup of environment variables for
// configuration keys. When this is enabled, the configuration system
// will automatically check for environment variables that match the
// keys in the configuration, using the prefix set by SetEnvPrefix if
// applicable. This allows environment variables to override
// configuration values without explicitly binding each key to an
// environment variable.
func (c *Config) automaticEnv() {
	c.config.AutomaticEnv()
}

// bindEnv binds the named configuration key to one or more environment
// variables. The first argument is the key in the configuration, and
// the subsequent arguments are the environment variable names to bind
// to that key. If no environment variable names are provided, it will
// default to using prefix + "_" + key as the environment variable name.
//
// prefix refers to the environment variable prefix set by SetEnvPrefix.
func (c *Config) bindEnv(key string, input ...string) error {
	return c.config.BindEnv(append([]string{key}, input...)...)
}

// bindPFlag binds a pflag.Flag to a key in the configuration.
func (c *Config) bindPFlag(key string, flag *pflag.Flag) (err error) {
	return c.config.BindPFlag(key, flag)
}

func (c *Config) registerAlias(alias, key string) {
	c.config.RegisterAlias(alias, key)
}

// expand is the internal function behind [Expand]. See the package
// documentation for the rules.
func expand[T string | []byte](c *Config, input string, options ...ExpandOption) (value T) {
	return T(expandString(c, evalExpandOptions(c, options...), input))
}

// ExpandAllSettings returns all the settings from config structure c
// applying Expand to all string values and all string slice
// values. Non-string types are left unchanged. Further types, e.g. maps
// of strings, may be added in future releases.
func (c *Config) expandAllSettings(options ...ExpandOption) (all map[string]any) {
	opts := evalExpandOptions(c, options...)
	as := c.allSettings()
	all = make(map[string]any, len(as))

	for k, v := range as {
		switch ev := v.(type) {
		case deletedKey:
			// skip deleted keys
			continue
		case string:
			all[k] = expandString(c, opts, ev)
		case []string:
			if len(ev) == 0 {
				all[k] = []string{}
				continue
			}
			all[k] = expandStrings(c, opts, ev)
		case map[string]any:
			nm := make(map[string]any, len(ev))
			for kk, vv := range ev {
				if vvs, ok := vv.(string); ok {
					nm[kk] = expandString(c, opts, vvs)
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

func expandStrings(c *Config, opts *expandOptions, input []string) []string {
	if len(input) == 0 {
		return nil
	}
	vals := make([]string, 0, len(input))
	for _, v := range input {
		vals = append(vals, expandString(c, opts, v))
	}
	return vals
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func (c *Config) expandStringSlice(input []string, options ...ExpandOption) (vals []string) {
	return expandStrings(c, evalExpandOptions(c, options...), input)
}

// getContents returns the string inside braces, checking for embedded
// braces and the number of bytes consumed to extract it. The contents
// must be enclosed in {} and two more bytes are needed than the length
// of the name.
//
// CHANGE: return if string does not start with an opening bracket
//
// CHANGE: skip any character after a backslash, including closing
// braces
//
// CHANGE: match embedded opening braces and closing ones inside the
// string.
func getContents(s string) (string, int) {
	// must start with an opening brace
	if s[0] != '{' {
		// skip
		return "", 0
	}

	// Scan to closing brace, skipping backslash+next and stacking opening braces
	var depth int
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
				continue
			}
			if i == 1 {
				return "", 2 // Bad syntax; eat "${}"
			}
			return s[1:i], i + 1
		default:
		}
	}
	return "", 1 // Bad syntax; eat "${"
}

// lookup is the internal function that returns a second boolean value
// indicating whether the key was found in the config. The type is not
// checked, leaving get() to fixup mappings. `found` is true only if the
// configuration item is set, not if the options provide a default
// value.
func lookup[T any](c *Config, key string, options ...ExpandOption) (value T, found bool) {
	if !c.isSet(key) {
		return
	}

	return get[T](c, key, options...), true
}

// fromString converts the already expanded string raw to the scalar
// type T. ok is false if T is not one of the scalar types supported.
func fromString[T any](c *Config, opts *expandOptions, raw string) (value T, ok bool) {
	switch any(value).(type) {
	case string:
		return any(expandString(c, opts, raw)).(T), true
	case []byte:
		return any([]byte(expandString(c, opts, raw))).(T), true
	case Secret:
		return any(Secret(expandString(c, opts, raw))).(T), true
	case bool:
		v, _ := strconv.ParseBool(expandString(c, opts, raw))
		return any(v).(T), true
	case int:
		v, _ := strconv.ParseInt(expandString(c, opts, raw), 10, 0)
		return any(int(v)).(T), true
	case int64:
		v, _ := strconv.ParseInt(expandString(c, opts, raw), 10, 64)
		return any(v).(T), true
	case uint:
		v, _ := strconv.ParseUint(expandString(c, opts, raw), 10, 0)
		return any(uint(v)).(T), true
	case uint16:
		v, _ := strconv.ParseUint(expandString(c, opts, raw), 10, 16)
		return any(uint16(v)).(T), true
	case float64:
		v, _ := strconv.ParseFloat(expandString(c, opts, raw), 64)
		return any(v).(T), true
	case time.Duration:
		v, _ := time.ParseDuration(expandString(c, opts, raw))
		return any(v).(T), true
	}
	return
}

// get is the internal function that returns the value associated with
// the key in the configuration structure c, applying the options given.
// The type T is the expected type of the value, which can be one of:
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
//   - map[string][]string
//   - []map[string]string
//   - time.Duration
//   - config.Secret
//
// If the key is not set then the zero value for the type is returned,
// with maps and slices initialised to an empty value to avoid nil
// pointer panics in callers that expect a map or slice. If the option
// [`config.DefaultValue`] is used, then the type must be identical to
// T. If it is not, then the default value is the zero value for the
// type T.
func get[T any](c *Config, key string, options ...ExpandOption) (value T) {
	opts := evalExpandOptions(c, options...)

	if !c.isSet(key) {
		// the initial value is expanded, like any configuration value,
		// unless NoExpand is used
		if !isZero(opts.initialValue) {
			if v, ok := opts.initialValue.(T); ok {
				if !opts.noExpand {
					switch iv := any(v).(type) {
					case string:
						return any(expandString(c, opts, iv)).(T)
					case []string:
						return any(expandStrings(c, opts, iv)).(T)
					}
				}
				return v
			}
			if s, ok := opts.initialValue.(string); ok && !opts.noExpand {
				v, _ := fromString[T](c, opts, s)
				return v
			}
			return
		}

		// the default value is returned as-is
		if !isZero(opts.defaultValue) {
			if v, ok := opts.defaultValue.(T); ok {
				return v
			}
			return
		}

		// otherwise return zero value for type T, but checking for
		// supported map types and slices and returning empty ones of
		// those types rather than nil, to avoid nil pointer panics in
		// callers that expect a map or slice.
		var zero T
		switch any(zero).(type) {
		case map[string]any:
			return any(make(map[string]any)).(T)
		case map[string]string:
			return any(make(map[string]string)).(T)
		case map[string][]string:
			return any(make(map[string][]string)).(T)
		case []map[string]string:
			return any(make([]map[string]string, 0)).(T)
		case []string:
			return any(make([]string, 0)).(T)
		default:
			return zero
		}
	}

	switch any(*new(T)).(type) {
	case bool, int, int64, uint, uint16, float64, string, []byte, time.Duration, Secret:
		v, _ := fromString[T](c, opts, c.config.GetString(key))
		return v
	case []string:
		return any(expandStrings(c, opts, c.config.GetStringSlice(key))).(T)
	case map[string]any:
		var result map[string]any
		if err := c.unmarshalKey(key, &result, options...); err != nil {
			return
		}
		return any(result).(T)
	case map[string]string:
		var result map[string]string
		if err := c.unmarshalKey(key, &result, options...); err != nil {
			return
		}
		return any(result).(T)
	case []map[string]string:
		var result []map[string]string
		if err := c.unmarshalKey(key, &result, options...); err != nil {
			return
		}
		return any(result).(T)
	case map[string][]string:
		var result map[string][]string
		if err := c.unmarshalKey(key, &result, options...); err != nil {
			return
		}
		return any(result).(T)
	default:
		var result T
		if err := c.unmarshalKey(key, &result, options...); err != nil {
			return
		}
		return any(result).(T)
	}
}

// promote moves the first set legacy key, as given by the PromoteFrom
// option, to key if key is not set. The value is moved as-is, without
// expansion, so that secrets are not written out decoded. It takes the
// write lock itself, only if there is something to do.
func (c *Config) promote(key string, options ...ExpandOption) {
	c.rwmutex.RLock()
	if len(options) == 0 && len(c.defaultExpandOptions) == 0 {
		c.rwmutex.RUnlock()
		return
	}
	opts := evalExpandOptions(c, options...)
	need := len(opts.promoteFrom) > 0 && !c.isSet(key)
	c.rwmutex.RUnlock()
	if !need {
		return
	}

	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()
	if c.isSet(key) {
		return
	}
	for _, legacy := range opts.promoteFrom {
		// prevent infinite recursion if the legacy key is the same as
		// the new key
		if legacy == key || !c.isSet(legacy) {
			continue
		}
		c.config.Set(key, c.config.Get(legacy))
		deleteKey(c, legacy)
		return
	}
}

type deletedKey struct{}

// delete sets a key to a special value that indicates it has been
// deleted. Our get and lookup functions will treat this as not set,
// even though it is technically.
//
// TODO: save routines should check for this value and not save it, to
// avoid confusion if the config file is edited by hand.
func deleteKey(c *Config, key string) {
	c.config.Set(key, deletedKey{})
}

// set a value
func set[T any](c *Config, key string, value T, options ...ExpandOption) {
	opts := evalExpandOptions(c, options...)

	if opts.noExpand {
		c.config.Set(key, value)
		return
	}

	switch vt := any(value).(type) {
	case string:
		c.config.Set(key, c.replaceStringParam(vt, opts))
	case []string:
		// do not modify the caller's slice
		nv := make([]string, len(vt))
		for i, v2 := range vt {
			nv[i] = c.replaceStringParam(v2, opts)
		}
		c.config.Set(key, nv)
	case map[string]string:
		// grab the underlying map before setting new values - this must come first
		m := c.config.GetStringMapString(key)

		// viper lower-cases all map keys, so remember the new keys in
		// that form to compare with the existing ones
		keep := make(map[string]struct{}, len(vt))
		for k, v := range vt {
			keep[strings.ToLower(k)] = struct{}{}
			c.config.Set(key+c.delimiter+k, c.replaceStringParam(v, opts))
		}

		// for maps we have to also delete any keys that have been
		// removed from the original map, viper is broken
		for k := range m {
			if _, found := keep[k]; !found {
				c.config.Set(key+c.delimiter+k, deletedKey{})
			}
		}

	case map[string]any:
		// grab the underlying map before setting new values - this must come first
		m := c.config.GetStringMap(key)

		// set each key individually in the config, as viper is broken
		// in the way it handles maps and deep keys, so we can't just
		// set the whole map
		keep := make(map[string]struct{}, len(vt))
		for k, v := range vt {
			keep[strings.ToLower(k)] = struct{}{}
			c.config.Set(key+c.delimiter+k, v)
		}

		// for maps we have to also delete any keys that have been
		// removed from the original map, viper is broken
		for k := range m {
			if _, found := keep[k]; !found {
				c.config.Set(key+c.delimiter+k, deletedKey{})
			}
		}

	default:
		// no replacement needed for non-string types, but still need to
		// set the value in the config
		c.config.Set(key, vt)
	}
}

func (c *Config) isSet(key string) (value bool) {
	set := c.config.IsSet(key)
	deleted := false
	if set {
		// check if the value is the deleted marker
		if _, ok := c.config.Get(key).(deletedKey); ok {
			deleted = true
		}
	}
	return set && !deleted
}

func (c *Config) mergeConfigMap(vals map[string]any) (err error) {
	return c.config.MergeConfigMap(vals)
}

func (c *Config) setDefault(key string, value any) {
	c.config.SetDefault(key, value)
}

// setEnvPrefix sets the prefix used when mapping environment variables
// to configuration keys. This prefix will be prepended to all
// environment variable names when looking up configuration values. For
// example, if the prefix is set to "ITRS" and the key is "GENEOS_HOME",
// the environment variable "ITRS_GENEOS_HOME" will be used.
func (c *Config) setEnvPrefix(prefix string) {
	c.config.SetEnvPrefix(prefix)
}

// itemRE is used to parse key-value pairs passed as strings, e.g. from
// command line arguments, in the form `key=value` or `key+=value` for
// appending to existing values. The key can contain word characters,
// dots, colons and hyphens, and the separator can be either `=` or `+=`
// (or `+` for backward compatibility). The value is everything after
// the separator, so `key==value` sets the value to `=value`. The regex
// captures the key, separator and value in three groups.
var itemRE = regexp.MustCompile(`^([\w\.\:-]+)(\+=|\+|=)(.*)`)

// setKeyValuePairs takes a list of `key=value` pairs as strings and
// applies them to the config object. Any item that is not of the form
// `key=value`, `key+=value` or `key+value` causes an error to be
// returned.
//
// If the separator is either `+=` or `+` then the given value is
// appended to any existing setting. If the value is starts with a dash
// then it is considered a command line option and is appended with a
// space separator, otherwise it is simply concatenated.
func (c *Config) setKeyValuePairs(items ...string) (err error) {
	for _, item := range items {
		fields := itemRE.FindStringSubmatch(item)
		if len(fields) == 0 {
			return fmt.Errorf("item %q is not a valid setting", item)
		}

		switch fields[2] {
		case "=":
			set(c, fields[1], fields[3])
		case "+=", "+":
			if !c.isSet(fields[1]) {
				set(c, fields[1], fields[3])
				continue
			}

			if strings.HasPrefix(fields[3], "-") {
				set(c, fields[1], get[string](c, fields[1], NoExpand())+" "+fields[3])
			} else {
				set(c, fields[1], get[string](c, fields[1], NoExpand())+fields[3])
			}
		}
	}
	return
}

// sub is the internal version of Sub that doesn't acquire the read
// lock. It assumes that the caller has already acquired the read lock
// if needed.
func (c *Config) sub(key string) *Config {
	vcf := &config{Viper: c.config.Sub(key)}

	if vcf.Viper == nil {
		// keep the delimiter, or nested keys set on the empty config
		// would be stored under the wrong names
		vcf = newWithOptions(&fileOptions{delimiter: c.delimiter, envDelimiter: "_"})
	}
	return &Config{
		config:               vcf,
		rwmutex:              &sync.RWMutex{}, // never copy mutex, always create a new one
		configType:           c.configType,
		delimiter:            c.delimiter,
		defaultExpandOptions: c.defaultExpandOptions,
		appUserConfDir:       c.appUserConfDir,
	}
}

// unmarshalKey is a wrapper around Viper's UnmarshalKey that uses the
// same default decoder configuration as our decode function, ensuring
// that time.Duration values and string slices are properly handled when
// unmarshalling into a struct. A key not being set is not an error.
//
// All string values are still subject to expansion, so the same options
// can be passed to control that, e.g. NoExpand() to disable expansion
// of string values when unmarshalling.
func (c *Config) unmarshalKey(key string, rawVal any, options ...ExpandOption) error {
	// do not append to the caller's slice
	options = append(slices.Clone(options), IncludeConfig(c))
	return decode(withoutDeleted(c.config.Get(key)), defaultDecoderConfig(c, rawVal, options...))
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input any, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// expandFieldsHook returns a mapstructure.DecodeHookFunc that expands
// string fields using the options given. This is intended to be used
// in Unmarshal calls to allow for dynamic values in the configuration
// file. The hook will only expand string fields, and will leave other
// types unchanged.
func expandFieldsHook(c *Config, opts *expandOptions) mapstructure.DecodeHookFunc {
	if opts.cf != nil {
		c = opts.cf
	}
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		// the value may be a named string type
		str := expandString(c, opts, reflect.ValueOf(data).String())
		return reflect.ValueOf(str).Convert(f).Interface(), nil
	}
}

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices. The options, including any
// default options set on c, determine if and how strings are expanded.
func defaultDecoderConfig(c *Config, output any, options ...ExpandOption) *mapstructure.DecoderConfig {
	dc := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          "mapstructure,json,yaml",
	}
	hooks := []mapstructure.DecodeHookFunc{
		mapstructure.StringToTimeHookFunc(time.RFC3339),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.TextUnmarshallerHookFunc(),
	}
	if opts := evalExpandOptions(c, options...); !opts.noExpand {
		hooks = slices.Insert(hooks, 0, expandFieldsHook(c, opts))
	}
	dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(hooks...)
	return dc
}

// replaceStringParam does the string replacement for the set function,
// replacing any config items in the value with their expanded values.
// It is careful to skip any config items that are part of an expand
// substring, e.g. `${config:foo}`. If the value to be replaced is
// prefixed with `config:` then it is replaced with `${config:...}` to
// ensure that it is not re-expanded when the value is later expanded as
// a whole.
func (c *Config) replaceStringParam(value string, opts *expandOptions) string {
	for _, r := range opts.replacements {
		// e.g. "home" -> "/opt/itrs/geneos"
		sub := get[string](c, r)

		// an empty value would match between every character
		if sub == "" {
			continue
		}

		// simple case, no expandable substrings
		if !strings.Contains(value, "${") {
			value = strings.ReplaceAll(value, sub, "${config:"+r+"}")
			continue
		}

		// iterate over value, skipping expand substrings
		var newValue strings.Builder
		remainingValue := value

		for {
			start := strings.Index(remainingValue, "${")
			if start == -1 {
				newValue.WriteString(strings.ReplaceAll(remainingValue, sub, "${config:"+r+"}"))
				break
			}
			end := strings.Index(remainingValue[start:], "}")
			if end == -1 {
				// an unterminated expand substring is treated as
				// ending the string, so copy the rest unchanged
				newValue.WriteString(strings.ReplaceAll(remainingValue[:start], sub, "${config:"+r+"}"))
				newValue.WriteString(remainingValue[start:])
				break
			}
			end += start
			// append substituted nonexpand substring
			newValue.WriteString(strings.ReplaceAll(remainingValue[:start], sub, "${config:"+r+"}"))
			// append expand substring
			newValue.WriteString(remainingValue[start : end+1])
			// remove above from remaining
			remainingValue = remainingValue[end+1:]
		}
		value = newValue.String()
	}

	return value
}

// fetchClient is used for all the expansion and input fetches from
// URLs. It has a timeout so that a stalled server cannot block
// expansion forever.
var fetchClient = &http.Client{Timeout: 30 * time.Second}

// fetchURL retrieves the contents of the specified URL. The first
// argument is a map of config items that may be used for variable
// substitution in the URL, but is currently unused. The trim parameter
// determines whether to trim whitespace from the retrieved content
// before returning it as a string.
func fetchURL(_ map[string]any, url string, trim bool, _ ...any) (s string, err error) {
	resp, err := fetchClient.Get(url) // url includes the prefix
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		err = fmt.Errorf("error fetching URL %q: status code %d", url, resp.StatusCode)
		return
	}
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

// fetchFile reads the contents of a file specified by the given path.
// If the path is prefixed with "file:", it is stripped before reading.
// The function also supports expanding the home directory if the path
// starts with "~/". The trim parameter determines whether to trim
// whitespace from the file contents before returning it as a string.
func fetchFile(_ map[string]any, p string, trim bool, _ ...any) (s string, err error) {
	b, err := os.ReadFile(ResolveHome(strings.TrimPrefix(p, "file:")))
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

// expr evaluates the expression using the `goval` library, with the given
// config items as variables. The expression can reference config items
// by name, and also environment variables via the `env` variable, which
// is a map of environment variable names to values. If the expression
// evaluates successfully then the result is returned as a string, with
// optional trimming of whitespace.
func expr(configItems map[string]any, expression string, trim bool, _ ...any) (s string, err error) {
	// the function is passed the whole of the expansion, including the prefix
	expression = strings.TrimPrefix(expression, "expr:")

	vars := maps.Clone(configItems)
	if vars == nil {
		vars = make(map[string]any, 1)
	}
	eval := goval.NewEvaluator()

	environ := os.Environ()
	env := make(map[string]string, len(environ))
	for _, e := range environ {
		if k, v, found := strings.Cut(e, "="); found {
			env[k] = v
		}
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
// can be substituted. Currently only HOME is supported as a special
// case for Windows.
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

// isZero checks if the given value is the zero value for its type. If
// the value is not valid return true as this implies and unset zero
// value, else return the result if reflect.Value.IsZero()
func isZero(n any) bool {
	v := reflect.ValueOf(n)
	if !v.IsValid() || v.IsZero() {
		return true
	}
	return false
}

func isEmpty(n any) bool {
	v := reflect.ValueOf(n)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return isZero(n)
	}
}
