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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/awnumar/memguard"
	"github.com/fsnotify/fsnotify"
	"github.com/go-viper/mapstructure/v2"
	"github.com/maja42/goval"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial/pkg/host"
)

// internal routines that don't lock the config structure and also, for
// now, call the Viper methods directly. These are used by the public
// methods that do the locking and also by other internal methods that
// need to call Viper methods without locking, such as Sub() and Save().

// internal placeholders, for now

func (c *Config) allKeys() (keys []string) {
	k := c.viper.AllKeys()
	for _, key := range k {
		if _, deleted := c.viper.Get(key).(deletedKey); !deleted {
			keys = append(keys, key)
		}
	}
	return keys
}

func (c *Config) allSettings() (value map[string]any) {
	s := c.viper.AllSettings()
	maps.DeleteFunc(s, func(_ string, v any) bool {
		// delete any keys that have been marked as deleted
		_, deleted := v.(deletedKey)
		return deleted
	})
	return s
}

func (c *Config) setFs(fs afero.Fs) {
	c.viper.SetFs(fs)
}

func (c *Config) setConfigType(t string) {
	c.viper.SetConfigType(t)
}

func (c *Config) setConfigFile(f string) {
	c.viper.SetConfigFile(f)
}

func (c *Config) readInConfig() (err error) {
	return c.viper.ReadInConfig()
}

func (c *Config) readConfig(r io.Reader) (err error) {
	return c.viper.ReadConfig(r)
}

func (c *Config) writeConfigAs(path string) (err error) {
	return c.viper.WriteConfigAs(path)
}

func (c *Config) writeConfigTo(w io.Writer) (err error) {
	return c.viper.WriteConfigTo(w)
}

func (c *Config) configFileUsed() (f string) {
	return c.viper.ConfigFileUsed()
}

func (c *Config) onConfigChange(run func(in fsnotify.Event)) {
	c.viper.OnConfigChange(run)
}

func (c *Config) watchConfig() {
	c.viper.WatchConfig()
}

func (c *Config) automaticEnv() {
	c.viper.AutomaticEnv()
}

func (c *Config) bindEnv(input ...string) error {
	return c.viper.BindEnv(input...)
}

func (c *Config) registerAlias(alias, key string) {
	c.viper.RegisterAlias(alias, key)
}

func expand[T string | []byte](c *Config, input string, options ...ExpandOptions) (value T) {
	opts := evalExpandOptions(c, options...)

	if opts.noExpand {
		if input != "" {
			return T(input)
		}

		if !isZero(opts.initialValue) {
			return T(fmt.Sprint(opts.initialValue))
		}

		if !isZero(opts.defaultValue) {
			return T(fmt.Sprint(opts.defaultValue))
		}

		return
	}

	if input == "" && !isZero(opts.initialValue) {
		input = fmt.Sprint(opts.initialValue)
	}

	value = _expand(input, func(s T) (r T) {
		if bytes.HasPrefix([]byte(s), []byte("enc:")) {
			if opts.noDecode {
				// return original string after restoring surrounding
				// ${...}
				return T(fmt.Sprint(`${`, s, `}`))
			}
			return expandEncoded(c, s[4:], options...)
		}
		str, _ := c.expandRawString(string(s), options...)
		return T(str)
	})

	if opts.trimSpace {
		value = T(strings.TrimSpace(string(value)))
	}

	if len(value) == 0 && !isZero(opts.defaultValue) {
		value = T(fmt.Sprint(opts.defaultValue))
	}

	return T(value)
}

