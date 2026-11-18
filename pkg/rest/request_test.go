package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// recorded is a request as seen by the test server
type recorded struct {
	Method string
	Path   string
	Query  string
	Body   string
	Header http.Header
}

// recordingServer returns a server that records every request and
// replies with JSON {"ok":true}
func recordingServer(t *testing.T) (*httptest.Server, func() []recorded) {
	t.Helper()
	var mu sync.Mutex
	var reqs []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		reqs = append(reqs, recorded{r.Method, r.URL.Path, r.URL.RawQuery, string(b), r.Header.Clone()})
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []recorded {
		mu.Lock()
		defer mu.Unlock()
		return append([]recorded(nil), reqs...)
	}
}

func TestEncodeBody(t *testing.T) {
	srv, got := recordingServer(t)
	c := NewClient(BaseURLString(srv.URL))
	ctx := context.Background()

	// not encodable as JSON: must fail and send nothing
	for _, call := range []func() error{
		func() error { _, err := c.Post(ctx, "/x", map[string]any{"c": make(chan int)}, nil); return err },
		func() error { _, err := c.Put(ctx, "/x", func() {}, nil); return err },
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "encoding request body") {
			t.Errorf("error = %v, want an encoding error", err)
		}
	}
	if n := len(got()); n != 0 {
		t.Fatalf("%d requests sent for unencodable bodies, want 0", n)
	}

	for name, in := range map[string]any{
		"string": `{"raw":1}`, "bytes": []byte(`{"raw":1}`),
	} {
		if _, err := c.Post(ctx, "/x", in, nil); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := c.Post(ctx, "/x", map[string]int{"a": 1}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Post(ctx, "/x", nil, nil); err != nil {
		t.Fatal(err)
	}
	r := got()
	if len(r) != 4 {
		t.Fatalf("got %d requests, want 4", len(r))
	}
	for i, want := range []string{`{"raw":1}`, `{"raw":1}`, `{"a":1}`, ``} {
		if r[i].Body != want {
			t.Errorf("request %d body = %q, want %q", i, r[i].Body, want)
		}
		if ct := r[i].Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("request %d content-type = %q", i, ct)
		}
	}
}

func TestEndpointsAndQueries(t *testing.T) {
	srv, got := recordingServer(t)
	type params struct {
		A string `url:"a,omitempty"`
		B int    `url:"b,omitempty"`
	}
	mustURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	var typedNil *params

	tests := []struct {
		name      string
		base      string
		endpoint  any
		request   any
		wantPath  string
		wantQuery string
	}{
		{"leading slash", "", "/items", nil, "/items", ""},
		{"no slash", "", "items", nil, "/items", ""},
		{"trailing slash", "", "items/", nil, "/items/", ""},
		{"base path prefix", "/api/v2", "/items", nil, "/api/v2/items", ""},
		{"url endpoint", "", mustURL("/items?x=1"), nil, "/items", "x=1"},
		{"query in string endpoint", "", "/items?x=1&y=2", nil, "/items", "x=1&y=2"},
		{"struct request", "", "/items", params{A: "z", B: 2}, "/items", "a=z&b=2"},
		{"struct pointer request", "", "/items", &params{A: "z"}, "/items", "a=z"},
		{"typed nil request", "", "/items", typedNil, "/items", ""},
		{"string request", "", "/items", "q=1&r=2", "/items", "q=1&r=2"},
		{"bytes request", "", "/items", []byte("q=1"), "/items", "q=1"},
		{"url.Values request", "", "/items", url.Values{"q": {"a b"}}, "/items", "q=a+b"},
		{"merged with endpoint query", "", "/items?x=1", params{A: "z"}, "/items", "x=1&a=z"},
		{"escaped value", "", "/items", params{A: "a&b=c"}, "/items", "a=a%26b%3Dc"},
	}
	for _, tt := range tests {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			t.Run(tt.name+"/"+method, func(t *testing.T) {
				before := len(got())
				c := NewClient(BaseURLString(srv.URL + tt.base))
				if _, err := c.Do(context.Background(), method, tt.endpoint, tt.request, nil); err != nil {
					t.Fatal(err)
				}
				r := got()
				if len(r) != before+1 {
					t.Fatalf("expected one request, got %d", len(r)-before)
				}
				last := r[len(r)-1]
				if last.Method != method || last.Path != tt.wantPath || last.Query != tt.wantQuery {
					t.Errorf("request = %s %s?%s, want %s %s?%s", last.Method, last.Path, last.Query, method, tt.wantPath, tt.wantQuery)
				}
			})
		}
	}

	// not something that can be a query
	if _, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), "/x", 42, nil); err == nil {
		t.Error("an int request should not be accepted as a query")
	}
	if _, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), nil, nil, nil); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("nil endpoint error = %v, want ErrUnsupported", err)
	}
	var nilURL *url.URL
	if _, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), nilURL, nil, nil); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("nil *url.URL error = %v, want ErrUnsupported", err)
	}
	if _, err := NewClient().Do(context.Background(), "PATCH", "/x", nil, nil); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("unsupported method error = %v", err)
	}
}

