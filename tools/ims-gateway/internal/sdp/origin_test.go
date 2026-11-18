package sdp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
)

// originStubs is an API server that redirects, which can also be the
// token endpoint, a token endpoint on another server, and a second
// server over http that records what it is sent
type originStubs struct {
	api, token, other *httptest.Server

	mu        sync.Mutex
	tokens    int
	apiAuth   []string
	otherAuth []string
}

func (o *originStubs) seen(list *[]string) []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), *list...)
}

func newOriginStubs(t *testing.T, tlsAPI, tokenOnAPI bool) *originStubs {
	t.Helper()
	o := &originStubs{}

	o.other = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.mu.Lock()
		o.otherAuth = append(o.otherAuth, r.Header.Get("Authorization"))
		o.mu.Unlock()
	}))
	t.Cleanup(o.other.Close)

	tokenHandler := func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh" {
			t.Errorf("token request form = %v, want a refresh", r.Form)
		}
		o.mu.Lock()
		o.tokens++
		o.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"fresh","token_type":"Zoho-oauthtoken","expires_in":3600}`)
	}

	o.token = httptest.NewServer(http.HandlerFunc(tokenHandler))
	t.Cleanup(o.token.Close)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/v2/token" {
			tokenHandler(w, r)
			return
		}
		o.mu.Lock()
		o.apiAuth = append(o.apiAuth, r.Header.Get("Authorization"))
		o.mu.Unlock()
		if to := r.URL.Query().Get("to"); to != "" {
			var code int
			fmt.Sscan(r.URL.Query().Get("code"), &code)
			http.Redirect(w, r, to, code)
		}
	})
	if tlsAPI {
		o.api = httptest.NewTLSServer(h)
	} else {
		o.api = httptest.NewServer(h)
	}
	t.Cleanup(o.api.Close)
	return o
}

func (o *originStubs) redirect(to string, code int) string {
	return fmt.Sprintf("%s/api/v3/requests?code=%d&to=%s", o.api.URL, code, url.QueryEscape(to))
}

// httpClient returns the HTTP client of a client for the stubs, with an
// expired token that has to be refreshed
func (o *originStubs) httpClient(t *testing.T, tokenOnAPI, follow bool) *http.Client {
	t.Helper()
	useToken(t, &oauth2.Token{AccessToken: "old", RefreshToken: "refresh", TokenType: "Zoho-oauthtoken", Expiry: time.Now().Add(-time.Hour)}, nil)
	origSaver := tokenSaver
	tokenSaver = func(*oauth2.Token) error { return nil }
	t.Cleanup(func() { tokenSaver = origSaver })

	cf := testConfig(o.api.URL)
	config.Set(cf, "sdp.datacentres.t.api", o.api.URL)
	if tokenOnAPI {
		config.Set(cf, "sdp.datacentres.t.auth", o.api.URL)
	} else {
		config.Set(cf, "sdp.datacentres.t.auth", o.token.URL)
	}
	config.Set(cf, "sdp.tls.skip-verify", true)
	config.Set(cf, "sdp.timeout", "5s")

	c, err := newClient(t.Context(), ims.Section(cf, "sdp"))
	if err != nil {
		t.Fatal(err)
	}
	if follow {
		// the client never follows redirects, but the token must still
		// stay with the API if that is ever changed
		c.HTTPClient.CheckRedirect = nil
	}
	return c.HTTPClient
}

func fetch(t *testing.T, hc *http.Client, target string) *http.Response {
	t.Helper()
	resp, err := hc.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func TestOAuthClientDoesNotFollowRedirects(t *testing.T) {
	o := newOriginStubs(t, false, false)
	hc := o.httpClient(t, false, false)

	resp := fetch(t, hc, o.redirect(o.other.URL, http.StatusTemporaryRedirect))
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want the redirect to be returned", resp.StatusCode)
	}
	if seen := o.seen(&o.otherAuth); len(seen) != 0 {
		t.Errorf("other server was sent %q", seen)
	}
	if seen := o.seen(&o.apiAuth); len(seen) != 1 || seen[0] != "Zoho-oauthtoken fresh" {
		t.Errorf("API was sent %q, want the refreshed token", seen)
	}
}

func TestOAuthTokenStaysWithAPI(t *testing.T) {
	for _, tlsAPI := range []bool{false, true} {
		for _, tokenOnAPI := range []bool{false, true} {
			for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
				t.Run(fmt.Sprintf("tls=%t/token-on-api=%t/%d", tlsAPI, tokenOnAPI, code), func(t *testing.T) {
					o := newOriginStubs(t, tlsAPI, tokenOnAPI)
					hc := o.httpClient(t, tokenOnAPI, true)

					// another port, and for TLS also from https to http
					fetch(t, hc, o.redirect(o.other.URL+"/other", code))

					if seen := o.seen(&o.otherAuth); len(seen) != 1 || seen[0] != "" {
						t.Errorf("other server was sent %q, want one request without a token", seen)
					}
					if seen := o.seen(&o.apiAuth); len(seen) != 1 || seen[0] != "Zoho-oauthtoken fresh" {
						t.Errorf("API was sent %q, want the refreshed token", seen)
					}
					o.mu.Lock()
					tokens := o.tokens
					o.mu.Unlock()
					if tokens != 1 {
						t.Errorf("token requests = %d, want 1 refresh", tokens)
					}
				})
			}
		}
	}
}

func TestOAuthSameOriginRedirectKeepsToken(t *testing.T) {
	o := newOriginStubs(t, false, false)
	hc := o.httpClient(t, false, true)

	fetch(t, hc, o.redirect(o.api.URL+"/api/v3/other", http.StatusTemporaryRedirect))
	if seen := o.seen(&o.apiAuth); len(seen) != 2 || strings.Contains(strings.Join(seen, ""), "old") || seen[0] != "Zoho-oauthtoken fresh" || seen[1] != "Zoho-oauthtoken fresh" {
		t.Errorf("API was sent %q, want the token on both requests", seen)
	}
}
