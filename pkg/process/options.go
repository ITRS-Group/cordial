/*
Copyright © 2023 ITRS Group

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
