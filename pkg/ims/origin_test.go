package ims

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestSameOrigin(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"https://snow.example.com", "https://snow.example.com/api/x?y=1", true},
		{"https://snow.example.com", "https://SNOW.example.com:443/", true},
		{"http://snow.example.com", "http://snow.example.com:80", true},
		{"https://snow.example.com", "https://snow.example.com:8443", false},
		{"https://snow.example.com", "http://snow.example.com", false},
		{"https://snow.example.com", "https://other.example.com", false},
		{"http://127.0.0.1:1000", "http://127.0.0.1:1001", false},
	}
	for _, test := range tests {
		if got := SameOrigin(mustURL(t, test.a), mustURL(t, test.b)); got != test.want {
			t.Errorf("SameOrigin(%q, %q) = %v, want %v", test.a, test.b, got, test.want)
		}
	}
}

type recorder struct {
	mu   sync.Mutex
	reqs []*http.Request
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}

func TestOriginOnly(t *testing.T) {
	authed, plain := &recorder{}, &recorder{}
	rt := OriginOnly(mustURL(t, "https://snow.example.com"), authed, plain)

	for _, target := range []string{"https://snow.example.com/a", "https://snow.example.com:8443/a", "http://snow.example.com/a", "https://other.example.com/a"} {
		req, _ := http.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer secret")
		req.Header.Set("Proxy-Authorization", "Basic secret")
		if _, err := rt.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
		// the caller's request is never changed
		if req.Header.Get("Authorization") == "" {
			t.Errorf("%s: header removed from the caller's request", target)
		}
	}

	if len(authed.reqs) != 1 || authed.reqs[0].URL.Path != "/a" || authed.reqs[0].URL.Host != "snow.example.com" {
		t.Errorf("authed requests = %v, want only the same origin", authed.reqs)
	}
	if len(plain.reqs) != 3 {
		t.Fatalf("plain requests = %d, want 3", len(plain.reqs))
	}
	for _, req := range plain.reqs {
		if req.Header.Get("Authorization") != "" || req.Header.Get("Proxy-Authorization") != "" {
			t.Errorf("%s: credentials sent to another origin", req.URL)
		}
	}
}

// seen records the headers of requests to a server
type seen struct {
	mu    sync.Mutex
	auth  []string
	extra []string
}

func (s *seen) handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.auth = append(s.auth, r.Header.Get("Authorization"))
		s.extra = append(s.extra, r.Header.Get("X-Api-Key"))
		s.mu.Unlock()
		next(w, r)
	}
}

func redirectTo(target func() string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target()+"/next", code)
	}
}

func ok(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func TestSameOriginRedirects(t *testing.T) {
	for _, code := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			var a, b seen
			srvB := httptest.NewServer(b.handler(ok))
			defer srvB.Close()
			srvA := httptest.NewServer(a.handler(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/next" {
					return
				}
				// a redirect to the same server is followed with the headers, and
				// then one to another port is not
				if r.URL.Query().Get("other") != "" {
					http.Redirect(w, r, srvB.URL+"/next", code)
					return
				}
				http.Redirect(w, r, "/next", code)
			}))
			defer srvA.Close()

			c := &http.Client{}
			SameOriginRedirects(c, mustURL(t, srvA.URL), "X-Api-Key")

			for _, q := range []string{"", "?other=1"} {
				req, _ := http.NewRequest(http.MethodGet, srvA.URL+"/start"+q, nil)
				req.Header.Set("Authorization", "Bearer secret")
				req.Header.Set("X-Api-Key", "key")
				resp, err := c.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
			}

			// start, same-origin next, start, and the redirect to B
			if len(a.auth) != 3 || a.auth[0] != "Bearer secret" || a.auth[1] != "Bearer secret" || a.auth[2] != "Bearer secret" || a.extra[1] != "key" {
				t.Errorf("A saw auth %q extra %q, want all with credentials", a.auth, a.extra)
			}
			if len(b.auth) != 1 || b.auth[0] != "" || b.extra[0] != "" {
				t.Errorf("B saw auth %q extra %q, want one request without credentials", b.auth, b.extra)
			}
		})
	}
}

func TestSameOriginRedirectsRefusesHTTPS(t *testing.T) {
	var b seen
	srvB := httptest.NewServer(b.handler(ok))
	defer srvB.Close()
	srvA := httptest.NewTLSServer(redirectTo(func() string { return srvB.URL }, http.StatusTemporaryRedirect))
	defer srvA.Close()

	c := srvA.Client()
	SameOriginRedirects(c, mustURL(t, srvA.URL))

	req, _ := http.NewRequest(http.MethodGet, srvA.URL, nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := c.Do(req)
	if err == nil {
		resp.Body.Close()
	}
	if !errors.Is(err, ErrInsecureRedirect) {
		t.Errorf("err = %v, want ErrInsecureRedirect", err)
	}
	if len(b.auth) != 0 {
		t.Errorf("http server received %d requests, want none", len(b.auth))
	}
}

func TestSameOriginRedirectsKeepsCheckRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/again", http.StatusFound)
	}))
	defer srv.Close()

	// the default limit applies without a CheckRedirect
	c := &http.Client{}
	SameOriginRedirects(c, mustURL(t, srv.URL))
	if resp, err := c.Get(srv.URL); err == nil {
		resp.Body.Close()
		t.Error("no error for a redirect loop")
	}

	// and the existing one is used
	var calls int
	c = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		calls++
		return http.ErrUseLastResponse
	}}
	SameOriginRedirects(c, mustURL(t, srv.URL))
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls != 1 || resp.StatusCode != http.StatusFound {
		t.Errorf("calls = %d status = %d, want the existing CheckRedirect to stop at the redirect", calls, resp.StatusCode)
	}
}
