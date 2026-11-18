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

package api

import "time"

// DefaultTimeout is how long an [XMLRPCClient] waits for a Netprobe to
// respond to a call unless changed with [Timeout].
const DefaultTimeout = 60 * time.Second

type apiOptions struct {
	insecureSkipVerify bool
	timeout            time.Duration
}

// Option TODO: Describe its purpose and intended use.
type Option func(*apiOptions)

func evalOptions(options ...Option) *apiOptions {
	opts := &apiOptions{timeout: DefaultTimeout}
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

// Timeout sets how long the client waits for the response headers of
// each call. A zero value means no limit.
func Timeout(timeout time.Duration) Option {
	return func(c *apiOptions) {
		c.timeout = timeout
	}
}

// InsecureSkipVerify disables verification of the Netprobe certificate
// for HTTPS endpoints.
func InsecureSkipVerify() Option {
	return func(c *apiOptions) {
		c.insecureSkipVerify = true
	}
}
