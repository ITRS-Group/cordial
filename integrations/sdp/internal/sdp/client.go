/*
Copyright © 2026 ITRS Group

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

package sdp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

type Client struct {
	*rest.Client
	cf *config.Config
}

// Client returns an HTTP client which automatically adds a valid access
// token to requests and handles token refreshes as needed. The client
// will use the provided config to obtain an initial token if necessary,
// and will persist any new tokens obtained through refreshes.
//
// The scopes parameter is passed to the initial token retrieval
// process, but is not used for token refreshes since the scopes cannot
// be changed after the initial token is obtained.
//
// Note that for SDP the initial authentication process requires an
// authorization code which must be obtained through a separate process
// (e.g. using the auth command) and is not handled by this function.
// Therefore, if no valid token is found, this function will return an
// error rather than prompting for the authorization code.
func NewClient(ctx context.Context, cf *config.Config, scopes ...string) (client *Client, err error) {
	var tcc *tls.Config

	token, err := LoadToken()
	if err != nil {
		return
	}

	clientID := cf.GetString("client-id")
	clientSecret := cf.GetPassword("client-secret")

	if clientID == "" || clientSecret.IsNil() {
		err = fmt.Errorf("client-id and/or client-secret are not valid")
		return
	}

	auth, err := url.Parse(cf.GetString(cf.Join("datacentres", cf.GetString("datacentre"), "auth")))
	if err != nil {
		return
	}

	if auth.Scheme == "https" {
		tcc = &tls.Config{
			InsecureSkipVerify: cf.GetBool(cf.Join("tls", "skip-verify")),
		}
	}

	conf := &Config{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret.String(),
			Endpoint: oauth2.Endpoint{
				TokenURL: auth.JoinPath("/oauth/v2/token").String(),
			},
			RedirectURL: "https://www.zoho.com",
			Scopes:      scopes,
		},
		Code: nil,
	}

	timeout := cf.GetDuration(cf.Join("proxy", "timeout"))
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	hc := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       tcc,
		},
		Timeout: timeout,
	}

	if cf.GetBool("trace") {
		hc.Transport = &LogTransport{
			Transport: hc.Transport.(*http.Transport),
		}
	}

	c := oauth2.NewClient(context.WithValue(context.Background(), oauth2.HTTPClient, hc), NewSDPTokenSource(ctx, conf, token))

	client = &Client{
		Client: rest.NewClient(
			rest.HTTPClient(c),
			rest.BaseURLString(cf.GetString(cf.Join("datacentres", cf.GetString("datacentre"), "api"))),
			rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
				req.Header.Set("Accept", "application/vnd.manageengine.sdp.v3+json")
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			})),
		cf: cf,
	}

	return
}

// debug transport for tracing

type LogTransport struct {
	Transport http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Printf("Error dumping request: %v", err)
	} else {
		log.Printf("REQUEST:\n%s", reqDump)
	}

	// Perform the actual request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Log the response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Printf("Error dumping response: %v", err)
	} else {
		log.Printf("RESPONSE:\n%s", respDump)
	}

	return resp, nil
}
