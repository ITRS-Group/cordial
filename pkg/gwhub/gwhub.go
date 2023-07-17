package gwhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

// Hub holds connection details for a GW Hub
type Hub struct {
	BaseURL string
	client  *http.Client
	token   string
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("Error from server (HTTP Status > 299)")

func New(options ...Options) *Hub {
	opts := evalOptions(options...)
	return &Hub{
		BaseURL: opts.baseURL,
		client:  opts.client,
	}
}

// Get method. On successful return the response body will be closed.
func (hub *Hub) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(hub.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
	if err != nil {
		return
	}
	if hub.token != "" {
		req.Header.Add("Authorization", "Bearer "+hub.token)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	resp, err = hub.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		err = ErrServerError
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
func (hub *Hub) Post(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(hub.BaseURL, endpoint)
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
	if hub.token != "" {
		req.Header.Add("Authorization", "Bearer "+hub.token)
	}
	req.Header.Add("content-type", "application/json")
	resp, err = hub.client.Do(req)
	if resp.StatusCode > 299 {
		err = ErrServerError
		return
	}
	defer resp.Body.Close()
	if response == nil {
		return
	}
	err = decodeResponse(resp, response)
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

func (hub *Hub) Ping(ctx context.Context) (resp *http.Response, err error) {
	return hub.Get(ctx, PingEndpoint, nil, nil)
}