func TestStatusHandling(t *testing.T) {
	statuses := []int{200, 201, 202, 204, 301, 400, 401, 403, 404, 409, 429, 500, 502, 503, 504}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var code int
		fmt.Sscanf(r.URL.Path, "/%d", &code)
		if code == 301 {
			// no Location, so there is nothing to follow
			w.WriteHeader(code)
			return
		}
		if code >= 400 {
			http.Error(w, fmt.Sprintf("failed with %d", code), code)
			return
		}
		w.WriteHeader(code)
	}))
	defer srv.Close()
	c := NewClient(BaseURLString(srv.URL))

	for _, code := range statuses {
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
			t.Run(fmt.Sprintf("%d/%s", code, method), func(t *testing.T) {
				resp, err := c.Do(context.Background(), method, fmt.Sprintf("/%d", code), nil, nil)
				if resp == nil {
					t.Fatalf("no response returned, err = %v", err)
				}
				if resp.StatusCode != code {
					t.Fatalf("status = %d", resp.StatusCode)
				}
				if code >= 200 && code <= 299 {
					if err != nil {
						t.Fatalf("error = %v, want success", err)
					}
					return
				}
				var he *HTTPError
				if !errors.As(err, &he) || he.StatusCode != code {
					t.Fatalf("error = %v, want *HTTPError with status %d", err, code)
				}
				if code >= 400 && !strings.Contains(string(he.Body), fmt.Sprintf("failed with %d", code)) {
					t.Errorf("body = %q", he.Body)
				}
			})
		}
	}
}

func TestHTTPErrorText(t *testing.T) {
	for name, tt := range map[string]struct {
		err  *HTTPError
		want string
	}{
		"status and body":  {&HTTPError{Status: "400 Bad Request", Body: []byte("bad\n")}, "400 Bad Request bad"},
		"empty body":       {&HTTPError{Status: "404 Not Found"}, "404 Not Found"},
		"no status string": {&HTTPError{StatusCode: 503}, "503 Service Unavailable"},
		"only whitespace":  {&HTTPError{Status: "500 Internal Server Error", Body: []byte(" \n")}, "500 Internal Server Error"},
	} {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("%s: Error() = %q, want %q", name, got, tt.want)
		}
	}

	long := bytes.Repeat([]byte("é"), 2000)
	e := &HTTPError{Status: "500 X", Body: long}
	if s := e.Error(); len(s) > maxErrorText+20 || !strings.HasSuffix(s, "...") {
		t.Errorf("long body not truncated, len = %d", len(s))
	}
	if len(e.Body) != len(long) {
		t.Error("Body must stay intact")
	}
}

func TestErrorBodyIsCapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 4*maxErrorBody))
	}))
	defer srv.Close()
	_, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), "/", nil, nil)
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("error = %v", err)
	}
	if len(he.Body) != maxErrorBody {
		t.Errorf("body length = %d, want %d", len(he.Body), maxErrorBody)
	}
}

