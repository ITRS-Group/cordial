package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Package rest provides simple client interfaces for REST calls with
// automatic marshalling and unmarshalling of request and response.

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	// if not nil, call SetUpRequest() just before the Do() call
	SetupRequest func(req *http.Request, c *Client, endpoint string, body []byte)
	authHeader   string
	authValue    string
}

// NewClient returns a *Client struct, ready to use. Unless options are
// supplied the base URL defaults to `https://localhost:443`.
func NewClient(options ...Options) *Client {
	opts := evalOptions(options...)
	return &Client{
		BaseURL:      opts.baseURL,
		HTTPClient:   opts.client,
		SetupRequest: opts.setupRequest,
	}
}

// SetAuth sets an explicit auth head and value for clients that do not
// use OAUTH etc.
func (c *Client) SetAuth(header, value string) {
	c.authHeader = header
	c.authValue = value
}

// Auth with client ID and Secret. If clientid is empty just return,
// allowing callers to call with config values even when not set.
func (c *Client) Auth(ctx context.Context, clientid string, clientsecret *config.Plaintext) {
	if clientid == "" {
		return
	}
	params := make(url.Values)
	params.Set("grant_type", "client_credentials")
	tokenauth, _ := url.JoinPath(c.BaseURL + "/oauth2/token")
	conf := &clientcredentials.Config{
		ClientID:       clientid,
		ClientSecret:   clientsecret.String(),
		EndpointParams: params,
		TokenURL:       tokenauth,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.HTTPClient)
	c.HTTPClient = conf.Client(ctx)
}

// GET method. On successful return the response body will be closed.
func (c *Client) Get(ctx context.Context, endpoint string, request any, response any) (resp *http.Response, err error) {
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
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
		c.SetupRequest(req, c, endpoint, nil)
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

// POST method
func (c *Client) Post(ctx context.Context, endpoint string, request any, response any) (resp *http.Response, err error) {
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	r, body := encodeBody(request)
	req, err := http.NewRequestWithContext(ctx, "POST", dest, r)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")

	if c.SetupRequest != nil {
		c.SetupRequest(req, c, endpoint, body)
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
func (c *Client) Put(ctx context.Context, endpoint string, request any, response any) (resp *http.Response, err error) {
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	r, body := encodeBody(request)
	req, err := http.NewRequestWithContext(ctx, "PUT", dest, r)
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")
	if c.SetupRequest != nil {
		c.SetupRequest(req, c, endpoint, body)
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
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", dest, nil)
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
		c.SetupRequest(req, c, endpoint, nil)
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
func decodeResponse(resp *http.Response, response interface{}) (err error) {
	if response == nil {
		return
	}
	// we only care about the main value, not char sets etc.
	ct := resp.Header.Get("content-type")
	ct, _, _ = strings.Cut(ct, ";")
	switch ct {
	case "text/plain", "text/html":
		// decode as plain string and return as the err
		var b []byte
		if b, err = io.ReadAll(resp.Body); err == nil { // all good?
			err = errors.New(string(b))
		}
		return
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
