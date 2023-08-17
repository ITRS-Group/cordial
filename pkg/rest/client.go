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

	"github.com/google/go-querystring/query"
	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Package rest provides simple client interfaces for REST calls with
// automatic marshalling and unmarshalling of request and response.

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	authHeader string
	authValue  string
}

var ErrServerError = errors.New("server error")

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
func (c *Client) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
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
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	defer resp.Body.Close()
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

// Post method
func (c *Client) Post(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(c.BaseURL, endpoint)
	if err != nil {
		return
	}
	j, err := json.Marshal(request)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "POST", dest, bytes.NewReader(j))
	if err != nil {
		return
	}
	if c.authHeader != "" {
		req.Header.Add(c.authHeader, c.authValue)
	}
	req.Header.Add("content-type", "application/json")
	resp, err = c.HTTPClient.Do(req)
	if resp.StatusCode > 299 {
		b, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("%w: %s", ErrServerError, string(b))
		return
	}
	defer resp.Body.Close()
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
	return
}

// Delete Method
func (c *Client) Delete(ctx context.Context, endpoint string, request interface{}) (resp *http.Response, err error) {
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
	if resp.StatusCode > 299 {
		err = ErrServerError
		return
	}
	// discard any body
	resp.Body.Close()
	return
}

func decodeResponse(resp *http.Response, response interface{}) (err error) {
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
	return
}
