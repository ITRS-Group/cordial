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
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

var log = cordial.Logger

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

// ClientConfig represents the configuration for creating a new IMS
// client.
type ClientConfig struct {
	URL     string        `json:"url,omitempty"`
	Token   string        `json:"token,omitempty"`
	Timeout time.Duration `json:"timeout,omitzero"`

	TLS struct {
		SkipVerify bool   `json:"skip-verify,omitzero"`
		CACerts    []byte `json:"ca-certs,omitempty"`
	} `json:"tls"`

	Trace bool `json:"trace,omitempty"`
}

// NewClient creates a new rest.Client for the given URL and
// configuration. The client is NOT cached as each execution is a single
// request to the first remote proxy that responds, however the
// underlying http.Client is cached and shared across all rest.Client
// instances created by this function, so TLS configuration is shared
// across all clients created by this function. The caller should ensure
// that the TLS configuration is compatible with all URLs that may be
// used.
func NewClient(clientConfig *ClientConfig) *rest.Client {
	var tcc *tls.Config

	if strings.HasPrefix(clientConfig.URL, "https:") {
		skip := clientConfig.TLS.SkipVerify
		roots, err := x509.SystemCertPool()
		if err != nil {
			log.Warn("cannot read system certificates, continuing anyway", slog.Any("error", err))
		}

		if !skip {
			if cacerts := clientConfig.TLS.CACerts; len(cacerts) != 0 {
				if ok := roots.AppendCertsFromPEM(cacerts); !ok {
					log.Warn("error reading CA certs")
				}
			}
		}

		tcc = &tls.Config{
			RootCAs:            roots,
			InsecureSkipVerify: skip,
		}
	}

	timeout := clientConfig.Timeout
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

	if clientConfig.Trace {
		hc.Transport = &LogTransport{
			Transport: hc.Transport.(*http.Transport),
		}
	}

	return rest.NewClient(
		rest.BaseURLString(clientConfig.URL),
		rest.HTTPClient(hc),
		rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ []byte) {
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", clientConfig.Token))
		}),
	)
}

// Connect returns a sequence of *ClientConfig for each URL in the
// configuration. The caller can attempt to connect to each configured
// URL in turn until a successful connection is made.
//
// The order of the URLs is not guaranteed, so the caller should not
// rely on the order of the URLs in the configuration. The caller should
// also be prepared to handle connection failures, as some of the URLs
// may be unreachable or misconfigured.
//
// The configuration keys suported are:
//
//   - `url`: a list of URLs to connect to, e.g. `["https://ims1.example.com/api", "https://ims2.example.com/api"]`. The IMS type (e.g. `snow`) will be appended, as required by `ims-gateway`, to the URL when creating the client, so the actual URLs used will be `https://ims1.example.com/api/snow` and `https://ims2.example.com/api/snow`.
//   - `authentication.token`: the token to use for authentication, e.g. `abc123`
//   - `timeout`: the timeout for the connection, e.g. `10s`
//   - `tls.skip-verify`: whether to skip TLS verification, e.g. `true`
//   - `tls.chain`: a PEM encoded certificate chain to use for TLS verification, e.g. `-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----`
//   - `trace`: whether to enable HTTP request/response tracing, e.g. `true`
func Connect(imsCf *config.Config, imsType string) iter.Seq[*rest.Client] {
	return func(yield func(*rest.Client) bool) {
		for _, r := range config.Get[[]string](imsCf, "url") {
			ccf := &ClientConfig{
				URL:     r + "/" + imsType,
				Token:   config.Get[string](imsCf, config.Join("authentication", "token")),
				Timeout: config.Get[time.Duration](imsCf, config.Join("timeout")),
			}
			ccf.TLS.SkipVerify = config.Get[bool](imsCf, config.Join("tls", "skip-verify"))
			ccf.TLS.CACerts = config.Get[[]byte](imsCf, config.Join("tls", "chain"))
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

// LogTransport is a transport for tracing HTTP requests and responses.
type LogTransport struct {
	Transport http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Error("Error dumping request", slog.Any("error", err))
	} else {
		log.Debug("REQUEST:\n%s", slog.String("request", string(reqDump)))
	}

	// Perform the actual request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Log the response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Error("Error dumping response", slog.Any("error", err))
	} else {
		log.Debug("RESPONSE:\n%s", slog.String("response", string(respDump)))
	}

	return resp, nil
}
