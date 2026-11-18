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
	"maps"
	"slices"
	"sync"
)

type fnmap struct {
	fn   func(configItems map[string]any, name string, trim bool, data ...any) (string, error)
	data []any
	nest bool // the result of fn is itself expanded, subject to the Loop limit
}

type expandOptions struct {
	cf                 *Config             // the Config object is included in the options to allow for options that need to access configuration settings, such as decoder configs
	defaultValue       any                 // the default value to be used if no other value is found during expansion
	expandNonString    bool                // whether to attempt expansion on non-string values, such as integers or booleans
	expandNonStringCSV bool                // whether to attempt expansion on non-string values that are represented as comma-separated values (CSV)
	expressions        bool                // whether to enable expression evaluation during expansion
	externalFuncMaps   bool                // whether to include the default external function maps (http, https, file) during expansion
	funcMaps           map[string]fnmap    // the set of function maps available for expansion, keyed by function name
	initialValue       any                 // the initial value to be used for expansion before any other processing
	lookupTables       []map[string]string // a list of lookup tables to be used during expansion, in order of precedence
	loop               int                 // the maximum depth of nested expansion, to prevent infinite loops, default 10
	noDecode           bool                // whether to skip decoding of values during expansion
	noExpand           bool                // whether to skip expansion entirely
	promoteFrom        []string            // if not set, look for the first legacy parameter and promote if found (i.e. save to new name and delete old name)
	replacements       []string            // a list of string replacements to apply during expansion
	// TODO: Revisit unused field trimPrefix; restore it only when needed.
	// 	trimPrefix         bool                // whether to trim the prefix from the expanded value
	trimSpace  bool   // whether to trim leading and trailing whitespace from the expanded value
	useKeyFile string // the path to a key file to be used for decryption during expansion
}

// ExpandOption control the way configuration options undergo string
// expansion through the underlying [ExpandString] functions.
// ExpandOption can be passed to any of the normal lookup functions
// that are provided to override [viper] versions, such as [GetString].
//
// e.g.
//
//	s := config.GetString("config.value", ExternalLookups(false), LookupTable(configMap), Prefix("myconf", myFunc))
type ExpandOption func(*expandOptions)

// builtinFuncMaps are the prefixes enabled by default, until disabled
// with ExternalLookups(false). The values returned are never expanded
// further. It is never modified and the mutex is only to protect against
// future changes.
var defaultFuncMapsMutex sync.RWMutex
var defaultFuncMaps = map[string]fnmap{
	"http":  {fn: fetchURL},
	"https": {fn: fetchURL},
	"file":  {fn: fetchFile},
}

// evalExpandOptions returns the expand options for c by applying, in
// order, the default options set on c and then options. The returned
// options, including the function map, belong to the caller and
// nothing from them is ever shared with another Config or call.
func evalExpandOptions(c *Config, options ...ExpandOption) (e *expandOptions) {
	e = &expandOptions{
		cf:               c, // default to config c if not included in options
		externalFuncMaps: true,
		funcMaps:         map[string]fnmap{}, // only the prefixes added by options
		loop:             10,                 // default maximum depth of nested expansion
		replacements:     []string{},
		trimSpace:        true,
	}

	for _, opt := range c.defaultExpandOptions {
		opt(e)
	}

	for _, opt := range options {
		opt(e)
	}

	// build the final set of functions: the built-in ones (unless
	// disabled) overridden by any added by options
	funcs := make(map[string]fnmap, len(e.funcMaps)+len(defaultFuncMaps)+1)
	if e.externalFuncMaps {
		defaultFuncMapsMutex.RLock()
		maps.Copy(funcs, defaultFuncMaps)
		defaultFuncMapsMutex.RUnlock()
	}
	maps.Copy(funcs, e.funcMaps)
	if e.expressions {
		funcs["expr"] = fnmap{fn: expr}
	}
	e.funcMaps = funcs

	// a depth of less than one is meaningless, always do one pass
	e.loop = max(e.loop, 1)

	if e.defaultValue == nil {
		e.defaultValue = ""
	}

	return
}

// DefaultExpandOptions returns the default set of ExpandOptions. This
// allows duplication to another Config object.
func (c *Config) DefaultExpandOptions() (options []ExpandOption) {
	c.rwmutex.RLock()
	defer c.rwmutex.RUnlock()
	return slices.Clone(c.defaultExpandOptions)
}

