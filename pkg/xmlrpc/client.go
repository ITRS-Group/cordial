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
)

// The Client struct carries the http Client and the url down to
// successive layers
type Client struct {
	http.Client
	url *url.URL
}

func NewClient(url *url.URL, options ...Options) (c *Client) {
	opt := &xmlrpcOptions{}
	evalOptions(opt, options...)
	c = &Client{url: url}
	if opt.insecureSkipVerify {
		c.Client.Transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
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

func (c *Client) InsecureSkipVerify() {
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c.Client = http.Client{Transport: tr}
}

// Sampler creates and returns a new Sampler struct from the lower
// level.
//
// XXX At the moment there is no error checking or validation
func (c Client) Sampler(entityName string, samplerName string) (sampler Sampler, err error) {
	sampler = Sampler{Client: c, entityName: entityName, samplerName: samplerName}
	return
}
