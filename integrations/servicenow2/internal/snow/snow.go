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
	"net/http"
	"net/url"
	"sync"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

var snowMutex sync.RWMutex
var snowConnection *rest.Client

type Context struct {
	echo.Context
	Conf *config.Config
}

func ServiceNow(cf *config.Config) (rc *rest.Client) {
	snowMutex.RLock()
	if snowConnection != nil {
		snowMutex.RUnlock()
		return snowConnection
	}
	snowMutex.RUnlock()

	username := cf.GetString("username")
	password := cf.GetPassword("password")

	clientID := cf.GetString("clientid")
	clientSecret := cf.GetPassword("clientsecret")

	sn, err := url.Parse(cf.GetString("url"))
	if err != nil {
		return
	}

	hc := &http.Client{}

	if sn.Scheme == "https" {
		hc.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cf.GetBool(cf.Join("tls", "skip-verify")),
			},
		}
	}

	p := sn.JoinPath(cf.GetString("path", config.Default("/api/now/v2/table")))

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

		rc = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
		)
	} else {
		rc = rest.NewClient(
			rest.BaseURL(p),
			rest.HTTPClient(hc),
			rest.SetupRequestFunc(func(req *http.Request, c *rest.Client, body []byte) {
				req.SetBasicAuth(username, password.String())
			}),
		)
	}

	return
}
