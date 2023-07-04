package gwhub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

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

// Get method
func (h *Hub) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(h.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
	if err != nil {
		return
	}
	if h.token != "" {
		req.Header.Add("Authorization", "Bearer "+h.token)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	resp, err = h.client.Do(req)
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
	switch t := response.(type) {
	case string:
		var b []byte
		b, err = io.ReadAll(resp.Body)
		t = string(b)
	default:
		d := json.NewDecoder(resp.Body)
		err = d.Decode(&t)
	}
	return
}

// Post method
func (h *Hub) Post(ctx context.Context, endpoint string, request interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(h.BaseURL, endpoint)
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
	if h.token != "" {
		req.Header.Add("Authorization", "Bearer "+h.token)
	}
	req.Header.Add("content-type", "application/json")
	return h.client.Do(req)
}

func (h *Hub) Ping(ctx context.Context) (resp *http.Response, err error) {
	return h.Get(ctx, PingEndpoint, nil, nil)
}
