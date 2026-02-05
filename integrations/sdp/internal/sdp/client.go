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
)

func Client(ctx context.Context, cf *config.Config) (client *http.Client, err error) {
	var tcc *tls.Config

	token, err := LoadToken()
	if err != nil {
		return
	}

	log.Debug().Msgf("loaded token: %+v", token)

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
			Scopes:      []string{"SDPOnDemand.requests.ALL"},
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

	client = oauth2.NewClient(context.WithValue(context.Background(), oauth2.HTTPClient, hc), NewSDPTokenSource(ctx, conf, token))

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
