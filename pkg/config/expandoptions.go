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

type expandOptions struct {
	lookupTables     []map[string]string
	funcMaps         map[string]func(*Config, string) string
	externalFuncMaps bool
	trimPrefix       bool
}

type ExpandOptions func(*expandOptions)

func evalExpandOptions(c *Config, options ...ExpandOptions) (e *expandOptions) {
	e = &expandOptions{}
	defaultFuncMaps := map[string]func(*Config, string) string{
		"http":  fetchURL,
		"https": fetchURL,
		"file":  fetchFile,
	}
	e.funcMaps = map[string]func(*Config, string) string{}
	e.externalFuncMaps = true
	for _, opt := range c.defaultExpandOptions {
		opt(e)
	}
	for _, opt := range options {
		opt(e)
	}
	if e.externalFuncMaps {
		for k, v := range e.funcMaps {
			defaultFuncMaps[k] = v
		}
		e.funcMaps = defaultFuncMaps
	}
	return
}

// DefaultExpandOptions sets defaults to all subsequent calls to
// functions that perform configuration expansion. These defaults can be
// reset by calling DefaultExpandOptions with no arguments.
func (c *Config) DefaultExpandOptions(options ...ExpandOptions) {
	c.defaultExpandOptions = options
}

// LookupTable adds a lookup map to the Expand functions. When string
// expansion is done to an plain word, e.g. `${item}`, then item is
// checked in any tables passed, in order defined and first match wins,
// to the function. If there are no maps defined then `item` is looked
// up as an environment variable.
func LookupTable(values map[string]string) ExpandOptions {
	return func(e *expandOptions) {
		e.lookupTables = append(e.lookupTables, values)
	}
}

// Prefix defines a custom mapping for the given prefix to an
// expand-like function. The prefix should not include the terminating
// ":". If the configuration prefix matches during expansion then the
// function is called with the config data and the contents of the
// expansion including the prefix (for URLs) but stripped of the opening
// `${` and the closing `}`
func Prefix(prefix string, fn func(*Config, string) string) ExpandOptions {
	return func(e *expandOptions) {
		e.funcMaps[prefix] = fn
	}
}

// ExternalLookups enables or disables the built-in expansion options
// that fetch data from outside the program, such as URLs and file
// paths. The default is true.
func ExternalLookups(yes bool) ExpandOptions {
	return func(e *expandOptions) {
		e.externalFuncMaps = yes
	}
}

// TrimPrefix enables the removal of the prefix from the string passed
// to expansion functions. If this is not set then URLs can be passed
// as-is since the prefix is part of the URL. If set then URLs would
// need the schema explicitly added after the prefix. Using this option
// allows standard function like [strings.ToUpper] to be used without
// additional wrappers.
func TrimPrefix() ExpandOptions {
	return func(e *expandOptions) {
		e.trimPrefix = true
	}
}
