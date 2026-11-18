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

// Package rest provides a JSON-oriented HTTP client with configurable
// authentication and request setup.
//
// A [Client] is configured once, using the [Option] functions and
// [Client.SetAuth] or [Client.Auth], and is then safe for concurrent
// use. Configuration errors, such as an unparseable base URL, are
// reported by the first request rather than by a panic.
//
// Requests that complete with a status outside 200-299 return an
// [*HTTPError], a request that is redirected in a way that changes its
// method (for example a POST that becomes a GET) returns a
// [*RedirectError], and failures to get any response (connection,
// TLS, timeout or context errors) are returned as the underlying
// transport error, so callers can tell the cases apart using
// [errors.As].
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/go-querystring/query"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/config"
)

const (
	// maxErrorBody is the most of an error response body that is read
	// and kept in an [HTTPError]
	maxErrorBody = 64 << 10

	// maxErrorText is the most of the body that is included in the
	// text of an [HTTPError]
	maxErrorText = 512
)

// HTTPError is returned when a request completes but the server
// responds with a status code outside the range 200-299. The response
// body, if any (up to 64KiB), is included in the error text and is
// available in Body. Use [errors.As] to distinguish a response from the
// server from a failure to get one.
type HTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

// Error returns the status followed by the response body. A long body
// is truncated in the text, but is available in full in Body.
func (e *HTTPError) Error() string {
	status := e.Status
	if status == "" {
		status = fmt.Sprintf("%d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}
	body := strings.TrimSpace(string(e.Body))
	if body == "" {
		return status
	}
	if len(body) > maxErrorText {
		body = strings.ToValidUTF8(body[:maxErrorText], "") + "..."
	}
	return status + " " + body
}

// RedirectError is returned when the server redirected a request in a
// way that changed the method, which means that the original request
// was not applied. The most common case is a POST that is redirected
// with a 301, 302 or 303 status and so is repeated as a GET, which can
// look like success to the caller.
type RedirectError struct {
	Method      string
	URL         string
	StatusCode  int // of the redirecting response, if known
	FinalMethod string
	FinalURL    string
}

// Error returns a description of the redirect
func (e *RedirectError) Error() string {
	return fmt.Sprintf("%s %s was redirected (%d) and re-sent as %s %s: the original request was not applied",
		e.Method, e.URL, e.StatusCode, e.FinalMethod, e.FinalURL)
}

// Client sends HTTP requests to a REST service and decodes responses.
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	// if not nil, call SetupRequest() just before the Do() call
	SetupRequest func(req *http.Request, c *Client, body []byte)
	authHeader   string
	authValue    string
	logger       *slog.Logger

	// err is a configuration error to be returned by every request
	err error
}

// NewClient returns a *Client struct, ready to use. Unless options are
// supplied the base URL defaults to `https://localhost`. Any error in
// the options, such as an invalid base URL, is returned by the first
// request made with the client.
func NewClient(options ...Option) *Client {
	opts := evalOptions(options...)
	hc := opts.client
	if opts.noRedirect {
		// copy, as the supplied client may be shared
		c := *hc
		c.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
		hc = &c
	}
	return &Client{
		BaseURL:      opts.baseURL,
		HTTPClient:   hc,
		SetupRequest: opts.setupRequest,
		logger:       opts.logger,
		err:          opts.err(),
	}
}

// SetAuth sets an explicit authentication header to value for clients
// that do not use OAUTH etc. It should be called before the client is
// used.
func (c *Client) SetAuth(header, value string) {
	c.authHeader = header
	c.authValue = value
}

// Auth sets up a 2-legged OAUTH2 with client ID and Secret. If clientID
// is empty just return, allowing callers to call when the value are not
// set (or empty) not set. It should be called before the client is
// used. ctx is not retained, so it may be cancelled after Auth returns
// without affecting later token refreshes.
func (c *Client) Auth(ctx context.Context, clientID string, clientSecret config.Secret) {
	if clientID == "" || c.BaseURL == nil {
		return
	}
	params := make(url.Values)
	params.Set("grant_type", "client_credentials")
	tokenauth := c.BaseURL.JoinPath("/oauth2/token")
	conf := &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   string(clientSecret),
		EndpointParams: params,
		TokenURL:       tokenauth.String(),
	}

	// the token source keeps the context it is created with for later
	// refreshes, so it must not be the caller's, which may be cancelled
	// once Auth returns. conf.Client keeps the timeout, redirect policy
	// and cookie jar of the client in the context.
	tokenCtx := context.WithValue(context.Background(), oauth2.HTTPClient, c.httpClient())
	c.HTTPClient = conf.Client(tokenCtx)
}