// SetDefaultExpandOptions sets defaults to all subsequent calls to
// functions that perform configuration expansion. These defaults can be
// reset by calling SetDefaultExpandOptions with no arguments.
//
// The options only apply to c; any [Prefix] functions given are never
// visible to any other Config.
func (c *Config) SetDefaultExpandOptions(options ...ExpandOption) {
	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()
	c.defaultExpandOptions = slices.Clone(options)
}

// IncludeConfig copies a point to the Config object cf into the
// options. This should not be used normally, but is required to
// bootstrap some expand options which do not have access to the
// original Config object, such as the built-in expression evaluation
// option. The default config object is set to the global configuration.
func IncludeConfig(cf *Config) ExpandOption {
	return func(e *expandOptions) {
		e.cf = cf
	}
}

// NoExpand overrides all other options except DefaultValue() and
// InitialValue() and returns the value as-is with no expansion applied.
// This is to allow the normal functions and methods to be called but to
// receive the underlying configuration item, such as an encoded
// password.
func NoExpand() ExpandOption {
	return func(e *expandOptions) {
		e.noExpand = true
	}
}

// NoDecode disables the expansion of encoded (`enc:`) values. Use this
// option if you want the expanded value of the configuration item but
// without decoding sensitive data such as secrets and passwords.
func NoDecode(n bool) ExpandOption {
	return func(e *expandOptions) {
		e.noDecode = n
	}
}

// LookupTable adds lookup maps, of name/value pairs, to the Expand
// functions. When string expansion is done to a name, i.e. `${name}`,
// then the name is first looked-up in each map, in the order the
// LookupTable options are given, and the first match, if any, wins.
// This applies to any name, including those that look like a
// configuration key or a file path. If there is no match in any of the
// lookup maps then a plain name, without a prefix, is treated as an
// environment variable and any other name is handled as per the usual
// rules.
//
// The values found in lookup tables are inserted exactly as they are,
// after any trimming of white space, and are never themselves expanded.
// This means that it is safe to use data from an untrusted source, such
// as a network request, in a lookup table.
func LookupTable(values ...map[string]string) ExpandOption {
	return func(e *expandOptions) {
		e.lookupTables = append(e.lookupTables, values...)
	}
}

// LookupTables sets the expansion lookup tables to the slice of maps
// passed as values. Any existing lookup tables are discarded.
//
// Deprecated: Use the singular LookupTable with a variadic list of
// tables instead.
func LookupTables(values []map[string]string) ExpandOption {
	return func(e *expandOptions) {
		e.lookupTables = values
	}
}

// Prefix defines a custom mapping for the given prefix to an
// expand-like function. The prefix should not include the terminating
// ":". If the configuration prefix matches during expansion then the
// function is called with the config data as a map[string]any, and the
// contents of the expansion including the prefix (for URLs) but
// stripped of the opening `${` and closing `}`. A boolean parameter
// trims white space from the final result if true.
//
// The result of the function is itself expanded, as a configuration
// value would be, subject to the limit set by [Loop]. This allows a
// function to return a string that refers to other configuration items.
// If the result may contain data that must not be expanded, such as
// content from a network request, use [PrefixVerbatim] instead.
//
// The function MUST NOT call other configuration functions on the
// Config being expanded or lock the configuration mutex as the calling
// function will have already done so. The function should return an
// error if the prefix matches but the value is invalid for any reason,
// such as a network error when fetching a URL or a missing file when
// fetching from the file system.
//
// A prefix, including one that replaces a built-in one such as `file`,
// only applies to the single call or Config that the option is passed
// to and is never visible to any other.
func Prefix(prefix string, fn func(configItems map[string]any, name string, trim bool, data ...any) (string, error), data ...any) ExpandOption {
	return func(e *expandOptions) {
		e.funcMaps[prefix] = fnmap{fn: fn, data: data, nest: true}
	}
}

// PrefixVerbatim is like [Prefix] except that the result of the
// function is inserted exactly as returned and is never expanded
// further. Use this for functions that return data that is not under
// the control of the author of the configuration.
func PrefixVerbatim(prefix string, fn func(configItems map[string]any, name string, trim bool, data ...any) (string, error), data ...any) ExpandOption {
	return func(e *expandOptions) {
		e.funcMaps[prefix] = fnmap{fn: fn, data: data}
	}
}

