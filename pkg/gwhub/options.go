package gwhub

import "net/http"

// Options are used to control behaviour of ICP functions
type Options func(*gwhubOptions)

type gwhubOptions struct {
	baseURL string
	client  *http.Client
}

func evalOptions(options ...Options) (opts *gwhubOptions) {
	opts = &gwhubOptions{
		baseURL: "https://localhost:8443",
		client:  &http.Client{},
	}
	for _, opt := range options {
		opt(opts)
	}
	return
}

// BaseURL sets the root of the REST API URL. The default is
// "https://localhost:8443"
func BaseURL(baseurl string) Options {
	return func(io *gwhubOptions) {
		io.baseURL = baseurl
	}
}

// HTTPClient sets the http.Client to use for requests. The default is
// the default http package client.
func HTTPClient(client *http.Client) Options {
	return func(io *gwhubOptions) {
		io.client = client
	}
}
