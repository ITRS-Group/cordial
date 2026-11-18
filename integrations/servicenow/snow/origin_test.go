package snow

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/itrs-group/cordial/pkg/ims"
)

// recorder is a server that records the Authorization header of the
// requests it is sent, and redirects those with a `to` query
type recorder struct {
	*httptest.Server
	mu    sync.Mutex
	auth  []string
	token int
}

func (r *recorder) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.auth...)
}

func newRecorder(t *testing.T, tls bool) *recorder {
	t.Helper()
	r := &recorder{}
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/oauth_token.do" {
			r.mu.Lock()
			r.token++
			r.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":3600}`))
			return
		}
		r.mu.Lock()
		r.auth = append(r.auth, req.Header.Get("Authorization"))
		r.mu.Unlock()
		if to := req.URL.Query().Get("to"); to != "" {
			var code int
			fmt.Sscan(req.URL.Query().Get("code"), &code)
			http.Redirect(w, req, to, code)
		}
	})
	if tls {
		r.Server = httptest.NewTLSServer(h)
	} else {
		r.Server = httptest.NewServer(h)
	}
	t.Cleanup(r.Close)
	return r
}

func (r *recorder) redirect(to string, code int) string {
	return fmt.Sprintf("%s/api/now/v2/table/incident?code=%d&to=%s", r.URL, code, url.QueryEscape(to))
}

func oauthFor(t *testing.T, api, tokenServer *recorder) *http.Client {
	t.Helper()
	conf := &clientcredentials.Config{ClientID: "id", ClientSecret: "secret", TokenURL: tokenServer.URL + "/oauth_token.do"}
	u, _ := url.Parse(api.URL)
	return oauthClient(context.WithValue(context.Background(), oauth2.HTTPClient, api.Client()), conf, u)
}

func do(t *testing.T, c *http.Client, target string) error {
	t.Helper()
	resp, err := c.Get(target)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func TestOAuthTokenStaysWithInstance(t *testing.T) {
	for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			// another port, with and without TLS
			a, b := newRecorder(t, true), newRecorder(t, true)
			if err := do(t, oauthFor(t, a, a), a.redirect(b.URL, code)); err != nil {
				t.Fatal(err)
			}
			if seen := b.seen(); len(seen) != 1 || seen[0] != "" {
				t.Errorf("other server was sent %q, want one request without a token", seen)
			}
			if seen := a.seen(); len(seen) != 1 || seen[0] != "Bearer tok" {
				t.Errorf("instance was sent %q, want the token", seen)
			}

			ah, bh := newRecorder(t, false), newRecorder(t, false)
			if err := do(t, oauthFor(t, ah, ah), ah.redirect(bh.URL, code)); err != nil {
				t.Fatal(err)
			}
			if seen := bh.seen(); len(seen) != 1 || seen[0] != "" {
				t.Errorf("other server was sent %q, want one request without a token", seen)
			}

			// https to http is refused
			c := newRecorder(t, false)
			if err := do(t, oauthFor(t, a, a), a.redirect(c.URL, code)); err == nil || !strings.Contains(err.Error(), ims.ErrInsecureRedirect.Error()) {
				t.Errorf("err = %v, want the redirect to http refused", err)
			}
			if seen := c.seen(); len(seen) != 0 {
				t.Errorf("http server was sent %q", seen)
			}
		})
	}
}

func TestOAuthTokenEndpointOnAnotherServer(t *testing.T) {
	a, tokens := newRecorder(t, true), newRecorder(t, true)
	if err := do(t, oauthFor(t, a, tokens), a.redirect(tokens.URL+"/x", http.StatusTemporaryRedirect)); err != nil {
		t.Fatal(err)
	}
	tokens.mu.Lock()
	n := tokens.token
	tokens.mu.Unlock()
	if n != 1 {
		t.Errorf("token requests = %d, want 1", n)
	}
	if seen := tokens.seen(); len(seen) != 1 || seen[0] != "" {
		t.Errorf("token server was sent %q for the redirect, want no token", seen)
	}
	if seen := a.seen(); len(seen) != 1 || seen[0] != "Bearer tok" {
		t.Errorf("instance was sent %q, want the token", seen)
	}
}

func TestOAuthSameOriginRedirectKeepsToken(t *testing.T) {
	a := newRecorder(t, true)
	if err := do(t, oauthFor(t, a, a), a.redirect(a.URL+"/api/now/v2/table/other", http.StatusTemporaryRedirect)); err != nil {
		t.Fatal(err)
	}
	if seen := a.seen(); len(seen) != 2 || seen[0] != "Bearer tok" || seen[1] != "Bearer tok" {
		t.Errorf("instance was sent %q, want the token on both requests", seen)
	}
}

func TestBasicAuthStaysWithInstance(t *testing.T) {
	a, b, plain := newRecorder(t, true), newRecorder(t, true), newRecorder(t, false)
	u, _ := url.Parse(a.URL)
	c := basicClient(u)
	c.Transport = a.Client().Transport

	get := func(target string) error {
		req, _ := http.NewRequest(http.MethodGet, target, nil)
		req.SetBasicAuth("user", "pass")
		resp, err := c.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}

	// the same host on another port, which http.Client alone sends it to
	if err := get(a.redirect(b.URL, http.StatusTemporaryRedirect)); err != nil {
		t.Fatal(err)
	}
	if seen := b.seen(); len(seen) != 1 || seen[0] != "" {
		t.Errorf("other server was sent %q, want one request without credentials", seen)
	}
	if err := get(a.redirect(plain.URL, http.StatusTemporaryRedirect)); err == nil {
		t.Error("redirect from https to http was followed")
	}
	if seen := plain.seen(); len(seen) != 0 {
		t.Errorf("http server was sent %q", seen)
	}
	if err := get(a.redirect(a.URL+"/same", http.StatusTemporaryRedirect)); err != nil {
		t.Fatal(err)
	}
	if seen := a.seen(); len(seen) != 4 || seen[3] != "Basic dXNlcjpwYXNz" {
		t.Errorf("instance was sent %q, want the credentials on the same-origin redirect", seen)
	}
}
