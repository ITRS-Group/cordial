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

// package ims provides basic support for Incident Management System
// (IMS) integration, such as ServiceDesk Plus, ServiceNow, etc.
package ims

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"iter"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

type ContextKey string

const (
	ContextKeyConfig   ContextKey = "config"   // Context key for passing configuration to handlers, type is *config.Config
	ContextKeyResponse ContextKey = "response" // Context key for passing response to handlers, type is *ims.Response
)

const (
	PROFILE              = "__itrs_profile"
	INCIDENT_UPDATE_ONLY = "__incident_update_only"
	INCIDENT_CORRELATION = "__incident_correlation"
)

// Values is a simple map of string key/value pairs that can be used to
// represent incident fields and values. This is used throughout the IMS
// package to represent the fields and values of an incident, as well as
// the configuration for a variety of operations.
type Values map[string]string

type ClientConfig struct {
	URL     string        `json:"url,omitempty"`
	Token   string        `json:"token,omitempty"`
	Timeout time.Duration `json:"timeout,omitzero"`

	TLS struct {
		SkipVerify bool   `json:"skip-verify,omitzero"`
		Chain      []byte `json:"chain,omitempty"`
	} `json:"tls"`

	Trace bool `json:"trace,omitempty"`
}

// NewClient creates a new rest.Client for the given URL and
// configuration. The client is NOT cached as each execution is a single
// request to the first remote proxy that responds.
func NewClient(cf *ClientConfig) *rest.Client {
	var tcc *tls.Config

	if strings.HasPrefix(cf.URL, "https:") {
		skip := cf.TLS.SkipVerify
		roots, err := x509.SystemCertPool()
		if err != nil {
			log.Warn().Err(err).Msg("cannot read system certificates, continuing anyway")
		}

		if !skip {
			if chain := cf.TLS.Chain; len(chain) != 0 {
				if ok := roots.AppendCertsFromPEM(chain); !ok {
					log.Warn().Msg("error reading cert chain")
				}
			}
		}

		tcc = &tls.Config{
			RootCAs:            roots,
			InsecureSkipVerify: skip,
		}
	}

	timeout := cf.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// use most of the default transport settings
	hc := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tcc,
		},
		Timeout: timeout,
	}

	if cf.Trace {
		hc.Transport = &LogTransport{
			Transport: hc.Transport.(*http.Transport),
		}
	}

	return rest.NewClient(
		rest.BaseURLString(cf.URL),
		rest.HTTPClient(hc),
		rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ []byte) {
			req.Header.Add(
				"Authorization",
				fmt.Sprintf("Bearer %s", cf.Token),
			)
		}),
	)
}

// Connect returns a sequence of *ClientConfig for each URL in the
// configuration. The caller can use this to attempt to connect to each
// configured URL in turn until a successful connection is made.
func Connect(imsCf *config.Config, imsType string) iter.Seq[*rest.Client] {
	return func(yield func(*rest.Client) bool) {
		for _, r := range config.Get[[]string](imsCf, "url") {
			ccf := &ClientConfig{
				URL:     r + "/" + imsType,
				Token:   config.Get[string](imsCf, config.Join("authentication", "token")),
				Timeout: config.Get[time.Duration](imsCf, config.Join("timeout")),
			}
			ccf.TLS.SkipVerify = config.Get[bool](imsCf, config.Join("tls", "skip-verify"))
			ccf.TLS.Chain = config.Get[[]byte](imsCf, config.Join("tls", "chain"))
			ccf.Trace = config.Get[bool](imsCf, config.Join("trace"))

			if !yield(NewClient(ccf)) {
				return
			}
		}
	}
}

// CorrelationID generates a correlation ID for the given data. The
// algorithm used, SHA1, does not have to be cryptographically secure,
// but should produce a reasonably unique ID for the given data. The
// same data should produce the same correlation ID, and different data
// should produce different correlation IDs with a very high
// probability. The correlation ID is returned as a hexadecimal string.
func CorrelationID(data string) string {
	return fmt.Sprintf("%X", sha1.Sum([]byte(data)))
}

// debug transport for tracing

type LogTransport struct {
	Transport http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Printf("Error dumping request: %v", err)
	} else {
		log.Printf("REQUEST:\n%s", reqDump)
	}

	// Perform the actual request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Log the response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Printf("Error dumping response: %v", err)
	} else {
		log.Printf("RESPONSE:\n%s", respDump)
	}

	return resp, nil
}
