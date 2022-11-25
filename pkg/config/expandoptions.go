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
	lookupTables      []map[string]string
	funcMaps          map[string]func(string) string
	noDefaultFuncMaps bool
}

type ExpandOptions func(*expandOptions)

func evalExpandOptions(options ...ExpandOptions) (e *expandOptions) {
	e = &expandOptions{}
	defaultFuncMaps := map[string]func(string) string{
		"http":  fetchURL,
		"https": fetchURL,
		"file":  fetchFile,
	}
	for _, opt := range options {
		opt(e)
	}
	if !e.noDefaultFuncMaps {
		for k, v := range e.funcMaps {
			defaultFuncMaps[k] = v
		}
		e.funcMaps = defaultFuncMaps
	}
	return
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

// ExpandFunc defines a custom prefix to function mapping for expansion.
// If the configuration prefix matches the one set then the function is
// called with the contents of the expansion including the prefix (for
// URLs) but stripped of the opening `${` and the closing `}`
func ExpandFunc(prefix string, fn func(string) string) ExpandOptions {
	return func(e *expandOptions) {
		e.funcMaps[prefix] = fn
	}
}

// NoExternalLookups disables the built-in expansion options that fetch
// data from outside the program, such as URLs and file paths.
func NoExternalLookups() ExpandOptions {
	return func(e *expandOptions) {
		e.noDefaultFuncMaps = true
	}
}