// Do executes the given method (GET, POST, PUT, DELETE) with the given
// endpoint, request and response. Endpoint is either a string or a
// *url.URL relative to the client base URL. Request is either a struct
// that can be encoded as query parameters (for GET and DELETE) or JSON
// (for POST and PUT) or a string or []byte that is sent as-is.
// Response is a non-nil pointer to a value that can be decoded from
// JSON, XML or text, depending on the content-type of the response. On
// failure, the error will contain the status code and any body returned
// by the server.
func (c *Client) Do(ctx context.Context, method string, endpoint any, request any, response any) (resp *http.Response, err error) {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		return c.do(ctx, method, endpoint, request, response)
	default:
		return nil, errors.ErrUnsupported
	}
}

// Get issues a GET request. endpoint is either a string or a *url.URL
// relative to the client c base URL. request, if not nil, is encoded as
// the query string: a struct is encoded using `url` struct tags and a
// string or []byte is used as-is. The query is added to any query in
// endpoint. If response is not nil it must be a pointer and the
// response body is decoded into it.
//
// resp.Body will be closed on return, so the caller should not attempt
// to read from it.
func (c *Client) Get(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	return c.do(ctx, http.MethodGet, endpoint, request, response)
}

// Post issues a POST request to the client using endpoint as either a
// relative path to the base URL in the client, encoding request as a
// JSON body and decoding any returned body into response, if set.
// endpoint is either a string or a *url.URL relative to the client c
// base URL.
//
// resp.Body will be closed on return, so the caller should not attempt
// to read from it.
func (c *Client) Post(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	return c.do(ctx, http.MethodPost, endpoint, request, response)
}

// Put issues a PUT request. endpoint is either a string or a *url.URL
// relative to the client c base URL. See [Client.Post].
//
// resp.Body will be closed on return, so the caller should not attempt
// to read from it.
func (c *Client) Put(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	return c.do(ctx, http.MethodPut, endpoint, request, response)
}

// Delete issues a DELETE request. request, if not nil, is encoded as
// the query string, as for [Client.Get].
//
// resp.Body will be closed on return, so the caller should not attempt
// to read from it.
func (c *Client) Delete(ctx context.Context, endpoint string, request any) (resp *http.Response, err error) {
	return c.do(ctx, http.MethodDelete, endpoint, request, nil)
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// do is the common implementation of all the methods
func (c *Client) do(ctx context.Context, method string, endpoint, request, response any) (resp *http.Response, err error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.BaseURL == nil {
		return nil, errNoBaseURL
	}

	dest, err := c.resolve(endpoint)
	if err != nil {
		return nil, err
	}

	var r io.Reader
	var body []byte
	hasBody := method == http.MethodPost || method == http.MethodPut
	if hasBody {
		if r, body, err = encodeBody(request); err != nil {
			return nil, err
		}
	} else if err = addQuery(dest, request); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, dest.String(), r)
	if err != nil {
		return nil, err
	}
	if c.authHeader != "" {
		req.Header.Set(c.authHeader, c.authValue)
	}
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.send(req, body, hasBody, response)
}

// Send sends a request that was built by the caller, for the cases
// where the request body cannot be encoded by the other methods, such
// as a stream. The caller must set the URL, normally using BaseURL, and
// any headers, but not the client's authentication header, which is
// added. The client's SetupRequest function, logging, redirect and
// status handling and response decoding are applied as for the other
// methods, and as with them resp.Body is closed on return.
func (c *Client) Send(req *http.Request, response any) (resp *http.Response, err error) {
	if c.err != nil {
		closeRequestBody(req)
		return nil, c.err
	}
	if c.authHeader != "" {
		req.Header.Set(c.authHeader, c.authValue)
	}
	return c.send(req, nil, req.Body != nil, response)
}

func closeRequestBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}

