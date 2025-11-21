package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/clbanning/mxj/v2"
	"github.com/google/go-querystring/query"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/config"
)

// Package rest provides a very simple client for REST calls with
// automatic marshalling and unmarshalling of request and response.
//
// GET requests encode parameters into the URL while POST and PUT
// requests...
//
// Responses must be in JSON and can be either an array or objects or a
// single object.

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	// if not nil, call SetUpRequest() just before the Do() call
	SetupRequest func(req *http.Request, c *Client, body []byte)
	authHeader   string
	authValue    string
	logger       *slog.Logger
}

// NewClient returns a *Client struct, ready to use. Unless options are
// supplied the base URL defaults to `https://localhost:443`.
func NewClient(options ...Options) *Client {
	opts := evalOptions(options...)
	return &Client{
		BaseURL:      opts.baseURL,
		HTTPClient:   opts.client,
		SetupRequest: opts.setupRequest,
		logger:       opts.logger,
	}
}

// SetAuth sets an explicit authentication header to value for clients
// that do not use OAUTH etc.
func (c *Client) SetAuth(header, value string) {
	c.authHeader = header
	c.authValue = value
}

// Auth sets up a 2-legged OAUTH2 with client ID and Secret. If clientID
// is empty just return, allowing callers to call when the value are not
// set (or empty) not set.
func (c *Client) Auth(ctx context.Context, clientID string, clientSecret *config.Plaintext) {
	if clientID == "" {
		return
	}
	params := make(url.Values)
	params.Set("grant_type", "client_credentials")
	tokenauth := c.BaseURL.JoinPath("/oauth2/token")
	conf := &clientcredentials.Config{
		ClientID:       clientID,
		ClientSecret:   clientSecret.String(),
		EndpointParams: params,
		TokenURL:       tokenauth.String(),
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.HTTPClient)
	c.HTTPClient = conf.Client(ctx)
}

// GET method. On successful return the response body will be closed.
// endpoint is either a string or a *url.URL relative to the client c
// base URL.
func (c *Client) Get(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	var dest *url.URL
	switch e := endpoint.(type) {
	case string:
		dest = c.BaseURL.JoinPath(e)
	case *url.URL:
		dest = c.BaseURL.ResolveReference(e)
	default:
		err = errors.ErrUnsupported
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest.String(), nil)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}

	switch r := request.(type) {
	case string:
		req.URL.RawQuery = r
	default:
		if r != nil {
			v, err := query.Values(r)
			if err != nil {
				return resp, err
			}
			req.URL.RawQuery = v.Encode()
		}
	}

	c.logger.Debug("get request", "url", req.URL.String(), "query", req.URL.RawQuery)

	if c.SetupRequest != nil {
		c.SetupRequest(req, c, nil)
	}
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}
	if response == nil {
		return
	}

	err = decodeResponse(resp, response)
	return
}

