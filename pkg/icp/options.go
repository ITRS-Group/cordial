package icp

import "net/http"

// Options are used to control behaviour of ICP functions
type Options func(*icpOptions)

type icpOptions struct {
	baseURL string
	client  *http.Client
}

func evalOptions(options ...Options) (opts *icpOptions) {
	opts = &icpOptions{
		baseURL: "https://icp-api.itrsgroup.com/v2.0",
		client:  &http.Client{},
	}
	for _, opt := range options {
		opt(opts)
	}
	return
}

// BaseURL sets the root of the REST API URL. The default is
// "https://icp-api.itrsgroup.com/v2.0". No change to the default URL is
// made if called with an empty string, so you can us this without
// conditional testing in callers
func BaseURL(baseurl string) Options {
	return func(io *icpOptions) {
		if baseurl != "" {
			io.baseURL = baseurl
		}
	}
}

// HTTPClient sets the http.Client to use for requests. The default is
// the default http package client. If client is nil then the default
// client is used, letting callers add this option without coditional
// testing.
func HTTPClient(client *http.Client) Options {
	return func(io *icpOptions) {
		if client != nil {
			io.client = client
		}
	}
}
