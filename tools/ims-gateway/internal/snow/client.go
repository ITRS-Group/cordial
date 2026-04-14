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

package snow

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

type client struct {
	*rest.Client
}

// var snowMutex sync.RWMutex
// var snowConnection client

func newClient(cf *config.Config) (c client) {
	c = client{}

	username := cf.GetString("username")
	password := config.Get[*config.Plaintext](cf, "password")

	clientID := cf.GetString("client-id")
	clientSecret := config.Get[*config.Plaintext](cf, "client-secret")

	sn, err := url.Parse(cf.GetString("url"))
	if err != nil {
		return
	}

	var tcc *tls.Config
	if sn.Scheme == "https" {
		tcc = &tls.Config{
			InsecureSkipVerify: cf.GetBool(cf.Join("tls", "skip-verify")),
		}
	}

	timeout := cf.GetDuration(cf.Join("proxy", "timeout"))
	if timeout <= 0 {
		timeout = 10 * time.Second
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
	}

	p := sn.JoinPath(cf.GetString("path", config.Default("/api/now/v2/table")))

	logger := slog.New(slogzerolog.Option{Level: slog.LevelDebug, Logger: &log.Logger}.NewZerologHandler())

	if clientID != "" && !clientSecret.IsNil() {
		params := make(url.Values)
		params.Set("grant_type", "password")
		params.Set("username", username)
		params.Set("password", password.String())

		conf := &clientcredentials.Config{
			ClientID:       clientID,
			ClientSecret:   clientSecret.String(),
			EndpointParams: params,
			TokenURL:       sn.JoinPath("/oauth_token.do").String(),
		}

		hc = conf.Client(context.WithValue(context.Background(), oauth2.HTTPClient, hc))

		c.Client = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
			rest.Logger(logger),
		)
	} else {
		c.Client = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
			rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
				req.SetBasicAuth(username, password.String())
			}),
			rest.Logger(logger),
		)
	}

	return
}
