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

package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
)

// expander holds the state for one top-level expansion of a string.
//
// Expansion is nested rather than repeated: each `${name}` in a string
// is resolved once and, if the source of the value is one that is
// documented as being expandable (configuration values and the results
// of functions registered with [Prefix]) then the value is itself
// expanded, to a depth limited by the [Loop] option. All other sources
// (lookup tables, environment variables, files, URLs and expressions)
// are inserted verbatim, so that data from them can never inject
// further expansions.
type expander struct {
	c       *Config
	opts    *expandOptions
	nesting []string // configuration keys currently being expanded, to stop cycles
}

// resolved is the result of resolving a single `${name}`
type resolved struct {
	value string
	nest  bool   // the value should be expanded in turn
	key   string // the configuration key the value came from, if any
}

// expandString is the core of [Expand]. All of the other expansion
// functions are built on top of this one.
func expandString(c *Config, opts *expandOptions, input string) string {
	if opts.noExpand {
		if input != "" {
			return input
		}

		if !isZero(opts.initialValue) {
			return fmt.Sprint(opts.initialValue)
		}

		if !isZero(opts.defaultValue) {
			return fmt.Sprint(opts.defaultValue)
		}

		return ""
	}

	if input == "" && !isZero(opts.initialValue) {
		input = fmt.Sprint(opts.initialValue)
	}

	x := &expander{c: c, opts: opts}
	value := x.expand(input, 1)

	if opts.trimSpace {
		value = strings.TrimSpace(value)
	}

	if value == "" && !isZero(opts.defaultValue) {
		value = fmt.Sprint(opts.defaultValue)
	}

	return value
}

// expand replaces all the `${...}` references in s. depth is the nesting
// level of s, with the original input being level 1.
func (x *expander) expand(s string, depth int) string {
	return expandReferences(s, func(name string) string {
		return x.resolve(name, depth)
	})
}

func (x *expander) resolve(name string, depth int) string {
	if encoded, ok := strings.CutPrefix(name, "enc:"); ok {
		if x.opts.noDecode {
			// return original string after restoring surrounding
			// ${...}
			return "${" + name + "}"
		}
		return x.decode(encoded, depth)
	}

	r, _ := x.lookup(name)
	return x.nested(r, depth)
}

// nested expands the value of r, if it is allowed to be, otherwise
// returns it as-is. Values at the maximum depth and values that refer
// back to a configuration key already being expanded are not expanded
// further.
func (x *expander) nested(r resolved, depth int) string {
	if !r.nest || depth >= x.opts.loop || (r.key != "" && slices.Contains(x.nesting, r.key)) {
		return r.value
	}

	if r.key != "" {
		x.nesting = append(x.nesting, r.key)
		defer func() { x.nesting = x.nesting[:len(x.nesting)-1] }()
	}

	value := x.expand(r.value, depth+1)
	if x.opts.trimSpace {
		value = strings.TrimSpace(value)
	}
	return value
}

// decode accepts input of the form:
//
//	keyfile[|keyfile...]:[+encs+HEX|external]
//
// Each keyfile is tried until the first that does not return a decoding
// error. `keyfile` may be prefixed `~/` in which case the file is
// relative to the user's home directory. If the encoded string is
// prefixed with `+encs+` (standard Geneos usage) then it is used
// directly, otherwise the value is looked-up using the normal
// conventions for external access, e.g. file or URL.
//
// The decoded value is returned as-is and is never expanded further.
func (x *expander) decode(s string, depth int) string {
	keyfiles, encodedValue := x.splitEncFields(s)

	if x.opts.useKeyFile != "" {
		keyfiles = x.opts.useKeyFile
	}

	if !strings.HasPrefix(encodedValue, "+encs+") {
		r, _ := x.lookup(encodedValue)
		encodedValue = x.nested(r, depth)
	}

	if len(encodedValue) == 0 {
		return ""
	}

	for k := range strings.SplitSeq(keyfiles, "|") {
		keyfile := KeyFile(ResolveHome(k))
		p, err := keyfile.DecodeString(host.Localhost, encodedValue)
		if err != nil {
			continue
		}
		return p
	}
	return ""
}

