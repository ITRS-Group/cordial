package snow

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

// origins is an instance that issues tokens and redirects, and another
// server, over http, that records what it is sent
type origins struct {
	a, b   *httptest.Server
	mu     sync.Mutex
	tokens int
	aAuth  []string
	bAuth  []string
}

func (o *origins) bSeen() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.bAuth...)
}

func (o *origins) aSeen() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.aAuth...)
}

func newOrigins(t *testing.T, tlsA bool) *origins {
	t.Helper()
	o := &origins{}
	o.b = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.mu.Lock()
		o.bAuth = append(o.bAuth, r.Header.Get("Authorization"))
		o.mu.Unlock()
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	t.Cleanup(o.b.Close)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth_token.do" {
			o.mu.Lock()
			o.tokens++
			n := o.tokens
			o.mu.Unlock()
			if r.Header.Get("Authorization") == "Bearer" || strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Errorf("token request sent a bearer token: %q", r.Header.Get("Authorization"))
			}
			// expired at once, so that each request refreshes the token
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"access_token":"tok%d","token_type":"Bearer","expires_in":1}`, n)
			return
		}
		o.mu.Lock()
		o.aAuth = append(o.aAuth, r.Header.Get("Authorization"))
		o.mu.Unlock()
		if to := r.URL.Query().Get("to"); to != "" {
			var code int
			fmt.Sscan(r.URL.Query().Get("code"), &code)
			http.Redirect(w, r, to, code)
			return
		}
		_, _ = io.WriteString(w, `{"result":[]}`)
	})
	if tlsA {
		o.a = httptest.NewTLSServer(h)
	} else {
		o.a = httptest.NewServer(h)
	}
	t.Cleanup(o.a.Close)
	return o
}

func (o *origins) redirect(to string, code int) string {
	return fmt.Sprintf("%s/api/now/v2/table/incident?code=%d&to=%s", o.a.URL, code, url.QueryEscape(to))
}

func oauthHTTPClient(t *testing.T, o *origins, follow bool) *http.Client {
	t.Helper()
	cf := testConfig(o.a.URL)
	config.Set(cf, "client-id", "id")
	config.Set(cf, "client-secret", "secret")
	config.Set(cf, "tls.skip-verify", true)
	c, err := New(cf)
	if err != nil {
		t.Fatal(err)
	}
	hc := c.HTTPClient
	if follow {
		// the client never follows redirects, but the token must still
		// stay with the instance if that is ever changed
		hc.CheckRedirect = nil
	}
	return hc
}

func get(t *testing.T, hc *http.Client, target string) *http.Response {
	t.Helper()
	resp, err := hc.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp
}

func TestOAuthClientDoesNotFollowRedirects(t *testing.T) {
	o := newOrigins(t, false)
	hc := oauthHTTPClient(t, o, false)

	resp := get(t, hc, o.redirect(o.b.URL, http.StatusTemporaryRedirect))
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want the redirect to be returned", resp.StatusCode)
	}
	if seen := o.bSeen(); len(seen) != 0 {
		t.Errorf("other server was sent %q", seen)
	}
	if seen := o.aSeen(); len(seen) != 1 || !strings.HasPrefix(seen[0], "Bearer tok") {
		t.Errorf("instance was sent %q, want a token", seen)
	}
}

func TestOAuthTokenStaysWithInstance(t *testing.T) {
	for _, tlsA := range []bool{false, true} {
		for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
			t.Run(fmt.Sprintf("tls=%t/%d", tlsA, code), func(t *testing.T) {
				o := newOrigins(t, tlsA)
				hc := oauthHTTPClient(t, o, true)

				// another port, and for TLS also from https to http
				get(t, hc, o.redirect(o.b.URL+"/other", code))
				if seen := o.bSeen(); len(seen) != 1 || seen[0] != "" {
					t.Errorf("other server was sent %q, want one request without a token", seen)
				}
				if seen := o.aSeen(); len(seen) != 1 || !strings.HasPrefix(seen[0], "Bearer tok") {
					t.Errorf("instance was sent %q, want a token", seen)
				}

				// the token is refreshed, as it has expired, but only for the instance
				get(t, hc, o.a.URL+"/api/now/v2/table/incident")
				o.mu.Lock()
				tokens := o.tokens
				o.mu.Unlock()
				if tokens < 2 {
					t.Errorf("token requests = %d, want a refresh", tokens)
				}
				if seen := o.aSeen(); len(seen) != 2 || !strings.HasPrefix(seen[1], "Bearer tok") {
					t.Errorf("instance was sent %q, want a token for each request", seen)
				}
			})
		}
	}
}

func TestOAuthSameOriginRedirectKeepsToken(t *testing.T) {
	o := newOrigins(t, false)
	hc := oauthHTTPClient(t, o, true)

	get(t, hc, o.redirect(o.a.URL+"/api/now/v2/table/other", http.StatusTemporaryRedirect))
	if seen := o.aSeen(); len(seen) != 2 || !strings.HasPrefix(seen[0], "Bearer tok") || !strings.HasPrefix(seen[1], "Bearer tok") {
		t.Errorf("instance was sent %q, want a token on both requests", seen)
	}
}
