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
	authHeader string
	authValue  string
}

func NewClient(options ...Options) *Client {
	opts := evalOptions(options...)
	return &Client{
		BaseURL:    opts.baseURL,
		HTTPClient: opts.client,
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

// Get method. On successful return the response body will be closed.
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

// Post method
func (c *Client) Post(ctx context.Context, endpoint string, request any, response any) (resp *http.Response, err error) {
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "POST", dest, encodeRequest(request))
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")
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
	req, err := http.NewRequestWithContext(ctx, "PUT", dest, encodeRequest(request))
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")
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

func encodeRequest(request any) io.Reader {
	if request == nil {
		return nil
	}
	if s, ok := request.(string); ok {
		return bytes.NewReader([]byte(s))
	}
	if b, ok := request.([]byte); ok {
		return bytes.NewReader(b)
	}
	if j, err := json.Marshal(request); err == nil {
		return bytes.NewReader(j)
	}
	return nil
}

// decodeResponse checks the content-type and decodes based on that.
// This could be better done in a http handler, but this is simple to
// understand
func decodeResponse(resp *http.Response, response interface{}) (err error) {
	switch resp.Header.Get("content-type") {
	case "text/plain":
		// decode as plain string
		var b []byte
		if b, err = io.ReadAll(resp.Body); err == nil { // all good?
			response = string(b)
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
