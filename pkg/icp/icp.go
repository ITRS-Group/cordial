package icp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ICP holds the projectID, endpoint and http client for the configured
// ICP instance. If token is set it is sent with each request as a auth header.
type ICP struct {
	ProjectID int
	BaseURL   string
	Client    *http.Client
	Token     string
}

// New returns a new ICP object. BaseURL defaults to
// "https://icp-api.itrsgroup.com/v2.0" and client, if nil, to a default
// http.Client
func New(projectID int, options ...Options) (icp *ICP) {
	opts := evalOptions(options...)
	icp = &ICP{
		ProjectID: projectID,
		BaseURL:   opts.baseURL,
		Client:    opts.client,
	}
	return
}

// Post makes a POST request to the endpoint after marshalling the body
// as json.
func (icp *ICP) Post(endpoint string, body interface{}) (resp *http.Response, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
	if err != nil {
		return
	}
	j, err := json.Marshal(body)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", dest, bytes.NewReader(j))
	if err != nil {
		return
	}
	if icp.Token != "" {
		req.Header.Add("Authorization", "SUMERIAN "+icp.Token)
	}
	req.Header.Add("content-type", "application/json")
	return icp.Client.Do(req)
}

// Get sends a GET request to the endpoint
func (icp *ICP) Get(endpoint string, params ...string) (reply string, err error) {
	dest, err := url.JoinPath(icp.BaseURL, endpoint)
	if err != nil {
		return
	}
	req, err := http.NewRequest("GET", dest, nil)
	if err != nil {
		return
	}
	if icp.Token != "" {
		req.Header.Add("Authorization", "SUMERIAN "+icp.Token)
	}
	q := url.Values{}
	for _, p := range params {
		s := strings.SplitN(p, "=", 2)
		if len(s) != 2 {
			continue
		}
		q.Add(s[0], s[1])
	}
	req.URL.RawQuery = q.Encode()
	resp, err := icp.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		err = fmt.Errorf("%s", resp.Status)
		return
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	reply = string(b)
	return
}
