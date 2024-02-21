package rest

import (
	"net/http"
)

// Options are used to control behaviour of ICP functions
type Options func(*restOptions)

type restOptions struct {
	baseURL      string
	client       *http.Client
	setupRequest func(req *http.Request, c *Client, endpoint string, body []byte)
}

func evalOptions(options ...Options) (opts *restOptions) {
	opts = &restOptions{
		baseURL: "https://localhost:443",
		client:  &http.Client{},
	}
	for _, opt := range options {
		opt(opts)
	}
	return
}

// BaseURL sets the root of the REST API URL. The default is
// "https://localhost:443"
func BaseURL(baseurl string) Options {
	return func(io *restOptions) {
		io.baseURL = baseurl
	}
}

// HTTPClient sets the http.Client to use for requests. The default is
// the default http package client.
func HTTPClient(client *http.Client) Options {
	return func(io *restOptions) {
		io.client = client
	}
}

func SetupRequestFunc(f func(req *http.Request, c *Client, endpoint string, body []byte)) Options {
	return func(ro *restOptions) {
		ro.setupRequest = f
	}
}
