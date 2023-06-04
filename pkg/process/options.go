/*
Copyright © 2023 ITRS Group

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

package process

type processOptions struct {
	lookup     []map[string]string
	expandArgs bool
	expandEnv  bool
}

// Options for Start and Batch
type Options func(*processOptions)

func evalOptions(options ...Options) (opts *processOptions) {
	opts = &processOptions{}
	// opts.lookup = []map[string]string{}{}
	for _, o := range options {
		o(opts)
	}
	return
}

// ExpandArgs controls the expansion of the values in the Args slice. If
// this option is used then each element of the Args slice is passed to
// ExpandString with an lookup tables set using LookupTable().
func ExpandArgs() Options {
	return func(po *processOptions) {
		po.expandArgs = true
	}
}

// ExpandEnv controls the expansion of the values in the Env slice. If
// this option is used then each element of the Env slice is passed to
// ExpandString with an lookup tables set using LookupTable().
func ExpandEnv() Options {
	return func(po *processOptions) {
		po.expandEnv = true
	}
}

// LookupTable adds a lookup map (string to string) to the set of lookup
// tables passed to ExpandString.
func LookupTable(table map[string]string) Options {
	return func(po *processOptions) {
		po.lookup = append(po.lookup, table)
	}
}