func TestTransportFailures(t *testing.T) {
	methods := map[string]func(c *Client, ctx context.Context) error{
		"get":    func(c *Client, ctx context.Context) error { _, err := c.Get(ctx, "/x", nil, nil); return err },
		"post":   func(c *Client, ctx context.Context) error { _, err := c.Post(ctx, "/x", "{}", nil); return err },
		"put":    func(c *Client, ctx context.Context) error { _, err := c.Put(ctx, "/x", "{}", nil); return err },
		"delete": func(c *Client, ctx context.Context) error { _, err := c.Delete(ctx, "/x", nil); return err },
	}

	t.Run("refused", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		u := srv.URL
		srv.Close()
		for name, call := range methods {
			err := call(NewClient(BaseURLString(u)), context.Background())
			var he *HTTPError
			var ue *url.Error
			if err == nil || errors.As(err, &he) || !errors.As(err, &ue) {
				t.Errorf("%s: error = %v, want a *url.Error that is not an HTTPError", name, err)
			}
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		srv, got := recordingServer(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		for name, call := range methods {
			if err := call(NewClient(BaseURLString(srv.URL)), ctx); !errors.Is(err, context.Canceled) {
				t.Errorf("%s: error = %v, want context.Canceled", name, err)
			}
		}
		if n := len(got()); n != 0 {
			t.Errorf("%d requests made with a cancelled context", n)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// read the body so that the server notices the client going away
			_, _ = io.Copy(io.Discard, r.Body)
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		}))
		defer srv.Close()
		c := NewClient(BaseURLString(srv.URL), HTTPClient(&http.Client{Timeout: 50 * time.Millisecond}))
		for name, call := range methods {
			err := call(c, context.Background())
			var ue *url.Error
			if !errors.As(err, &ue) || !ue.Timeout() {
				t.Errorf("%s: error = %v, want a timeout", name, err)
			}
		}
	})

	t.Run("connection dropped mid body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err != nil {
				return
			}
			defer conn.Close()
			_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"a\":")
			_ = buf.Flush()
		}))
		defer srv.Close()
		var m map[string]any
		_, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), "/", nil, &m)
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("error = %v, want unexpected EOF", err)
		}
	})

	t.Run("no response at all", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()
		_, err = NewClient(BaseURLString("http://"+l.Addr().String())).Get(context.Background(), "/", nil, nil)
		if err == nil {
			t.Error("a closed connection must be an error")
		}
	})
}

