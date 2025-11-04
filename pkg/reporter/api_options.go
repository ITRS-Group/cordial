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
	ReporterCommon
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

func evalAPIOptions(options ...APIReporterOptions) (fro *apiReportOptions) {
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

type APIReporterOptions func(*apiReportOptions)

func APIHostname(hostname string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.hostname = hostname
	}
}

func APIPort(port int) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.port = port
	}
}

func APISecure(secure bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.secure = secure
	}
}

func APISkipVerify(skip bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.skipVerify = skip
	}
}

func APIEntity(entity string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.entity = entity
	}
}

func APISampler(sampler string) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.sampler = sampler
	}
}

func DataviewCreateDelay(delay time.Duration) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.dvCreateDelay = delay
	}
}

func ResetDataviews(reset bool) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.reset = reset
	}
}

func APIMaxRows(n int) APIReporterOptions {
	return func(aro *apiReportOptions) {
		aro.maxrows = n
	}
}
