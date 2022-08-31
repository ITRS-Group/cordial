/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
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
		logError.Print(err)
		return false
	}
	return res
}

func (c *Client) InsecureSkipVerify() {
	tr := &http.Transport{
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