// splitEncFields breaks the string enc into two strings, the first the
// keyfile(s) and the second the ciphertext or a reference to it.
//
// The split is at the first colon that is followed by `+encs+`, a
// `config:` or `env:` prefix, or the prefix of any other registered
// function (e.g. `file:` or `https:`). This allows the reference forms
// in the documentation. Otherwise the split is on the *last* colon and
// not the first, as Windows drive letter paths contain one.
func (x *expander) splitEncFields(enc string) (keyfiles, ciphertext string) {
	for i, r := range enc {
		if r != ':' {
			continue
		}
		rest := enc[i+1:]
		if strings.HasPrefix(rest, "+encs+") || strings.HasPrefix(rest, "config:") || strings.HasPrefix(rest, "env:") {
			return enc[:i], rest
		}
		if prefix, _, found := strings.Cut(rest, ":"); found {
			if _, ok := x.opts.funcMaps[prefix]; ok {
				return enc[:i], rest
			}
		}
	}

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

// lookup resolves name, which is the contents of a `${...}` with the
// braces removed, using the same rules and options as [Expand]. The
// name is looked-up in this order:
//
//  1. an exact match in any of the lookup tables
//  2. as a file path, if the name contains a `/` but not a `:`
//  3. as a configuration key, if prefixed `config:` or containing the
//     key delimiter and no `:`
//  4. as an environment variable if there is no `:` (or it is prefixed
//     `env:`)
//  5. using a function registered for the prefix (up to the first `:`)
//
// With the ExpandNonStringToCSV() or ExpandNonStringToJSON() options,
// if the item to be expanded is a configuration value and the value
// resolved is not a plain string then it is either, for a slice of
// values, returned as a string containing comma-separated strings found
// or as a JSON encoded representation of the value.
func (x *expander) lookup(name string) (r resolved, err error) {
	opts := x.opts
	trim := func(s string) string {
		if opts.trimSpace {
			return strings.TrimSpace(s)
		}
		return s
	}

	for _, table := range opts.lookupTables {
		if v, ok := table[name]; ok {
			return resolved{value: trim(v)}, nil
		}
	}

	switch {
	case strings.Contains(name, "/") && !strings.Contains(name, ":"):
		// if the string looks like a file path and there is a "file"
		// function registered then try to read the file and return its
		// contents.
		if fm, ok := opts.funcMaps["file"]; ok {
			return x.call(fm, name)
		}

	case strings.HasPrefix(name, "config:"), !strings.Contains(name, ":"):
		// if the string is a config reference or contains a delimiter,
		// implying a config item, then try to resolve it
		if strings.HasPrefix(name, "config:") || (x.c.delimiter != "" && strings.Contains(name, x.c.delimiter)) {
			return x.configValue(strings.TrimPrefix(name, "config:"))
		}

		return resolved{value: trim(mapEnv(name))}, nil

	case strings.HasPrefix(name, "env:"):
		return resolved{value: trim(mapEnv(strings.TrimPrefix(name, "env:")))}, nil

	default:
		// check for any registered functions and call that with the
		// whole of the config string. there must be a ":" here, else
		// the above test would have picked it up. it is up to the
		// function called to trim whitespace, if required.
		if fname, _, found := strings.Cut(name, ":"); found {
			if fm, ok := opts.funcMaps[fname]; ok {
				return x.call(fm, name)
			}
		}
	}

	return
}

func (x *expander) call(fm fnmap, name string) (r resolved, err error) {
	r.value, err = fm.fn(x.c.allSettings(), name, x.opts.trimSpace, fm.data...)
	r.nest = fm.nest
	// the result of a call that refers back to the same call, directly or
	// through other values, is not expanded again. The key cannot be a
	// configuration key.
	if r.nest {
		r.key = "\x00" + name
	}
	return
}

// configValue returns the value for the configuration key, converting
// non-string values according to the options
func (x *expander) configValue(key string) (r resolved, err error) {
	opts := x.opts
	r.nest = true
	r.key = key

	if !opts.expandNonString {
		// the value is a string, any further expansion is done by the
		// caller, subject to the loop limit
		r.value = x.c.config.GetString(key)
		if opts.trimSpace {
			r.value = strings.TrimSpace(r.value)
		}
		return
	}

	switch w := x.c.config.Get(key).(type) {
	case deletedKey:
		r.value = ""
	case string:
		// strings still get returned "as-is"
		r.value = w
		if opts.trimSpace {
			r.value = strings.TrimSpace(r.value)
		}
	case []any:
		if opts.expandNonStringCSV {
			var u []string
			for _, i := range w {
				if s, ok := i.(string); ok {
					u = append(u, s)
				}
			}
			r.value = strings.Join(u, ",")
			return
		}
		// type switches do not support fallthrough
		var u []byte
		if u, err = json.Marshal(w); err != nil {
			return resolved{}, err
		}
		r.value = string(u)
	default:
		// if caller has asked for CSV then do not return JSON
		if opts.expandNonStringCSV {
			return
		}
		var u []byte
		if u, err = json.Marshal(w); err != nil {
			return resolved{}, err
		}
		r.value = string(u)
	}

	return
}

// expandReferences replaces each `${name}` in s with the result of
// calling mapping(name). The text `$${` is an escape for a literal `${`
// and the remainder of the reference is copied as-is.
func expandReferences(s string, mapping func(string) string) string {
	var buf []byte
	// `${...}`` is all ASCII, so bytes are fine for this operation.
	i := 0

	for j := 0; j < len(s); j++ {
		if s[j] != '$' || j+1 >= len(s) {
			continue
		}

		if buf == nil {
			buf = make([]byte, 0, 2*len(s))
		}
		buf = append(buf, s[i:j]...)

		if s[j+1] == '$' && j+2 < len(s) && s[j+2] == '{' {
			// escaped, keep one `$` and skip the brace
			buf = append(buf, '$', '{')
			j += 2
			i = j + 1
			continue
		}

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
			buf = append(buf, mapping(name)...)
		}
		j += w
		i = j + 1
	}

	if buf == nil {
		return s
	}

	return string(append(buf, s[i:]...))
}
