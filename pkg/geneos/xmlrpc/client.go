/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xmlrpc

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// The Client struct carries the http Client and the url down to
// successive layers
type Client struct {
	http.Client
	url *url.URL
}

// idleConnTimeout is how long an unused connection is kept open by the
// client's transport before being closed.
const idleConnTimeout = 30 * time.Second

// newTransport returns a transport that closes idle connections itself,
// without needing a background goroutine to do so.
func newTransport(insecureSkipVerify bool) *http.Transport {
	return &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		IdleConnTimeout: idleConnTimeout,
	}
}

// NewClient returns a Client for the XML-RPC endpoint at url. Unless
// the InsecureSkipVerify or Secure(false) options are given, HTTPS
// certificates are verified. The Timeout option sets a limit on each
// request, the default being no limit. Idle connections are closed by
// the transport after 30 seconds so there is nothing to clean up when
// the Client is discarded.
func NewClient(url *url.URL, options ...Option) (c *Client) {
	opt := &xmlrpcOptions{}
	evalOptions(opt, options...)
	c = &Client{
		Client: http.Client{
			Transport: newTransport(opt.insecureSkipVerify),
			Timeout:   opt.timeout,
		},
		url: url,
	}
	return
}

// String conforms to the Stringer Type
func (c Client) String() string {
	return c.url.String()
}

// Connected checks if the client is connected to a gateway.
func (c Client) Connected() bool {
	res, err := c.gatewayConnected()
	if err != nil {
		return false
	}
	return res
}

// InsecureSkipVerify disables verification of the server certificate
// for HTTPS connections made by this Client.
func (c *Client) InsecureSkipVerify() {
	c.Client.Transport = newTransport(true)
}

// Sampler creates and returns a new Sampler struct from the lower
// level.
//
// XXX At the moment there is no error checking or validation
func (c Client) Sampler(entityName string, samplerName string) (sampler Sampler, err error) {
	sampler = Sampler{Client: c, entityName: entityName, samplerName: samplerName}
	return
}
