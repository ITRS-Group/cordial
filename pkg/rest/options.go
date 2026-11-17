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

package rest

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Option configures a Client created by NewClient.
type Option func(*restOptions)

type restOptions struct {
	baseURL      *url.URL
	baseErr      error
	client       *http.Client
	noRedirect   bool
	setupRequest func(req *http.Request, c *Client, body []byte)
	logger       *slog.Logger
}

func evalOptions(options ...Option) (opts *restOptions) {
	opts = &restOptions{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	opts.baseURL, _ = url.Parse("https://localhost")
	for _, opt := range options {
		opt(opts)
	}
	return
}

// err returns any error in the options that makes the client unusable
func (o *restOptions) err() error {
	switch {
	case o.baseErr != nil:
		return o.baseErr
	case o.baseURL == nil:
		return errors.New("rest: base URL is nil")
	case !strings.EqualFold(o.baseURL.Scheme, "http") && !strings.EqualFold(o.baseURL.Scheme, "https"), o.baseURL.Host == "":
		return fmt.Errorf("rest: base URL %q is not an absolute http or https URL", o.baseURL.Redacted())
	}
	return nil
}

// BaseURLString sets the root of the REST API URL. The default is
// "https://localhost". An invalid URL is reported by the first request
// made with the client.
func BaseURLString(baseurl string) Option {
	return func(io *restOptions) {
		io.baseURL, io.baseErr = url.Parse(baseurl)
		if io.baseErr != nil {
			io.baseErr = fmt.Errorf("rest: invalid base URL: %w", io.baseErr)
		}
	}
}

// BaseURL sets the root URL used by the REST client. The default is
// "https://localhost". A nil URL is reported by the first request made
// with the client.
func BaseURL(baseurl *url.URL) Option {
	return func(io *restOptions) {
		io.baseURL = baseurl
		io.baseErr = nil
	}
}

// HTTPClient sets the http.Client to use for requests. The default is
// an http.Client with a 30 second timeout. A nil client is ignored.
func HTTPClient(client *http.Client) Option {
	return func(io *restOptions) {
		if client != nil {
			io.client = client
		}
	}
}

// NoRedirect stops the client following redirects. The redirecting
// response is then returned as an [*HTTPError]. The http.Client being
// used is copied, not modified, so it is safe to use with a shared
// client.
//
// Whether or not this option is used, a redirect that changes the
// method of a request, such as a POST that is re-sent as a GET, is
// returned as a [*RedirectError].
func NoRedirect() Option {
	return func(io *restOptions) {
		io.noRedirect = true
	}
}

// SetupRequestFunc sets a function to call while setting up the
// request. For example, to add Basic Authentication:
//
//	client = rest.NewClient(
//	        rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
//	            req.SetBasicAuth(username, password.String())
//	        }),
//	    )
func SetupRequestFunc(f func(req *http.Request, c *Client, body []byte)) Option {
	return func(ro *restOptions) {
		ro.setupRequest = f
	}
}

// Logger sets the logger used by the REST client. A nil logger disables
// request logging. Requests and responses are logged at debug level,
// including the URL and query or body of each request, so do not enable
// debug logging where these may contain credentials that must not be
// recorded. Authentication headers are never logged.
func Logger(logger *slog.Logger) Option {
	return func(ro *restOptions) {
		ro.logger = logger
	}
}