func TestRedirects(t *testing.T) {
	var mu sync.Mutex
	hits := map[string][]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		hits[r.URL.Path] = append(hits[r.URL.Path], r.Method+" "+string(b))
		mu.Unlock()
		switch r.URL.Path {
		case "/302":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/307":
			http.Redirect(w, r, "/target", http.StatusTemporaryRedirect)
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer srv.Close()
	hitsFor := func(path string) []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), hits[path]...)
	}
	reset := func() {
		mu.Lock()
		clear(hits)
		mu.Unlock()
	}

	t.Run("POST 302 is not reported as success", func(t *testing.T) {
		reset()
		var m map[string]any
		_, err := NewClient(BaseURLString(srv.URL)).Post(context.Background(), "/302", `{"a":1}`, &m)
		var re *RedirectError
		if !errors.As(err, &re) {
			t.Fatalf("error = %v, want *RedirectError", err)
		}
		if re.Method != "POST" || re.FinalMethod != "GET" || re.StatusCode != 302 || !strings.HasSuffix(re.FinalURL, "/target") {
			t.Errorf("RedirectError = %+v", re)
		}
		if m != nil {
			t.Error("response was decoded from a redirected request")
		}
	})

	t.Run("PUT 302 is not reported as success", func(t *testing.T) {
		_, err := NewClient(BaseURLString(srv.URL)).Put(context.Background(), "/302", `{}`, nil)
		if _, ok := errors.AsType[*RedirectError](err); !ok {
			t.Fatalf("error = %v, want *RedirectError", err)
		}
	})

	t.Run("POST 307 keeps method and body", func(t *testing.T) {
		reset()
		var m map[string]any
		if _, err := NewClient(BaseURLString(srv.URL)).Post(context.Background(), "/307", `{"a":1}`, &m); err != nil {
			t.Fatal(err)
		}
		if h := hitsFor("/target"); len(h) != 1 || h[0] != `POST {"a":1}` {
			t.Errorf("target hits = %q", h)
		}
	})

	t.Run("GET redirect is followed", func(t *testing.T) {
		var m map[string]any
		if _, err := NewClient(BaseURLString(srv.URL)).Get(context.Background(), "/302", nil, &m); err != nil || m["ok"] != true {
			t.Errorf("m = %v, err = %v", m, err)
		}
	})

	t.Run("NoRedirect", func(t *testing.T) {
		reset()
		shared := &http.Client{}
		c := NewClient(BaseURLString(srv.URL), HTTPClient(shared), NoRedirect())
		for _, call := range []func() error{
			func() error { _, err := c.Get(context.Background(), "/302", nil, nil); return err },
			func() error { _, err := c.Post(context.Background(), "/307", "{}", nil); return err },
		} {
			var he *HTTPError
			if err := call(); !errors.As(err, &he) || (he.StatusCode != 302 && he.StatusCode != 307) {
				t.Errorf("error = %v, want a redirect *HTTPError", err)
			}
		}
		if h := hitsFor("/target"); len(h) != 0 {
			t.Errorf("redirect was followed: %q", h)
		}
		if shared.CheckRedirect != nil {
			t.Error("NoRedirect modified the supplied http.Client")
		}
	})
}

func TestHooksAndAuthHeader(t *testing.T) {
	srv, got := recordingServer(t)
	var bodies [][]byte
	var methods []string
	c := NewClient(BaseURLString(srv.URL), SetupRequestFunc(func(req *http.Request, c *Client, body []byte) {
		methods = append(methods, req.Method)
		bodies = append(bodies, body)
		if req.Method == http.MethodPut {
			req.Header.Set("Content-Type", "text/plain")
		}
		req.SetBasicAuth("u", "p")
	}))
	c.SetAuth("X-Token", "abc")
	ctx := context.Background()
	_, _ = c.Get(ctx, "/x", nil, nil)
	_, _ = c.Post(ctx, "/x", `{"p":1}`, nil)
	_, _ = c.Put(ctx, "/x", `{"q":1}`, nil)
	_, _ = c.Delete(ctx, "/x", nil)

	if strings.Join(methods, ",") != "GET,POST,PUT,DELETE" {
		t.Errorf("SetupRequest called for %v", methods)
	}
	if string(bodies[1]) != `{"p":1}` || string(bodies[2]) != `{"q":1}` || bodies[0] != nil || bodies[3] != nil {
		t.Errorf("SetupRequest bodies = %q", bodies)
	}
	for i, r := range got() {
		if r.Header.Get("X-Token") != "abc" {
			t.Errorf("request %d: X-Token = %q", i, r.Header.Get("X-Token"))
		}
		if u, p, ok := func() (string, string, bool) { req := &http.Request{Header: r.Header}; return req.BasicAuth() }(); !ok || u != "u" || p != "p" {
			t.Errorf("request %d: basic auth not set", i)
		}
	}
	if ct := got()[2].Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("SetupRequest could not override content-type, got %q", ct)
	}
}

func TestLogger(t *testing.T) {
	srv, _ := recordingServer(t)
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c := NewClient(BaseURLString(srv.URL), Logger(log))
	c.SetAuth("Authorization", "Bearer secret-token")
	ctx := context.Background()
	_, _ = c.Get(ctx, "/g", "a=1", nil)
	_, _ = c.Post(ctx, "/p", `{"k":"v"}`, nil)
	_, _ = c.Put(ctx, "/u", `{"k":"v"}`, nil)
	_, _ = c.Delete(ctx, "/d", nil)

	out := buf.String()
	for _, want := range []string{"get request", "post request", "put request", "delete request", "post response", "status", "body_bytes", "query_keys=a"} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q:\n%s", want, out)
		}
	}
	for _, secret := range []string{"secret-token", `"k"`, "a=1"} {
		if strings.Contains(out, secret) {
			t.Errorf("log output contains %q:\n%s", secret, out)
		}
	}
}

