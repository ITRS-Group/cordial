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

package config

import (
	"maps"
	"path"
	"slices"
	"strings"
)

// Credentials handling functions

// credentialsOptions returns options with defaults for credentials
// files first, so that anything in options can override them.
func credentialsOptions(options []FileOption) []FileOption {
	return append([]FileOption{KeyDelimiter("::")}, options...)
}

// credentialsWriteOptions is like credentialsOptions but also restricts
// the permissions of any file and directories created.
func credentialsWriteOptions(options []FileOption) []FileOption {
	return append([]FileOption{FileMode(0600), DirMode(0700)}, credentialsOptions(options)...)
}

// Credentials can carry a number of different credential types. Add
// more as required.
type Credentials struct {
	Domain       string `json:"domain,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Token        string `json:"token,omitempty"`
	Renewal      string `json:"renewal,omitempty"`
}

// FindCreds finds a set of credentials in the given config under the
// key "credentials" and returns the longest case-insensitive match, if
// any. A domain in the credentials file matches p if it either matches
// the whole of p as a shell pattern, as per `path.Match()`, or is a
// substring of p, so that a plain domain name matches any URL for
// that domain. creds is nil if no matching credentials found.
func (c *Config) FindCreds(p string, options ...FileOption) (creds *Config) {
	if c == nil {
		return nil
	}

	cr := Get[map[string]any](c, "credentials")
	if cr == nil {
		return
	}

	domains := slices.SortedFunc(maps.Keys(cr), func(i, j string) int {
		return len(j) - len(i)
	})

	p = strings.ToLower(p)
	creds = New(options...)
	for _, domain := range domains {
		d := strings.ToLower(domain)
		if matched, _ := path.Match(d, p); matched || strings.Contains(p, d) {
			creds.MergeConfigMap(Get[map[string]any](c, c.Join("credentials", domain)))
			return
		}
	}
	return nil
}

// FindCreds looks for matching credentials in a default "credentials"
// file. Options are the same as for [Read] but the default KeyDelimiter
// is set to "::" as credential domains are likely to be hostnames or
// URLs, which can be overridden by passing [KeyDelimiter]. The longest
// match wins.
func FindCreds(p string, options ...FileOption) (creds *Config) {
	options = credentialsOptions(options)
	cf, err := Read("credentials", options...)
	if err != nil {
		return nil
	}
	return cf.FindCreds(p, options...)
}

// AddCreds adds credentials to the "credentials" file identified by the
// options. creds.Domain is used as the key for matching later on. Any
// existing credential with the same Domain is overwritten. If there is
// an error un the underlying routines it is returned without change.
//
// The file is written so that only the user can read it, unless
// overridden using [FileMode] and [DirMode].
func AddCreds(creds Credentials, options ...FileOption) (err error) {
	options = credentialsOptions(options)
	cf, err := Read("credentials", options...)
	if err != nil {
		return
	}
	Set(cf, cf.Join("credentials", creds.Domain), creds)
	return cf.Write("credentials", credentialsWriteOptions(options)...)
}

// DeleteCreds removes the entry for domain from the credentials file
// identified by options.
func DeleteCreds(domain string, options ...FileOption) (err error) {
	options = credentialsOptions(options)
	cf, err := Read("credentials", options...)
	if err != nil {
		return
	}
	Delete(cf, cf.Join("credentials", domain))
	return cf.Write("credentials", credentialsWriteOptions(options)...)
}

// DeleteAllCreds will remove all the credentials in the credentials
// file identified by options.
func DeleteAllCreds(options ...FileOption) (err error) {
	options = credentialsOptions(options)
	cf := New(options...)
	Set(cf, "credentials", &Credentials{}, NoExpand())
	return cf.Write("credentials", credentialsWriteOptions(options)...)
}
