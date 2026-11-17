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
	"net/url"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/rest"
)

type client struct {
	*rest.Client
	sdpCf *config.Config
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
func newClient(ctx context.Context, sdpCf *config.Config, scopes ...string) (c *client, err error) {
	token, err := tokenLoader()
	if err != nil {
		return
	}

	clientID := config.Get[string](sdpCf, "client-id")
	clientSecret := config.Get[config.Secret](sdpCf, "client-secret")

	if clientID == "" || len(clientSecret) == 0 {
		err = fmt.Errorf("client-id and/or client-secret are not valid")
		return
	}

	datacentre := config.Get[string](sdpCf, "datacentre")
	authURL := config.Get[string](sdpCf, sdpCf.Join("datacentres", datacentre, "auth"))
	apiURL := config.Get[string](sdpCf, sdpCf.Join("datacentres", datacentre, "api"))

	auth, err := validURL("auth", authURL, datacentre)
	if err != nil {
		return
	}
	if _, err = validURL("api", apiURL, datacentre); err != nil {
		return
	}

	conf := &Config{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: string(clientSecret),
			Endpoint: oauth2.Endpoint{
				TokenURL: auth.JoinPath("/oauth/v2/token").String(),
			},
			RedirectURL: "https://www.zoho.com",
			Scopes:      scopes,
		},
		Code: nil,
	}

	timeout := config.Get[time.Duration](sdpCf, "timeout")
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	var rt http.RoundTripper = sharedTransport(config.Get[bool](sdpCf, sdpCf.Join("tls", "skip-verify")))
	if config.Get[bool](sdpCf, "trace") {
		rt = &ims.LogTransport{Transport: rt}
	}

	// token refreshes must use the same transport and timeout as the
	// API requests
	base := &http.Client{Transport: rt, Timeout: timeout}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, base)

	hc := oauth2.NewClient(ctx, NewSDPTokenSource(ctx, conf, token))
	// the OAuth client does not inherit the settings of the base client
	hc.Timeout = timeout
	// never follow redirects, as a redirected POST or PUT would be
	// silently turned into a GET without the data
	hc.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	c = &client{
		Client: rest.NewClient(
			rest.HTTPClient(hc),
			rest.BaseURLString(apiURL),
			rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
				req.Header.Set("Accept", "application/vnd.manageengine.sdp.v3+json")
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			})),
		sdpCf: sdpCf,
	}

	return
}

// validURL parses and checks a datacentre URL
func validURL(name, value, datacentre string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s url for datacentre %q: %w", name, datacentre, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid %s url %q for datacentre %q: scheme and host are required", name, value, datacentre)
	}
	return u, nil
}

var transportMutex sync.Mutex
var transports = make(map[bool]*http.Transport)

// sharedTransport returns a transport that is shared by all clients, as
// a new client is created for each request and each transport holds its
// own pool of connections.
func sharedTransport(skipVerify bool) *http.Transport {
	transportMutex.Lock()
	defer transportMutex.Unlock()

	if t, ok := transports[skipVerify]; ok {
		return t
	}

	t := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: skipVerify},
	}
	transports[skipVerify] = t
	return t
}
