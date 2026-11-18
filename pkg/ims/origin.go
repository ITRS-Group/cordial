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

package ims

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// ErrInsecureRedirect is returned for a redirect from https to http by a
// client that sends credentials.
var ErrInsecureRedirect = errors.New("refusing to follow a redirect from https to http")

// maxRedirects is the limit used when the client has no CheckRedirect,
// which is also the limit of [http.Client].
const maxRedirects = 10

// SameOrigin returns true if a and b have the same scheme, host name
// and port, using the default port for the scheme if there is none.
func SameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Hostname(), b.Hostname()) &&
		originPort(a) == originPort(b)
}

func originPort(u *url.URL) string {
	if p := u.Port(); p != "" {
		return p
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	}
	return ""
}

// OriginOnly returns a [http.RoundTripper] that sends requests for the
// same origin (scheme, host and port) as base using authed, which
// should add the credentials, such as an [oauth2.Transport], and all
// other requests, such as after a redirect to another host, port or from
// https to http, using plain with any Authorization or
// Proxy-Authorization header removed.
//
// A [http.Client] adds the token for every request it sends, including
// those that follow a redirect, so this stops the token being sent to
// any other origin.
func OriginOnly(base *url.URL, authed, plain http.RoundTripper) http.RoundTripper {
	if plain == nil {
		plain = http.DefaultTransport
	}
	return &originTransport{origin: *base, authed: authed, plain: plain}
}

type originTransport struct {
	origin url.URL
	authed http.RoundTripper
	plain  http.RoundTripper
}

func (t *originTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if SameOrigin(&t.origin, req.URL) {
		return t.authed.RoundTrip(req)
	}
	if req.Header.Get("Authorization") != "" || req.Header.Get("Proxy-Authorization") != "" {
		// a RoundTripper must not modify the request
		req = req.Clone(req.Context())
		req.Header.Del("Authorization")
		req.Header.Del("Proxy-Authorization")
	}
	return t.plain.RoundTrip(req)
}

// SameOriginRedirects sets the CheckRedirect of c so that a client that
// sends credentials in headers does not send them to another origin
// than base. A redirect to another origin (scheme, host or port) has the
// Authorization and Proxy-Authorization headers, and any others given in
// headers, such as an API key, removed. A redirect from https to http is
// refused with [ErrInsecureRedirect]. Any CheckRedirect already set on c
// is then called, or [http.Client] is limited to 10 redirects.
func SameOriginRedirects(c *http.Client, base *url.URL, headers ...string) {
	next := c.CheckRedirect
	origin := *base
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && strings.EqualFold(via[len(via)-1].URL.Scheme, "https") && strings.EqualFold(req.URL.Scheme, "http") {
			return ErrInsecureRedirect
		}
		if !SameOrigin(&origin, req.URL) {
			req.Header.Del("Authorization")
			req.Header.Del("Proxy-Authorization")
			for _, h := range headers {
				req.Header.Del(h)
			}
		}
		if next != nil {
			return next(req, via)
		}
		if len(via) >= maxRedirects {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
}