// Post issues a POST request to the client using endpoint as either a
// relative path to the base URL in the client, encoding request as a
// JSON body and decoding any returned body into result, if set.
// endpoint is either a string or a *url.URL relative to the client c
// base URL.
func (c *Client) Post(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	var dest *url.URL
	switch e := endpoint.(type) {
	case string:
		dest = c.BaseURL.JoinPath(e)
	case *url.URL:
		dest = c.BaseURL.ResolveReference(e)
	default:
		err = errors.ErrUnsupported
	}
	r, body := encodeBody(request)
	req, err := http.NewRequestWithContext(ctx, "POST", dest.String(), r)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")

	c.logger.Debug("post request", "url", req.URL.String(), "body", body)

	if c.SetupRequest != nil {
		c.SetupRequest(req, c, body)
	}
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

// PUT method
// endpoint is either a string or a *url.URL relative to the client c
// base URL.
func (c *Client) Put(ctx context.Context, endpoint any, request any, response any) (resp *http.Response, err error) {
	var dest *url.URL
	switch e := endpoint.(type) {
	case string:
		dest = c.BaseURL.JoinPath(e)
	case *url.URL:
		dest = c.BaseURL.ResolveReference(e)
	default:
		err = errors.ErrUnsupported
	}
	r, body := encodeBody(request)
	req, err := http.NewRequestWithContext(ctx, "PUT", dest.String(), r)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")

	c.logger.Debug("put request", "url", req.URL.String(), "body", body)

	if c.SetupRequest != nil {
		c.SetupRequest(req, c, body)
	}
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

// Delete Method
func (c *Client) Delete(ctx context.Context, endpoint string, request any) (resp *http.Response, err error) {
	dest := c.BaseURL.JoinPath(endpoint)
	req, err := http.NewRequestWithContext(ctx, "DELETE", dest.String(), nil)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	if c.SetupRequest != nil {
		c.SetupRequest(req, c, nil)
	}
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%s %s", resp.Status, string(b))
		return
	}
	return
}

func encodeBody(request any) (r io.Reader, body []byte) {
	if request == nil {
		return nil, nil
	}
	if s, ok := request.(string); ok {
		return bytes.NewReader([]byte(s)), []byte(s)
	}
	if b, ok := request.([]byte); ok {
		return bytes.NewReader(b), b
	}
	if j, err := json.Marshal(request); err == nil {
		return bytes.NewReader(j), j
	}
	return nil, nil
}

// decodeResponse checks the content-type and decodes based on that.
// This could be better done in a http handler, but this is simple to
// understand
func decodeResponse(resp *http.Response, response any) (err error) {
	// we only care about the main value, not char sets etc.
	ct := resp.Header.Get("content-type")
	ct, _, _ = strings.Cut(ct, ";")
	switch ct {
	case "text/plain", "text/html":
		switch rt := response.(type) {
		case []byte:
			b, _ := io.ReadAll(resp.Body)
			rt = b
		case string:
			b, _ := io.ReadAll(resp.Body)
			rt = string(b)
		default:
			err = http.ErrNotSupported
			_ = rt // satisfy linter
		}

	case "application/json":
		// stream JSON
		d := json.NewDecoder(resp.Body)
		rt := reflect.TypeOf(response)
		switch rt.Kind() {
		case reflect.Slice:
			rv := reflect.ValueOf(response)
			var t json.Token
			t, err = d.Token()
			if err != nil {
				return
			}
			if t != "[" {
				err = errors.New("not an array")
				return
			}
			for d.More() {
				var s interface{}
				if err = d.Decode(&s); err != nil {
					return
				}
				rv = reflect.Append(rv, reflect.ValueOf(s))
			}
			t, err = d.Token()
			if err != nil {
				return
			}
			if t != "]" {
				err = errors.New("array not terminated")
				return
			}
		default:
			err = d.Decode(&response)
		}
	case "text/xml", "application/xml":
		var mv mxj.Map
		if mv, err = mxj.NewMapXmlReader(resp.Body); err == nil { // all good ?
			response = mv
		}
	default:
		err = errors.ErrUnsupported
	}
	return
}

// decodeResponseBytes checks the content-type and decodes based on
// that. On failure it returns the body in b.
func decodeResponseBytes(resp *http.Response, response interface{}) (b []byte, err error) {
	if response == nil {
		return
	}
	// we only care about the main value, not char sets etc.
	ct := resp.Header.Get("content-type")
	ct, _, _ = strings.Cut(ct, ";")
	switch ct {
	case "application/json":
		// stream JSON
		d := json.NewDecoder(resp.Body)
		rt := reflect.TypeOf(response)
		switch rt.Kind() {
		case reflect.Slice:
			rv := reflect.ValueOf(response)
			var t json.Token
			t, err = d.Token()
			if err != nil {
				return
			}
			if t != "[" {
				err = errors.New("not an array")
				return
			}
			for d.More() {
				var s interface{}
				if err = d.Decode(&s); err != nil {
					return
				}
				rv = reflect.Append(rv, reflect.ValueOf(s))
			}
			t, err = d.Token()
			if err != nil {
				return
			}
			if t != "]" {
				err = errors.New("array not terminated")
				return
			}
		default:
			err = d.Decode(&response)
		}
	case "text/xml", "application/xml":
		var mv mxj.Map
		if mv, err = mxj.NewMapXmlReader(resp.Body); err == nil { // all good ?
			response = mv
		}
	default:
		// return bytes
		return io.ReadAll(resp.Body)
	}
	return
}
