/*
Copyright © 2024 ITRS Group

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

package reporter

import (
	"time"
)

type apiReportOptions struct {
	hostname      string
	port          int
	secure        bool
	skipVerify    bool
	entity        string
	sampler       string
	dvCreateDelay time.Duration
	reset         bool
	maxrows       int
}

func evalAPIOptions(options ...APIReporterOption) (fro *apiReportOptions) {
	fro = &apiReportOptions{
		hostname:   "localhost",
		port:       7036,
		secure:     false,
		skipVerify: false,
	}
	for _, opt := range options {
		opt(fro)
	}
	return
}

// APIReporterOption TODO: Describe its purpose and intended use.
type APIReporterOption func(*apiReportOptions)

// APIHostname TODO: Document its behavior, inputs, and effects.
func APIHostname(hostname string) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.hostname = hostname
	}
}

// APIPort TODO: Document its behavior, inputs, and effects.
func APIPort(port int) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.port = port
	}
}

// APISecure TODO: Document its behavior, inputs, and effects.
func APISecure(secure bool) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.secure = secure
	}
}

// APISkipVerify TODO: Document its behavior, inputs, and effects.
func APISkipVerify(skip bool) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.skipVerify = skip
	}
}

// APIEntity TODO: Document its behavior, inputs, and effects.
func APIEntity(entity string) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.entity = entity
	}
}

// APISampler TODO: Document its behavior, inputs, and effects.
func APISampler(sampler string) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.sampler = sampler
	}
}

// DataviewCreateDelay TODO: Document its behavior, inputs, and effects.
func DataviewCreateDelay(delay time.Duration) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.dvCreateDelay = delay
	}
}

// ResetDataviews TODO: Document its behavior, inputs, and effects.
func ResetDataviews(reset bool) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.reset = reset
	}
}

// APIMaxRows TODO: Document its behavior, inputs, and effects.
func APIMaxRows(n int) APIReporterOption {
	return func(aro *apiReportOptions) {
		aro.maxrows = n
	}
}