// ExpandAllSettings returns all the settings from config structure c
// applying Expand to all string values and all string slice
// values. Non-string types are left unchanged. Further types, e.g. maps
// of strings, may be added in future releases.
func (c *Config) expandAllSettings(options ...ExpandOptions) (all map[string]any) {
	as := c.allSettings()
	all = make(map[string]any, len(as))

	for k, v := range as {
		switch ev := v.(type) {
		case string:
			all[k] = expand[string](c, ev, options...)
		case []string:
			ns := []string{}
			for _, s := range ev {
				ns = append(ns, expand[string](c, s, options...))
			}
			all[k] = ns
		case map[string]any:
			nm := make(map[string]any, len(ev))
			for kk, vv := range ev {
				if vvs, ok := vv.(string); ok {
					nm[kk] = expand[string](c, vvs, options...)
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
func expandEncoded[T string | []byte](c *Config, s T, options ...ExpandOptions) (value T) {
	opts := evalExpandOptions(c, options...)

	keyfiles, encodedValue := splitEncFields(string(s))

	if opts.useKeyFile != "" {
		keyfiles = opts.useKeyFile
	}

	if !strings.HasPrefix(encodedValue, "+encs+") {
		str, _ := c.expandRawString(encodedValue, options...)
		encodedValue = str
	}

	if len(encodedValue) == 0 {
		return
	}

	for _, k := range strings.Split(keyfiles, "|") {
		keyfile := KeyFile(ResolveHome(k))
		p, err := keyfile.DecodeString(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return T(p)
	}
	return
}

// splitEncFields breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext. The split is done on the
// *last* colon and not the first, otherwise Windows drive letter paths
// would be considered the split point.
func splitEncFields(enc string) (keyfiles, ciphertext string) {
	c := strings.LastIndex(enc, ":")
	if c == -1 {
		return
	}
	keyfiles = enc[:c]
	if len(enc) > c+1 {
		ciphertext = enc[c+1:]
	}
	return
}

// expandRawString is the internal function that expands the string s
// using the same rules and options as [Expand] but treats the
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
	// if the string looks like a file path and there is a "file"
	// function registered then try to read the file and return its
	// contents.
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

	// if the string is either a `config:` or a plain reference to a
	// config item, i.e. no colon
	case strings.HasPrefix(s, "config:"), !strings.Contains(s, ":"):
		// if the string is a config reference or contains a delimiter,
		// implying a config item, then try to resolve it
		if strings.HasPrefix(s, "config:") || strings.Contains(s, c.delimiter) {
			s = strings.TrimPrefix(s, "config:")
			if !opts.expandNonString {
				// this call to GetString() must NOT be recursive
				value = c.viper.GetString(s)
				if opts.trimSpace {
					value = strings.TrimSpace(value)
				}
				return
			}

			v := c.viper.Get(s)

			switch w := v.(type) {
			case deletedKey:
				value = ""
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

	// if the string is an environment variable reference then try to
	// resolve it
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

func _expand[T string | []byte](s string, mapping func(T) T) T {
	var buf []byte
	// // ${} is all ASCII, so bytes are fine for this operation.
	i := 0

	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getContents(s[j+1:])
			if name == "" {
				if w == 1 {
					// if invalid after opening `${` then return them
					// unchanged
					buf = append(buf, s[j:j+2]...)
				} else if w > 0 {
					// Encountered invalid syntax; eat the
					// characters.
				} else {
					// Valid syntax, but $ was not followed by a
					// name. Leave the dollar character untouched.
					buf = append(buf, s[j])
				}
			} else {
				buf = append(buf, mapping(T(name))...)
			}
			j += w
			i = j + 1
		}
	}

	if buf == nil {
		return T(s)
	}

	//
	// return string(buf) + s[i:]
	return T(append(buf, s[i:]...))
}

// ExpandStringSlice applies ExpandString to each member of the input
// slice
func (c *Config) expandStringSlice(input []string, options ...ExpandOptions) (vals []string) {
	for _, v := range input {
		vals = append(vals, expand[string](c, v, options...))
	}
	return
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
// indicating whether the key was found in the config and is of the
// correct type. `found` is true only if the configuration item is set,
// not if the options provide a default value.
func lookup[T any](c *Config, key string, options ...ExpandOptions) (value T, found bool) {
	if !c.isSet(key) {
		return
	}

	v := c.viper.Get(key)
	if _, ok := v.(T); !ok {
		return
	}

	return get[T](c, key, options...), true
}

func get[T any](c *Config, key string, options ...ExpandOptions) (value T) {
	if !c.isSet(key) {
		opts := evalExpandOptions(c, options...)
		if !isZero(opts.defaultValue) {
			if v, ok := opts.defaultValue.(T); ok {
				return v
			}
			log.Debug().Msgf("default value for key %q is not of type %T, returning zero value", key, value)
		}
		return
	}

	switch any(*new(T)).(type) {
	case bool:
		v, _ := strconv.ParseBool(expand[string](c, c.viper.GetString(key), options...))
		return any(v).(T)
	case int:
		v, _ := strconv.ParseInt(expand[string](c, c.viper.GetString(key), options...), 10, 0)
		return any(int(v)).(T)
	case int64:
		v, _ := strconv.ParseInt(expand[string](c, c.viper.GetString(key), options...), 10, 64)
		return any(v).(T)
	case uint:
		v, _ := strconv.ParseUint(expand[string](c, c.viper.GetString(key), options...), 10, 0)
		return any(uint(v)).(T)
	case uint16:
		v, _ := strconv.ParseUint(expand[string](c, c.viper.GetString(key), options...), 10, 16)
		return any(uint16(v)).(T)
	case float64:
		v, _ := strconv.ParseFloat(expand[string](c, c.viper.GetString(key), options...), 64)
		return any(v).(T)
	case string:
		return any(expand[string](c, c.viper.GetString(key), options...)).(T)
	case []byte:
		return any(expand[[]byte](c, c.viper.GetString(key), options...)).(T)
	case []string:
		var result []string
		opts := evalExpandOptions(c, options...)

		// if the key is set then use the value from the config,
		// otherwise use the initial value or default value if they are
		// set and of the correct type - leaving the decision to
		// expand() is too late as it needs to test and use the
		// correct type
		if c.isSet(key) {
			result = c.viper.GetStringSlice(key)
		} else if init, ok := opts.initialValue.([]string); ok {
			result = init
		} else if def, ok := opts.defaultValue.([]string); ok {
			result = def
		}

		var slice []string
		for _, n := range result {
			slice = append(slice, expand[string](c, n, options...))
		}
		return any(slice).(T)
	case map[string]any:
		v := c.viper.GetStringMap(key)
		if v == nil {
			v = make(map[string]any)
		}
		for k, v2 := range v {
			if s, ok := v2.(string); ok {
				v[k] = expand[string](c, s, options...)
			}
		}
		return any(v).(T)
	case map[string]string:
		var result map[string]string
		if err := c.unmarshalKey(key, &result); err != nil {
			return
		}
		for k, v := range result {
			result[k] = expand[string](c, v, options...)
		}
		return any(result).(T)
	case []map[string]string:
		var result []map[string]string
		if err := c.unmarshalKey(key, &result); err != nil {
			return
		}
		for _, m := range result {
			for k, v := range m {
				m[k] = expand[string](c, v, options...)
			}
		}
		return any(result).(T)
	case time.Duration:
		v, _ := time.ParseDuration(expand[string](c, c.viper.GetString(key), options...))
		return any(v).(T)
	case *Secret:
		return any(&Secret{memguard.NewEnclave(expand[[]byte](c, c.viper.GetString(key), options...))}).(T)
	default:
		return any(c.viper.Get(key)).(T)
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
	c.viper.Set(key, deletedKey{})
}

// set a value
func set[T any](c *Config, key string, value T, options ...ExpandOptions) {
	opts := evalExpandOptions(c, options...)

	if opts.noExpand {
		c.viper.Set(key, value)
		return
	}

	switch vt := any(value).(type) {
	case string:
		c.viper.Set(key, c.replaceString(vt, options...))
	case []string:
		for i, v2 := range vt {
			vt[i] = c.replaceString(v2, options...)
		}
		c.viper.Set(key, vt)
	case map[string]string:
		for k, v := range vt {
			c.viper.Set(key+c.delimiter+k, c.replaceString(v, options...))
		}
	default:
		// no replacement needed for non-string types, but still need to
		// set the value in the config
		c.viper.Set(key, vt)
	}
}

func (c *Config) isSet(key string) (value bool) {
	set := c.viper.IsSet(key)
	deleted := false
	if set {
		// check if the value is the deleted marker
		if _, ok := c.viper.Get(key).(deletedKey); ok {
			deleted = true
		}
	}
	return set && !deleted
}

func (c *Config) mergeConfigMap(vals map[string]any) (err error) {
	return c.viper.MergeConfigMap(vals)
}

func (c *Config) setDefault(key string, value any) {
	c.viper.SetDefault(key, value)
}

func (c *Config) setEnvPrefix(prefix string) {
	c.viper.SetEnvPrefix(prefix)
}

// itemRE is used to parse key-value pairs passed as strings, e.g. from
// command line arguments, in the form `key=value` or `key+=value` for
// appending to existing values. The key can contain word characters,
// dots, colons and hyphens, and the separator can be either `=` or `+=`
// (or `+` for backward compatibility). The value is everything after
// the separator. The regex captures the key, separator and value in
// three groups.
var itemRE = regexp.MustCompile(`^([\w\.\:-]+)([+=]=?)(.*)`)

// setKeyValuePairs takes a list of `key=value` pairs as strings and applies
// them to the config object. Any item without an `=` is skipped.
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
	vcf := c.viper.Sub(key)

	if vcf == nil {
		vcf = viper.New()
	}
	return &Config{
		viper:                vcf,
		mutex:                &sync.RWMutex{}, // never copy mutex, always create a new one
		configType:           c.configType,
		delimiter:            c.delimiter,
		defaultExpandOptions: c.defaultExpandOptions,
		appUserConfDir:       c.appUserConfDir,
	}
}

// unmarshalKey is a wrapper around Viper's UnmarshalKey that uses the same
// default decoder configuration as our decode function, ensuring that
// time.Duration values and string slices are properly handled when unmarshalling
// into a struct. A key not being set is not an error.
func (c *Config) unmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	return decode(c.viper.Get(key), defaultDecoderConfig(rawVal, opts...))
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input any, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
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

// replaceString does the string replacement for the set function,
// replacing any config items in the value with their expanded values.
// It is careful to skip any config items that are part of an expand
// substring, e.g. `${config:foo}`. If the value to be replaced is
// prefixed with `config:` then it is replaced with `${config:...}` to
// ensure that it is not re-expanded when the value is later expanded as
// a whole.
func (c *Config) replaceString(value string, options ...ExpandOptions) string {
	opts := evalExpandOptions(c, options...)

	for _, r := range opts.replacements {
		// e.g. "home" _> "/opt/itrs/geneos"
		sub := get[string](c, r, options...)

		// simple case, no expandable substrings
		if !strings.Contains(value, "${") {
			value = strings.ReplaceAll(value, sub, "${config:"+r+"}")
			continue
		}

		// iterate over value, skipping expand substrings
		var newValue string
		remainingValue := value

		for {
			start := strings.Index(remainingValue, "${")
			end := strings.Index(remainingValue, "}")
			if start == -1 || end == -1 {
				// finished with expand options. any unterminated
				// substring is treated as ending the string, so just
				// return the concatenated string
				value = newValue + remainingValue
				break
			}
			// append substituted nonexpand substring
			newValue += strings.ReplaceAll(remainingValue[:start], sub, "${config:"+r+"}")
			// append expand substring
			newValue += remainingValue[start : end+1]
			// remove above from remaining
			remainingValue = remainingValue[end+1:]
		}
	}

	return value
}

// fetchURL retrieves the contents of the specified URL. The first
// argument is a map of config items that may be used for variable
// substitution in the URL, but is currently unused. The trim parameter
// determines whether to trim whitespace from the retrieved content
// before returning it as a string.
func fetchURL(_ map[string]any, url string, trim bool) (s string, err error) {
	resp, err := http.Get(url) // url includes the prefix
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

// fetchFile reads the contents of a file specified by the given path.
// If the path is prefixed with "file:", it is stripped before reading.
// The function also supports expanding the home directory if the path
// starts with "~/". The trim parameter determines whether to trim
// whitespace from the file contents before returning it as a string.
func fetchFile(_ map[string]any, p string, trim bool) (s string, err error) {
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