// send is the common part of do and Send. body is the encoded request
// body, if known, and hasBody is true if there is one.
func (c *Client) send(req *http.Request, body []byte, hasBody bool, response any) (resp *http.Response, err error) {
	method := req.Method
	if c.logger != nil {
		// request bodies and query values may contain credentials, so
		// only their size and names are logged
		if hasBody {
			c.logger.Debug(strings.ToLower(method)+" request", "url", logURL(req.URL), "body_bytes", req.ContentLength)
		} else {
			c.logger.Debug(strings.ToLower(method)+" request", "url", logURL(req.URL), "query_keys", queryKeys(req.URL.RawQuery))
		}
	}

	if c.SetupRequest != nil {
		c.SetupRequest(req, c, body)
	}

	resp, err = c.clientFor(req).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if c.logger != nil {
		c.logger.Debug(strings.ToLower(method)+" response", "url", logURL(req.URL), "status", resp.Status)
	}

	// a redirect that changed the method means the server never saw
	// the original request
	if resp.Request != nil && resp.Request.Method != method {
		re := &RedirectError{
			Method:      method,
			URL:         req.URL.Redacted(),
			FinalMethod: resp.Request.Method,
		}
		if resp.Request.URL != nil {
			re.FinalURL = resp.Request.URL.Redacted()
		}
		if resp.Request.Response != nil {
			re.StatusCode = resp.Request.Response.StatusCode
		}
		return resp, re
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return resp, &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status, Body: b}
	}

	if response == nil {
		// let the connection be reused
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBody))
		return
	}

	err = decodeResponse(resp, response)
	return
}

// clientFor returns the http client to use for req. If a custom
// authentication header is set it is removed from any redirected
// request to a different host, as Go itself only does this for a fixed
// list of headers.
func (c *Client) clientFor(req *http.Request) *http.Client {
	hc := c.httpClient()
	if c.authHeader == "" {
		return hc
	}
	header, host, check := c.authHeader, req.URL.Host, hc.CheckRedirect
	cc := *hc
	cc.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if !strings.EqualFold(next.URL.Host, host) {
			next.Header.Del(header)
		}
		if check != nil {
			return check(next, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &cc
}

// logURL returns u as a string without any query or password
func logURL(u *url.URL) string {
	c := *u
	c.RawQuery, c.ForceQuery, c.Fragment, c.RawFragment = "", false, "", ""
	return c.Redacted()
}

// queryKeys returns a comma separated list of the names in a raw query
func queryKeys(rawQuery string) string {
	v, _ := url.ParseQuery(rawQuery)
	return strings.Join(slices.Sorted(maps.Keys(v)), ",")
}

var errNoBaseURL = errors.New("rest: client has no base URL")

// resolve returns the destination URL for endpoint, which is either a
// string or a *url.URL
func (c *Client) resolve(endpoint any) (dest *url.URL, err error) {
	switch e := endpoint.(type) {
	case string:
		// a query in the string must not be escaped into the path
		path, rawQuery, _ := strings.Cut(e, "?")
		dest = c.BaseURL.JoinPath(path)
		if rawQuery != "" {
			dest.RawQuery = rawQuery
		}
	case *url.URL:
		if e == nil {
			return nil, errors.ErrUnsupported
		}
		dest = c.BaseURL.ResolveReference(e)
	default:
		return nil, errors.ErrUnsupported
	}
	return
}

// addQuery encodes request as a query string and adds it to the query
// in dest. request is a string or []byte, which is used as-is, or
// [url.Values] or a struct with `url` tags.
func addQuery(dest *url.URL, request any) error {
	var q string
	switch r := request.(type) {
	case nil:
	case string:
		q = r
	case []byte:
		q = string(r)
	case url.Values:
		q = r.Encode()
	default:
		v, err := query.Values(r)
		if err != nil {
			return err
		}
		q = v.Encode()
	}
	switch {
	case q == "":
	case dest.RawQuery == "":
		dest.RawQuery = q
	default:
		dest.RawQuery += "&" + q
	}
	return nil
}

// encodeBody returns a reader and the bytes of the request body. A
// string or []byte is sent as-is and anything else is encoded as JSON.
func encodeBody(request any) (r io.Reader, body []byte, err error) {
	switch rt := request.(type) {
	case nil:
		return nil, nil, nil
	case string:
		body = []byte(rt)
	case []byte:
		body = rt
	default:
		if body, err = json.Marshal(request); err != nil {
			return nil, nil, fmt.Errorf("rest: encoding request body: %w", err)
		}
	}
	return bytes.NewReader(body), body, nil
}
