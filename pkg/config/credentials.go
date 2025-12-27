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
	"slices"
	"strings"
)

// Credentials handling functions

// Credentials can carry a number of different credential types. Add
// more as required. Eventually this will go into memguard.
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
// any. The domains in the credentials file can use shell patterns, as
// per `file.Match()`. creds is nil if no matching credentials found.
func (c *Config) FindCreds(p string) (creds *Config) {
	if c == nil {
		return nil
	}

	cr := c.GetStringMap("credentials")
	if cr == nil {
		return
	}

	domains := slices.SortedFunc(maps.Keys(cr), func(i, j string) int {
		return len(j) - len(i)
	})

	creds = New()
	for _, domain := range domains {
		if strings.Contains(strings.ToLower(p), strings.ToLower(domain)) {
			creds.MergeConfigMap(c.GetStringMap(c.Join("credentials", domain)))
			return
		}
	}
	return nil
}

// FindCreds looks for matching credentials in a default "credentials"
// file. Options are the same as for [Load] but the default KeyDelimiter
// is set to "::" as credential domains are likely to be hostnames or
// URLs. The longest match wins.
func FindCreds(p string, options ...FileOptions) (creds *Config) {
	options = append(options, KeyDelimiter("::"))
	cf, err := Load("credentials", options...)
	if err != nil {
		return nil
	}
	return cf.FindCreds(p)
}

// AddCreds adds credentials to the "credentials" file identified by the
// options. creds.Domain is used as the key for matching later on. Any
// existing credential with the same Domain is overwritten. If there is
// an error un the underlying routines it is returned without change.
func AddCreds(creds Credentials, options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf, err := Load("credentials", options...)
	if err != nil {
		return
	}
	cf.Set(cf.Join("credentials", creds.Domain), creds)
	return cf.Save("credentials", options...)
}

// DeleteCreds removes the entry for domain from the credentials file
// identified by options.
func DeleteCreds(domain string, options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf, err := Load("credentials", options...)
	if err != nil {
		return
	}
	credmap := cf.GetStringMap("credentials")
	delete(credmap, domain)
	cf.Set("credentials", credmap)
	return cf.Save("credentials", options...)
}

// DeleteAllCreds will remove all the credentials in the credentials
// file identified by options.
func DeleteAllCreds(options ...FileOptions) (err error) {
	options = append(options, KeyDelimiter("::"))
	cf := New(options...)
	cf.Set("credentials", &Credentials{})
	return cf.Save("credentials", options...)
}