func TestConnectionIsReusedWhenResponseIgnored(t *testing.T) {
	var conns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"padding":"` + strings.Repeat("x", 1000) + `"}`))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()
	c := NewClient(BaseURLString(srv.URL), HTTPClient(&http.Client{Transport: &http.Transport{}}))
	for range 5 {
		if _, err := c.Get(context.Background(), "/", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if n := conns.Load(); n != 1 {
		t.Errorf("%d connections used, want 1", n)
	}
}

func TestConcurrentRequests(t *testing.T) {
	srv, got := recordingServer(t)
	c := NewClient(BaseURLString(srv.URL))
	c.SetAuth("X-Token", "abc")
	var wg sync.WaitGroup
	for i := range 40 {
		wg.Go(func() {
			var m map[string]any
			var err error
			if i%2 == 0 {
				_, err = c.Get(context.Background(), "/x", nil, &m)
			} else {
				_, err = c.Post(context.Background(), "/x", map[string]int{"i": i}, &m)
			}
			if err != nil || m["ok"] != true {
				t.Errorf("request %d: m = %v, err = %v", i, m, err)
			}
		})
	}
	wg.Wait()
	if n := len(got()); n != 40 {
		t.Errorf("server saw %d requests, want 40", n)
	}
}

func TestClientConfigurationErrors(t *testing.T) {
	ctx := context.Background()
	var nilURL *url.URL
	for name, c := range map[string]*Client{
		"unparseable":       NewClient(BaseURLString("http://[::1")),
		"empty":             NewClient(BaseURLString("")),
		"no scheme":         NewClient(BaseURLString("localhost:8080")),
		"unsupported":       NewClient(BaseURLString("ftp://example.com")),
		"no host":           NewClient(BaseURLString("https://")),
		"nil url":           NewClient(BaseURL(nilURL)),
		"zero value client": {},
	} {
		t.Run(name, func(t *testing.T) {
			for _, call := range []func() error{
				func() error { _, err := c.Get(ctx, "/x", nil, nil); return err },
				func() error { _, err := c.Post(ctx, "/x", nil, nil); return err },
				func() error { _, err := c.Put(ctx, "/x", nil, nil); return err },
				func() error { _, err := c.Delete(ctx, "/x", nil); return err },
			} {
				if err := call(); err == nil || !strings.HasPrefix(err.Error(), "rest:") {
					t.Errorf("error = %v, want a configuration error", err)
				}
			}
			c.Auth(ctx, "id", config.Secret("secret")) // must not panic
		})
	}

	t.Run("later valid base URL wins", func(t *testing.T) {
		srv, _ := recordingServer(t)
		c := NewClient(BaseURLString("http://[::1"), BaseURLString(srv.URL))
		if _, err := c.Get(ctx, "/", nil, nil); err != nil {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		c := NewClient()
		if c.BaseURL.String() != "https://localhost" || c.HTTPClient.Timeout != 30*time.Second || c.err != nil {
			t.Errorf("defaults = %v, %v, %v", c.BaseURL, c.HTTPClient.Timeout, c.err)
		}
		if c := NewClient(HTTPClient(nil)); c.HTTPClient == nil {
			t.Error("HTTPClient(nil) must keep the default client")
		}
	})

	t.Run("nil HTTPClient field", func(t *testing.T) {
		srv, _ := recordingServer(t)
		u, _ := url.Parse(srv.URL)
		c := &Client{BaseURL: u}
		if _, err := c.Get(ctx, "/", nil, nil); err != nil {
			t.Errorf("error = %v", err)
		}
	})
}

func TestAuthClientCredentials(t *testing.T) {
	var tokens atomic.Int32
	var failToken atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokens.Add(1)
			if failToken.Load() {
				http.Error(w, "bad client", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok-1","token_type":"Bearer","expires_in":3600}`))
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok-1" {
			http.Error(w, "unauthorised: "+got, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	ctx := context.Background()

	t.Run("empty client ID is a no-op", func(t *testing.T) {
		c := NewClient(BaseURLString(srv.URL))
		hc := c.HTTPClient
		c.Auth(ctx, "", config.Secret(""))
		if c.HTTPClient != hc {
			t.Error("HTTPClient replaced")
		}
	})

	t.Run("token fetched once and reused", func(t *testing.T) {
		tokens.Store(0)
		c := NewClient(BaseURLString(srv.URL), HTTPClient(&http.Client{Timeout: 7 * time.Second}))
		c.Auth(ctx, "id", config.Secret("secret"))
		if c.HTTPClient.Timeout != 7*time.Second {
			t.Errorf("timeout = %v, want it preserved", c.HTTPClient.Timeout)
		}
		for range 3 {
			var m map[string]any
			if _, err := c.Get(ctx, "/data", nil, &m); err != nil || m["ok"] != true {
				t.Fatalf("m = %v, err = %v", m, err)
			}
		}
		if n := tokens.Load(); n != 1 {
			t.Errorf("token endpoint called %d times, want 1", n)
		}
	})

	t.Run("token failure is an error", func(t *testing.T) {
		failToken.Store(true)
		defer failToken.Store(false)
		c := NewClient(BaseURLString(srv.URL))
		c.Auth(ctx, "id", config.Secret("wrong"))
		_, err := c.Get(ctx, "/data", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "bad client") {
			t.Errorf("error = %v, want the token failure", err)
		}
	})
}

