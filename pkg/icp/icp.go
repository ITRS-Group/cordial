package icp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/google/go-querystring/query"
)

// ICP holds the projectID, endpoint and http client for the configured
// ICP instance. If token is set it is sent with each request as a auth header.
type ICP struct {
	ProjectID int
	BaseURL   string
	client    *http.Client
	token     string
}

var ErrServerError = errors.New("Error from server (HTTP Status > 299)")

// New returns a new ICP object. BaseURL defaults to
// "https://icp-api.itrsgroup.com/v2.0" and client, if nil, to a default
// http.Client
func New(projectID int, options ...Options) (icp *ICP) {
	opts := evalOptions(options...)
	icp = &ICP{
		ProjectID: projectID,
		BaseURL:   opts.baseURL,
		client:    opts.client,
	}
	return
}

// Post makes a POST request to the endpoint after marshalling the body
// as json.
func (icp *ICP) Post(ctx context.Context, endpoint string, body interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
	if err != nil {
		return
	}
	j, err := json.Marshal(body)
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
	return icp.client.Do(req)
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
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&response)
	return
}
