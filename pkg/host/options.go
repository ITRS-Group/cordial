/*
Copyright © 2026 ITRS Group

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

package host

type ProcessOptions func(*processOptions)

type processOptions struct {
	errfile    string
	detach     bool
	createCore bool
}

func evalProcessOptions(options ...ProcessOptions) (d *processOptions) {
	// defaults
	d = &processOptions{}
	for _, opt := range options {
		opt(d)
	}
	return
}

func ProcessErrfile(errfile string) ProcessOptions {
	return func(po *processOptions) {
		po.errfile = errfile
	}
}

// ProcessDetach makes the process run detached from the parent
func ProcessDetach() ProcessOptions {
	return func(po *processOptions) {
		po.detach = true
	}
}

func ProcessCreateCore() ProcessOptions {
	return func(po *processOptions) {
		po.createCore = true
	}
}
