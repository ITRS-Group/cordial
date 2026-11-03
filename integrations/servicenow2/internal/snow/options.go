/*
Copyright © 2025 ITRS Group

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

package snow

import (
	"fmt"
)

type reqOptions struct {
	limit   string
	fields  string
	offset  string
	query   string
	display string
	sysID   string
}

// Option TODO: Describe its purpose and intended use.
type Option func(*reqOptions)

func evalReqOptions(options ...Option) (opts *reqOptions) {
	opts = &reqOptions{}
	for _, r := range options {
		r(opts)
	}
	return
}

// Limit TODO: Document its behavior, inputs, and effects.
func Limit(limit int) Option {
	return func(ro *reqOptions) {
		ro.limit = fmt.Sprintf("%d", limit)
	}
}

// Fields TODO: Document its behavior, inputs, and effects.
func Fields(fields string) Option {
	return func(ro *reqOptions) {
		ro.fields = fields
	}
}

// Offset TODO: Document its behavior, inputs, and effects.
func Offset(offset string) Option {
	return func(ro *reqOptions) {
		ro.offset = offset
	}
}

// Query TODO: Document its behavior, inputs, and effects.
func Query(query string) Option {
	return func(ro *reqOptions) {
		ro.query = query
	}
}

// Display sets the `sysparm_display_value` parameter
func Display(display string) Option {
	return func(ro *reqOptions) {
		ro.display = display
	}
}

// SysID TODO: Document its behavior, inputs, and effects.
func SysID(sysID string) Option {
	return func(ro *reqOptions) {
		ro.sysID = sysID
	}
}
