package icp

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

// ICP holds the projectID, endpoint and http client for the configured
// ICP instance. If token is set it is sent with each request as a auth header.
type ICP struct {
	BaseURL string
	client  *http.Client
	token   string
}

// ErrServerError makes it a little easier for the caller to check the
// underlying HTTP response
var ErrServerError = errors.New("error from server (HTTP status > 299)")

// New returns a new ICP object. BaseURL defaults to
// "https://icp-api.itrsgroup.com/v2.0" and client, if nil, to a default
// http.Client
func New(options ...Options) (icp *ICP) {
	opts := evalOptions(options...)
	icp = &ICP{
		BaseURL: opts.baseURL,
		client:  opts.client,
	}
	return
}

// Post makes a POST request to the endpoint after marshalling the body
// as json. The caller must close resp.Body.
func (icp *ICP) Post(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
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
	if icp.token != "" {
		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
	}
	req.Header.Add("content-type", "application/json")
	resp, err = icp.client.Do(req)
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
	// return icp.client.Do(req)
}

// Get sends a GET request to the endpoint
func (icp *ICP) Get(ctx context.Context, endpoint string, request interface{}, response interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", dest, nil)
	if err != nil {
		return
	}
	if icp.token != "" {
		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	resp, err = icp.client.Do(req)
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

// Delete Method
func (icp *ICP) Delete(ctx context.Context, endpoint string, request interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", dest, nil)
	if err != nil {
		return
	}
	if icp.token != "" {
		req.Header.Add("Authorization", "SUMERIAN "+icp.token)
	}
	if request != nil {
		v, err := query.Values(request)
		if err != nil {
			return resp, err
		}
		req.URL.RawQuery = v.Encode()
	}
	resp, err = icp.client.Do(req)
	if resp.StatusCode > 299 {
		err = ErrServerError
		return
	}
	// discard any body
	resp.Body.Close()
	return
}
