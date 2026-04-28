package rest

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Option func(*restOptions)

type restOptions struct {
	baseURL      *url.URL
	client       *http.Client
	setupRequest func(req *http.Request, c *Client, body []byte)
	logger       *slog.Logger
}

func evalOptions(options ...Option) (opts *restOptions) {
	opts = &restOptions{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	opts.baseURL, _ = url.Parse("https://localhost")
	for _, opt := range options {
		opt(opts)
	}
	return
}

// BaseURLString sets the root of the REST API URL. The default is
// "https://localhost"
func BaseURLString(baseurl string) Option {
	return func(io *restOptions) {
		io.baseURL, _ = url.Parse(baseurl)
	}
}

// BaseURLString sets the root of the REST API URL. The default is
// "https://localhost"
func BaseURL(baseurl *url.URL) Option {
	return func(io *restOptions) {
		io.baseURL = baseurl
	}
}

// HTTPClient sets the http.Client to use for requests. The default is
// the default http package client.
func HTTPClient(client *http.Client) Option {
	return func(io *restOptions) {
		io.client = client
	}
}

// SetupRequestFunc sets a function to call while setting up the
// request. For example, to add Basic Authentication:
//
//	client = rest.NewClient(
//	        rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
//	            req.SetBasicAuth(username, password.String())
//	        }),
//	    )
func SetupRequestFunc(f func(req *http.Request, c *Client, body []byte)) Option {
	return func(ro *restOptions) {
		ro.setupRequest = f
	}
}

func Logger(logger *slog.Logger) Option {
	return func(ro *restOptions) {
		ro.logger = logger
	}
}
