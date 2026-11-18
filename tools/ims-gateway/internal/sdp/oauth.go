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
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// DefaultUserKeyfile is the path to the user's key file as a
// config.Keyfile type
var DefaultUserKeyfile = config.KeyFile(
	config.Path("keyfile",
		config.AppName("geneos"), // we use the geneos keyfile, not a separate one for this integration
		config.Format("aes"),
		config.SkipWorkingDir(),
	),
)

// tokenLoader loads the saved token and can be replaced in tests
var tokenLoader = loadToken

// redirectURL is the redirect URL registered for the ServiceDesk Plus
// client, which is not used as the authorization code is entered by hand
const redirectURL = "https://www.zoho.com"

// tokenDirOptions limit the token file to the user configuration
// directory, which is where [saveToken] writes it, so that a file in the
// working directory, or elsewhere, never takes precedence over the
// saved token
var tokenDirOptions = []config.FileOption{
	config.AppName("geneos"),
	config.Format("json"),
	config.SkipWorkingDir(),
	config.SkipSystemDir(),
}

// loadToken reads the token saved by [saveToken], which decrypts any
// encoded values, and returns an error if there is no usable token.
func loadToken() (token *oauth2.Token, err error) {
	pf, err := config.Read("sdp.token", append([]config.FileOption{config.MustExist()}, tokenDirOptions...)...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errNoToken, err)
	}

	token = &oauth2.Token{}
	if err = pf.UnmarshalKey("token", token); err != nil {
		return nil, fmt.Errorf("cannot read saved ServiceDesk Plus token: %w", err)
	}

	if token.RefreshToken == "" {
		return nil, fmt.Errorf("%w (the saved token has no refresh token, or it could not be decrypted)", errNoToken)
	}

	return
}

// tokenSaver saves a token and can be replaced in tests
var tokenSaver = saveToken

func saveToken(token *oauth2.Token) (err error) {
	pf := config.New()

	// encrypt this token when stored
	config.Set(pf, pf.Join("token", "token_type"), "Zoho-oauthtoken")
	config.Set(pf, pf.Join("token", "expiry"), token.Expiry.Format(time.RFC3339))
	config.Set(pf, pf.Join("token", "expires_in"), token.ExpiresIn)

	at, err := DefaultUserKeyfile.EncodeString(host.Localhost, token.AccessToken, true)
	if err != nil {
		return err
	}
	config.Set(pf, pf.Join("token", "access_token"), at)

	rt, err := DefaultUserKeyfile.EncodeString(host.Localhost, token.RefreshToken, true)
	if err != nil {
		return err
	}
	config.Set(pf, pf.Join("token", "refresh_token"), rt)

	// the values are encrypted, but the file is only for the user. The
	// file is created here, with the permissions wanted, as an existing
	// file keeps its permissions when it is written
	dir, err := config.UserConfigPath("geneos")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	p := path.Join(dir, "sdp.token.json")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	f.Close()
	if err = os.Chmod(p, 0o600); err != nil {
		return err
	}

	return pf.Write("sdp.token", config.FilePath(p), config.Format("json"))
}

// InitialAuth
//
// The oauth2/clientcredentials package tries to use the code twice, once to get the token
// and once to refresh it, which fails. So we have to do this manually.
func InitialAuth(sdpCf *config.Config, code config.Secret) (tok *oauth2.Token, err error) {
	var tcc *tls.Config

	clientID := config.Get[string](sdpCf, "client-id")
	clientSecret := config.Get[config.Secret](sdpCf, "client-secret")

	if clientID == "" || len(clientSecret) == 0 {
		return nil, fmt.Errorf("client-id and/or client-secret are not valid")
	}

	if len(code) == 0 {
		err = fmt.Errorf("authorization code is required for initial authentication")
		return
	}

	auth, err := url.Parse(config.Get[string](sdpCf, sdpCf.Join("datacentres", config.Get[string](sdpCf, "datacentre"), "auth")))
	if err != nil {
		return
	}

	if auth.Scheme == "https" {
		tcc = &tls.Config{
			InsecureSkipVerify: config.Get[bool](sdpCf, sdpCf.Join("tls", "skip-verify")),
		}
	}

	timeout := config.Get[time.Duration](sdpCf, sdpCf.Join("timeout"))
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
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	params := make(url.Values)
	params.Set("code", string(code))
	params.Set("client_id", clientID)
	params.Set("client_secret", string(clientSecret))
	params.Set("redirect_uri", redirectURL)
	params.Set("grant_type", "authorization_code")

	// there is no context for the command
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, auth.JoinPath("/oauth/v2/token").String(), strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Body = io.NopCloser(strings.NewReader(params.Encode()))

	log.Debug("requesting OAuth2 token", slog.String("url", req.URL.String()))
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
	if err = json.Unmarshal(body, tok); err != nil {
		return nil, fmt.Errorf("failed to decode OAuth2 token response: %w", err)
	}
	tok.Expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	// do not log the token itself
	log.Debug("received OAuth2 token", slog.Time("expiry", tok.Expiry), slog.Bool("refresh_token", tok.RefreshToken != ""))

	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token received from OAuth2 token response")
	}

	if err = tokenSaver(tok); err != nil {
		return nil, err
	}

	log.Info("saved OAuth2 token persistently")

	return
}