// ExternalLookups enables or disables the built-in expansion options
// that fetch data from outside the program, such as URLs and file
// paths. The default is true.
func ExternalLookups(yes bool) ExpandOption {
	return func(e *expandOptions) {
		e.externalFuncMaps = yes
	}
}

// Expressions enables or disables the built-in expansion for
// expressions via the `github.com/maja42/goval` package. The default is
// false.
func Expressions(yes bool) ExpandOption {
	return func(e *expandOptions) {
		e.expressions = yes
	}
}

// TrimSpace enables the removal of leading and trailing spaces on all
// values in an expansion. The default is `true`. If a default
// value is given using the Default() then this is never trimmed.
func TrimSpace(trim bool) ExpandOption {
	return func(e *expandOptions) {
		e.trimSpace = trim
	}
}

// DefaultValue sets a default value to be returned if the resulting
// expansion of the config key is the zero value for the type (after any
// optional trimming of leading and trailing spaces).
//
// The type of the default value must be the same as the type of the
// configuration item being used, otherwise the default value is
// ignored. e.g.
//
//	config.Get[uint16]("config.value", config.DefaultValue(uint16(1234)))
//
// The default value is returned as-is and is never expanded.
func DefaultValue(value any) ExpandOption {
	return func(e *expandOptions) {
		e.defaultValue = value
	}
}

// InitialValue sets an initial value to be used if the configuration
// item is empty (or nil) to start. This differs from DefaultValue() which
// supplies a value to use if the value if empty after expansion. The
// initial value, if used, is expanded as would any configuration value
// (when it is a string or []string, unless NoExpand() is used).
//
// If config.NoExpand() is also used then this initial value is used as
// a secondary default - i.e. if config.DefaultValue() is empty.
func InitialValue(value any) ExpandOption {
	return func(e *expandOptions) {
		e.initialValue = value
	}
}

// Replace is used by config.Set* (except config.Set itself) functions
// to replace substrings with the formatted configuration item given as
// name with an equivalent expand string, where the value of the name
// key is only tested as Set time.
//
// e.g. if ${config:home} is "/home/user" then:
//
//	config.SetString("path", "/home/user/file.txt", config.Replace("home"))
//
// results in path being set to "${config:home}/file.txt" for future expansion,
// as "home" may change
//
// Replace can be used multiple times, each name being checked in order.
//
// Expand strings in the value are never substituted.
//
// name is not checked for self-referencing
func Replace(name string) ExpandOption {
	return func(e *expandOptions) {
		e.replacements = append(e.replacements, name)
	}
}

// UseKeyfile overrides the path to the embedded keyfile in the
// `${enc:/path:xxx}` value. This can be useful when the keyfile is
// placed in an alternative location.
func UseKeyfile(file string) ExpandOption {
	return func(eo *expandOptions) {
		eo.useKeyFile = file
	}
}

// ExpandNonStringToCSV causes any non-string configuration item
// expansions of lists to return a comma-separated list of any strings
// in that list. Non-string values are skipped.
func ExpandNonStringToCSV() ExpandOption {
	return func(eo *expandOptions) {
		eo.expandNonString = true
		eo.expandNonStringCSV = true
	}
}

// ExpandNonStringToJSON causes any non-string configuration expansions
// to be returned as a JSON encoded string of the item
func ExpandNonStringToJSON() ExpandOption {
	return func(eo *expandOptions) {
		eo.expandNonString = true
	}
}

// PromoteFrom sets a list of legacy configuration names to check for
// and promote to the new name. If the primary name is found then these
// legacy names are ignored, otherwise the first legacy name found is
// promoted to the new name and the legacy name is deleted. This is
// useful for renaming configuration items without breaking existing
// configurations.
//
// This option is only valid for Get() function and is ignored for
// Lookup() and Set() functions.
func PromoteFrom(names ...string) ExpandOption {
	return func(eo *expandOptions) {
		eo.promoteFrom = names
	}
}

// Loop sets the maximum depth of nested expansion, to prevent infinite
// loops when configuration values refer to each other. The string
// being expanded is level 1 and the value of each configuration item
// (or [Prefix] function result) referred to in it is level 2, and so
// on. References found at the last level are replaced by their values,
// which are then not expanded further. A cycle of configuration items
// is also stopped, without waiting for the limit. The default is 10.
//
// A value of 1, or any value less than 1, means that there is no
// nested expansion at all.
func Loop(max int) ExpandOption {
	return func(eo *expandOptions) {
		eo.loop = max
	}
}
