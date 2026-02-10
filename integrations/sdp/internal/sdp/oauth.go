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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// DefaultUserKeyfile is the path to the user's key file as a
// config.Keyfile type
var DefaultUserKeyfile = config.KeyFile(
	config.Path("keyfile",
		config.SetAppName("geneos"), // we use the geneos keyfile, not a separate one for this integration
		config.SetFileExtension("aes"),
		config.IgnoreWorkingDir(),
	),
)

type Config struct {
	oauth2.Config
	Code *config.Plaintext
}

type SDPTokenSource struct {
	next   oauth2.TokenSource
	expiry time.Time
}

func NewSDPTokenSource(ctx context.Context, conf *Config, token *oauth2.Token) *SDPTokenSource {
	return &SDPTokenSource{
		next:   conf.TokenSource(ctx, token),
		expiry: token.Expiry,
	}
}

func (s *SDPTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.next.Token()
	if err != nil {
		return nil, err
	}

	if s.expiry != token.Expiry {
		if err := saveToken(token); err != nil {
			return nil, err
		}
		s.expiry = token.Expiry
	}

	return token, nil
}

func LoadToken() (token *oauth2.Token, err error) {
	pf, err := config.Load("sdp.token",
		config.SetAppName("geneos"),
		config.SetFileExtension("json"),
	)

	token = &oauth2.Token{}

	if err = pf.UnmarshalKey("token", &token,
		viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
			dc.TagName = "json"
		}),
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeHookFunc(time.RFC3339),
				expandFieldsHook(),
			),
		),
	); err != nil {
		return nil, err
	}

	return
}

var expandFieldsHook = func() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		str := data.(string)

		return config.ExpandString(str), nil
	}
}

func saveToken(token *oauth2.Token) (err error) {
	pf := config.New()

	// encrypt this token when stored
	pf.Set(pf.Join("token", "token_type"), "Zoho-oauthtoken")
	pf.Set(pf.Join("token", "expiry"), token.Expiry.Format(time.RFC3339))
	pf.Set(pf.Join("token", "expires_in"), token.ExpiresIn)

	at, err := DefaultUserKeyfile.EncodeString(host.Localhost, token.AccessToken, true)
	if err != nil {
		return err
	}
	pf.Set(pf.Join("token", "access_token"), at)

	rt, err := DefaultUserKeyfile.EncodeString(host.Localhost, token.RefreshToken, true)
	if err != nil {
		return err
	}
	pf.Set(pf.Join("token", "refresh_token"), rt)

	return pf.Save("sdp.token",
		config.SetAppName("geneos"),
		config.SetFileExtension("json"),
	)
}

// InitialAuth
//
// The oauth2/clientcredentials package tries to use the code twice, once to get the token
// and once to refresh it, which fails. So we have to do this manually.
func InitialAuth(cf *config.Config, code *config.Plaintext) (tok *oauth2.Token, err error) {
	var tcc *tls.Config

	clientID := cf.GetString("client-id")
	clientSecret := cf.GetPassword("client-secret")

	if clientID == "" || clientSecret.IsNil() {
		return nil, fmt.Errorf("client-id and/or client-secret are not valid")
	}

	log.Debug().Msgf("using client ID %s", clientID)
	log.Debug().Msgf("using client secret %s", clientSecret.String())

	if code == nil || code.IsNil() {
		err = fmt.Errorf("authorization code is required for initial authentication")
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

	conf := &Config{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret.String(),
			Endpoint: oauth2.Endpoint{
				TokenURL: auth.JoinPath("/oauth/v2/token").String(),
			},
			RedirectURL: "https://www.zoho.com",
		},
		Code: code,
	}

	params := make(url.Values)
	params.Set("code", conf.Code.String())
	params.Set("client_id", conf.ClientID)
	params.Set("client_secret", conf.ClientSecret)
	params.Set("redirect_uri", conf.RedirectURL)
	params.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", auth.JoinPath("/oauth/v2/token").String(), strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Body = io.NopCloser(strings.NewReader(params.Encode()))

	log.Debug().Msgf("requesting OAuth2 token from %s", req.URL.String())
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve OAuth2 token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respDump, _ := httputil.DumpResponse(resp, true)
		return nil, fmt.Errorf("failed to retrieve OAuth2 token, status %s\n%s", resp.Status, string(respDump))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OAuth2 token response body: %v", err)
	}

	tok = &oauth2.Token{}
	json.Unmarshal(body, &tok)
	tok.Expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	log.Debug().Msgf("received OAuth2 token: %+v", tok)

	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token received from OAuth2 token response")
	}

	if err = saveToken(tok); err != nil {
		return nil, err
	}

	log.Info().Msgf("saved OAuth2 token persistently")

	return
}
