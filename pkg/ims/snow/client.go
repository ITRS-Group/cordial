/*
Copyright © 2025 ITRS Group

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

// Package snow provides the ServiceNow Table API support shared by the
// ServiceNow proxy and the IMS gateway: a client that is cached by
// connection settings, lookup, create and update of records with
// responses that report failures as errors, and the configuration of
// the tables.
package snow

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	pathpkg "path"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/rest"
)

var log = logger.Logger

// Client is a client for the Table API of a ServiceNow instance
type Client struct {
	*rest.Client
	trace bool
}

var clientMutex sync.RWMutex
var clients = make(map[string]*Client)

// New returns the [Client] for the ServiceNow instance described by cf,
// which is the configuration section for the instance, creating and
// caching it on first use so that connections and OAuth tokens are
// reused between requests. Clients are cached by the settings that
// define the connection. An error is returned if the configuration is
// not usable.
//
// The configuration keys are `url`, `path` (default
// `/api/now/v2/table`), `username` and `password`, or `client-id` and
// `client-secret` as well for OAuth, `timeout` (default 10s),
// `tls.skip-verify` and `trace`. For compatibility with earlier
// versions `proxy.timeout` is used if `timeout` is not set.
//
// The `path` is the root of the Table API, to which the name of the
// table is added, and so `/api/now/v2/table`, with or without a
// trailing slash, and `/api/now/v2` are the same.
//
// A client is replaced if any of the settings, including the password
// and client secret, are changed.
func New(cf *config.Config) (c *Client, err error) {
	username := config.Get[string](cf, "username")
	password := config.Get[config.Secret](cf, "password")

	clientID := config.Get[string](cf, "client-id")
	clientSecret := config.Get[config.Secret](cf, "client-secret")

	sn, err := url.Parse(config.Get[string](cf, "url"))
	if err != nil {
		return nil, fmt.Errorf("invalid servicenow url: %w", err)
	}
	if sn.Scheme == "" || sn.Host == "" {
		return nil, fmt.Errorf("invalid servicenow url %q: scheme and host are required", sn.String())
	}

	skipVerify := config.Get[bool](cf, cf.Join("tls", "skip-verify"))
	var tcc *tls.Config
	if sn.Scheme == "https" {
		tcc = &tls.Config{
			InsecureSkipVerify: skipVerify,
		}
	}

	timeout := config.Get[time.Duration](cf, "timeout")
	if timeout <= 0 {
		timeout = config.Get[time.Duration](cf, cf.Join("proxy", "timeout"))
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	path := config.Get[string](cf, "path", config.DefaultValue("/api/now/v2/table"))
	trace := config.Get[bool](cf, "trace")

	// the secrets are part of the key, so that a client is not reused
	// after they are changed, but only as a hash
	secrets := sha256.Sum256([]byte(string(password) + "\x00" + string(clientSecret)))
	key := fmt.Sprint(sn, "|", tableBase(path), "|", username, "|", clientID, "|", timeout, "|", skipVerify, "|", trace, "|", secrets)
	clientMutex.RLock()
	c, ok := clients[key]
	clientMutex.RUnlock()
	if ok {
		return c, nil
	}

	// use most of the default transport settings
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
		// never follow redirects, as a redirected POST or PUT would be
		// silently turned into a GET without the record
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	p := sn.JoinPath(tableBase(path))
	if !strings.HasSuffix(p.Path, "/") {
		p.Path += "/"
	}

	c = &Client{trace: trace}

	if clientID != "" && len(clientSecret) > 0 {
		params := make(url.Values)
		params.Set("grant_type", "password")
		params.Set("username", username)
		params.Set("password", string(password))

		conf := &clientcredentials.Config{
			ClientID:       clientID,
			ClientSecret:   string(clientSecret),
			EndpointParams: params,
			TokenURL:       sn.JoinPath("/oauth_token.do").String(),
		}

		// the OAuth client does not inherit the settings of the base
		// client, so reapply them
		base := hc
		hc = conf.Client(context.WithValue(context.Background(), oauth2.HTTPClient, base))
		hc.Timeout = base.Timeout
		hc.CheckRedirect = base.CheckRedirect
		// the token is only for the instance, whatever the client is
		// redirected to
		hc.Transport = ims.OriginOnly(sn, hc.Transport, base.Transport)

		c.Client = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
			rest.Logger(log),
		)
	} else {
		c.Client = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
			rest.SetupRequestFunc(func(req *http.Request, _ *rest.Client, _ []byte) {
				req.SetBasicAuth(username, string(password))
			}),
			rest.Logger(log),
		)
	}

	// cache the connection
	clientMutex.Lock()
	clients[key] = c
	clientMutex.Unlock()

	return c, nil
}

// tableBase returns the path that the name of a table is added to, as
// the resolution of a relative URL, such as the result of [AssembleURL],
// replaces the last segment of the base URL, which must therefore end
// with a slash. A last segment of `table` is removed as that is part
// of the relative URL.
func tableBase(path string) string {
	p := pathpkg.Clean("/" + path)
	if pathpkg.Base(p) != "/" && strings.EqualFold(pathpkg.Base(p), "table") {
		p = pathpkg.Dir(p)
	}
	return strings.TrimRight(p, "/") + "/"
}
