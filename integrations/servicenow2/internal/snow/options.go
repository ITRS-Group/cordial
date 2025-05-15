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
	limit  string
	fields string
	offset string
	query  string
	sysID  string
}

type Options func(*reqOptions)

func evalReqOptions(options ...Options) (opts *reqOptions) {
	opts = &reqOptions{}
	for _, r := range options {
		r(opts)
	}
	return
}

func Limit(limit int) Options {
	return func(ro *reqOptions) {
		ro.limit = fmt.Sprintf("%d", limit)
	}
}

func Fields(fields string) Options {
	return func(ro *reqOptions) {
		ro.fields = fields
	}
}

func Offset(offset string) Options {
	return func(ro *reqOptions) {
		ro.offset = offset
	}
}

func Query(query string) Options {
	return func(ro *reqOptions) {
		ro.query = query
	}
}

func SysID(sysID string) Options {
	return func(ro *reqOptions) {
		ro.sysID = sysID
	}
}
