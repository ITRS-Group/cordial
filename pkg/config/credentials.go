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

package config

import (
	"sort"
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
// key "credentials" and returns the longest match, if any. creds
// is nil if no matching credentials found.
func (cf *Config) FindCreds(p string) (creds *Config) {
	cr := cf.GetStringMap("credentials")
	if cr == nil {
		return
	}
	domains := []string{}
	for k := range cr {
		domains = append(domains, k)
	}
	// sort the paths longest to shortest
	sort.Slice(domains, func(i, j int) bool {
		return len(domains[i]) > len(domains[j])
	})
	creds = New()
	for _, domain := range domains {
		if strings.Contains(strings.ToLower(p), strings.ToLower(domain)) {
			creds.MergeConfigMap(cf.GetStringMap(cf.Join("credentials", domain)))
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
	cf, _ := Load("credentials", options...)
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