func TestAuthSurvivesCancelledContext(t *testing.T) {
	var tokens atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokens.Add(1)
			w.Header().Set("Content-Type", "application/json")
			// already expired, so every request refreshes the token
			_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"Bearer","expires_in":1}`))
			return
		}
		_, _ = w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := NewClient(BaseURLString(srv.URL))
	c.Auth(ctx, "id", config.Secret("secret"))
	cancel()
	for range 2 {
		if _, err := c.Get(context.Background(), "/data", nil, nil); err != nil {
			t.Fatalf("Get() after the Auth context was cancelled: %v", err)
		}
	}
	if n := tokens.Load(); n < 2 {
		t.Errorf("token endpoint called %d times, want a refresh for each request", n)
	}
}

func TestCustomAuthHeaderNotSentToOtherHostOnRedirect(t *testing.T) {
	var other atomic.Value
	other.Store("unset")
	otherSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		other.Store(r.Header.Get("X-Token"))
		_, _ = w.Write([]byte(`ok`))
	}))
	defer otherSrv.Close()

	var same atomic.Value
	same.Store("unset")
	srv := httptest.NewServer(nil)
	defer srv.Close()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/other":
			http.Redirect(w, r, otherSrv.URL+"/final", http.StatusTemporaryRedirect)
		case "/same":
			http.Redirect(w, r, srv.URL+"/final", http.StatusTemporaryRedirect)
		default:
			same.Store(r.Header.Get("X-Token"))
			_, _ = w.Write([]byte(`ok`))
		}
	})

	c := NewClient(BaseURLString(srv.URL))
	c.SetAuth("X-Token", "secret")
	if _, err := c.Post(context.Background(), "/other", `{}`, nil); err != nil {
		t.Fatalf("redirect to other host: %v", err)
	}
	if got := other.Load(); got != "" {
		t.Errorf("X-Token sent to the other host = %q, want none", got)
	}
	if _, err := c.Post(context.Background(), "/same", `{}`, nil); err != nil {
		t.Fatalf("redirect to same host: %v", err)
	}
	if got := same.Load(); got != "secret" {
		t.Errorf("X-Token on a same-host redirect = %q, want it kept", got)
	}
}
