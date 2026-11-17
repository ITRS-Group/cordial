package client

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/integrations/servicenow2/cmd/proxy"
)

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

// fakeProxy is a fake proxy that counts requests and records the last
// one
type fakeProxy struct {
	*httptest.Server
	calls atomic.Int32
	last  struct {
		path, auth, body string
	}
}

func newFakeProxy(t *testing.T, h http.HandlerFunc) *fakeProxy {
	t.Helper()
	p := &fakeProxy{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.calls.Add(1)
		b, _ := io.ReadAll(r.Body)
		p.last.path, p.last.auth, p.last.body = r.URL.Path, r.Header.Get("Authorization"), string(b)
		h(w, r)
	}))
	t.Cleanup(p.Close)
	return p
}

// respond returns a handler sending status code and JSON body
func respond(code int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b []byte
		switch v := body.(type) {
		case string:
			b = []byte(v)
		default:
			b, _ = json.Marshal(v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write(b)
	}
}

func created(number string) proxy.Response {
	return proxy.Response{Action: proxy.Created, Number: number, Result: "Created " + number}
}

func failedResponse(msg string) proxy.Response {
	return proxy.Response{Action: proxy.Failed, Error: msg, Result: "failed: " + msg}
}

// deadURL returns a URL that nothing is listening on
func deadURL(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	u := "http://" + l.Addr().String()
	l.Close()
	return u
}

func clientConfig(urls ...string) *config.Config {
	cf := config.New()
	config.Set(cf, "proxy.url", urls)
	config.Set(cf, "proxy.authentication.token", "tok")
	config.Set(cf, "proxy.timeout", "300ms")
	return cf
}

func send(cf *config.Config, args ...string) (out string, err error) {
	var buf bytes.Buffer
	err = submit(context.Background(), cf, submitOptions{}, args, &buf)
	return buf.String(), err
}

func TestSubmitSuccess(t *testing.T) {
	for _, action := range []string{proxy.Created, proxy.Updated, proxy.Ignored} {
		t.Run(action, func(t *testing.T) {
			p := newFakeProxy(t, respond(200, proxy.Response{Action: action, Result: "result for " + action}))
			out, err := send(clientConfig(p.URL), "cmdb_ci=abc")
			if err != nil || out != "result for "+action+"\n" {
				t.Fatalf("submit() = %q, %v", out, err)
			}
		})
	}

	t.Run("request contents", func(t *testing.T) {
		p := newFakeProxy(t, respond(201, created("INC1")))
		cf := clientConfig(p.URL)
		config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{"short_description": "from defaults"}}})

		_, err := send(cf, "cmdb_ci=abc", "_correlation_id=raw", "ignored")
		if err != nil {
			t.Fatal(err)
		}
		if p.last.path != "/incident" || p.last.auth != "Bearer tok" {
			t.Errorf("request path %q, auth %q", p.last.path, p.last.auth)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(p.last.body), &got); err != nil {
			t.Fatal(err)
		}
		want := fmt.Sprintf("%x", sha1.Sum([]byte("raw")))
		if got["cmdb_ci"] != "abc" || got["short_description"] != "from defaults" ||
			got["correlation_id"] != want || got["_correlation_id"] != "" {
			t.Errorf("unexpected request body %v, want correlation_id %s", got, want)
		}
	})

	t.Run("quiet", func(t *testing.T) {
		p := newFakeProxy(t, respond(201, created("INC1")))
		var buf bytes.Buffer
		if err := submit(context.Background(), clientConfig(p.URL), submitOptions{quiet: true}, nil, &buf); err != nil || buf.Len() != 0 {
			t.Errorf("submit() = %q, %v", buf.String(), err)
		}
	})

	t.Run("table", func(t *testing.T) {
		p := newFakeProxy(t, respond(201, created("INC1")))
		if err := submit(context.Background(), clientConfig(p.URL), submitOptions{table: "problem"}, nil, io.Discard); err != nil || p.last.path != "/problem" {
			t.Errorf("submit() = %v, path %q", err, p.last.path)
		}
		if _, err := send(clientConfig(p.URL), "_table=change_request"); err != nil || p.last.path != "/change_request" {
			t.Errorf("submit() = %v, path %q", err, p.last.path)
		}
	})
}

// TestSubmitFailures checks that every way the proxy can answer, other
// than a clear success, results in an error and so a non-zero exit code
func TestSubmitFailures(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{"failed action with 200", respond(200, failedResponse("something broke")), "something broke"},
		{"failed action without error text", respond(200, proxy.Response{Action: proxy.Failed, Result: "result text"}), "result text"},
		{"empty object", respond(200, `{}`), "unexpected response"},
		{"null", respond(200, `null`), "unexpected response"},
		{"unknown action", respond(200, `{"_action":"Exploded"}`), "Exploded"},
		{"foreign json", respond(200, `{"status":"ok"}`), "unexpected response"},
		{"truncated json", respond(200, `{"_action":"Cre`), ""},
		{"html page", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html>captive portal</html>")
		}, ""},
		{"no content", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }, ""},
		{"bad request", respond(400, failedResponse("field names are invalid")), "field names are invalid"},
		{"unauthorised", respond(401, `{"message":"missing or invalid token"}`), "missing or invalid token"},
		{"not found", respond(404, `{"message":"Not Found"}`), "404"},
		{"server error", respond(500, `{"message":"internal"}`), "internal"},
		{"plain text error", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "nope", 403) }, "nope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := newFakeProxy(t, test.handler)
			out, err := send(clientConfig(p.URL))
			if err == nil {
				t.Fatalf("submit() succeeded with output %q, want an error", out)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, test.wantErr)
			}
			if out != "" {
				t.Errorf("failure produced output %q", out)
			}
		})
	}

	t.Run("redirects are not followed", func(t *testing.T) {
		for _, code := range []int{301, 302, 303, 307, 308} {
			var target atomic.Int32
			dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				target.Add(1)
				respond(201, created("INC1"))(w, r)
			}))
			p := newFakeProxy(t, func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, dest.URL, code)
			})
			if _, err := send(clientConfig(p.URL)); err == nil || target.Load() != 0 {
				t.Errorf("status %d: error = %v, redirect target hit %d times", code, err, target.Load())
			}
			dest.Close()
		}
	})

	t.Run("no proxy configured", func(t *testing.T) {
		for name, cf := range map[string]*config.Config{"empty": config.New(), "empty list": clientConfig()} {
			if _, err := send(cf); err == nil || !strings.Contains(err.Error(), "no proxy URL") {
				t.Errorf("%s: error = %v", name, err)
			}
		}
	})

	t.Run("invalid defaults", func(t *testing.T) {
		cf := clientConfig(newFakeProxy(t, respond(201, created("INC1"))).URL)
		config.Set(cf, "defaults", "not a list of groups")
		if _, err := send(cf); err == nil || !strings.Contains(err.Error(), "defaults") {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("invalid profile", func(t *testing.T) {
		p := newFakeProxy(t, respond(201, created("INC1")))
		cf := clientConfig(p.URL)
		config.Set(cf, "profiles.broken", "not a list of groups")
		var buf bytes.Buffer
		err := submit(context.Background(), cf, submitOptions{profile: "broken"}, nil, &buf)
		if err == nil || !strings.Contains(err.Error(), "broken") || p.calls.Load() != 0 {
			t.Errorf("error = %v, calls = %d", err, p.calls.Load())
		}
	})
}

func TestSubmitProxyFallback(t *testing.T) {
	ok := func(t *testing.T) *fakeProxy { return newFakeProxy(t, respond(201, created("INC9"))) }

	t.Run("unreachable proxy", func(t *testing.T) {
		good := ok(t)
		out, err := send(clientConfig(deadURL(t), good.URL))
		if err != nil || out != "Created INC9\n" || good.calls.Load() != 1 {
			t.Errorf("submit() = %q, %v (calls %d)", out, err, good.calls.Load())
		}
	})

	t.Run("timeout", func(t *testing.T) {
		slow := newFakeProxy(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(600 * time.Millisecond) })
		good := ok(t)
		if _, err := send(clientConfig(slow.URL, good.URL)); err != nil || good.calls.Load() != 1 {
			t.Errorf("submit() = %v (calls %d)", err, good.calls.Load())
		}
	})

	t.Run("proxy unavailable statuses", func(t *testing.T) {
		for _, code := range []int{502, 503, 504} {
			bad := newFakeProxy(t, respond(code, failedResponse("upstream down")))
			good := ok(t)
			if _, err := send(clientConfig(bad.URL, good.URL)); err != nil || good.calls.Load() != 1 {
				t.Errorf("status %d: submit() = %v (calls %d)", code, err, good.calls.Load())
			}
		}
	})

	t.Run("definitive answers are not retried", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 404, 413, 500} {
			first := newFakeProxy(t, respond(code, failedResponse("rejected")))
			second := ok(t)
			if _, err := send(clientConfig(first.URL, second.URL)); err == nil {
				t.Errorf("status %d: submit() succeeded", code)
			}
			if second.calls.Load() != 0 {
				t.Errorf("status %d: second proxy was tried", code)
			}
		}
	})

	t.Run("failed action is not retried", func(t *testing.T) {
		first := newFakeProxy(t, respond(200, failedResponse("no")))
		second := ok(t)
		if _, err := send(clientConfig(first.URL, second.URL)); err == nil || second.calls.Load() != 0 {
			t.Errorf("error = %v, second proxy calls = %d", err, second.calls.Load())
		}
	})

	t.Run("garbage from first proxy is not retried", func(t *testing.T) {
		first := newFakeProxy(t, respond(200, `{"unrelated":true}`))
		second := ok(t)
		if _, err := send(clientConfig(first.URL, second.URL)); err == nil || second.calls.Load() != 0 {
			t.Errorf("error = %v, second proxy calls = %d", err, second.calls.Load())
		}
	})

	t.Run("all proxies fail", func(t *testing.T) {
		dead := deadURL(t)
		unavailable := newFakeProxy(t, respond(503, failedResponse("busy")))
		_, err := send(clientConfig(dead, unavailable.URL))
		if err == nil || !strings.Contains(err.Error(), dead) || !strings.Contains(err.Error(), unavailable.URL) {
			t.Errorf("error = %v, want both proxies reported", err)
		}
	})

	t.Run("success stops at first proxy", func(t *testing.T) {
		first, second := ok(t), ok(t)
		if _, err := send(clientConfig(first.URL, second.URL)); err != nil || second.calls.Load() != 0 {
			t.Errorf("error = %v, second proxy calls = %d", err, second.calls.Load())
		}
	})
}

func TestSubmitArgumentsAndProfiles(t *testing.T) {
	p := newFakeProxy(t, respond(201, created("INC1")))
	cf := clientConfig(p.URL)
	config.Set(cf, "defaults", []any{map[string]any{"set": map[string]any{"a": "default", "b": "default", "_profile": "special"}}})
	config.Set(cf, "profiles.special", []any{map[string]any{"set": map[string]any{"b": "profile"}}})

	if _, err := send(cf, "c=arg", "a=", "bare", "=nokey"); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(p.last.body), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["a"]; ok || got["b"] != "profile" || got["c"] != "arg" || got["bare"] != "" {
		t.Errorf("unexpected request body %v", got)
	}

	// an undefined profile only warns and the defaults are still used
	if err := submit(context.Background(), cf, submitOptions{profile: "missing"}, nil, io.Discard); err != nil {
		t.Errorf("undefined profile: %v", err)
	}
}

func TestCheckResponse(t *testing.T) {
	for _, action := range []string{proxy.Created, proxy.Updated, proxy.Ignored} {
		if err := checkResponse(proxy.Response{Action: action}); err != nil {
			t.Errorf("checkResponse(%q) = %v", action, err)
		}
	}
	for _, action := range []string{"", proxy.Failed, "created", "Unknown"} {
		if err := checkResponse(proxy.Response{Action: action}); err == nil {
			t.Errorf("checkResponse(%q) = nil, want an error", action)
		}
	}
}

func TestRunQuery(t *testing.T) {
	results := `{"fields":["number","short_description"],"results":[{"number":"INC1","short_description":"a, b\nc"}]}`

	t.Run("csv", func(t *testing.T) {
		p := newFakeProxy(t, respond(200, results))
		var buf bytes.Buffer
		if err := runQuery(context.Background(), clientConfig(p.URL), "", "active=true", "csv", false, &buf); err != nil {
			t.Fatal(err)
		}
		want := "number,short_description\nINC1,a\\, b\\nc\n<!>table,incident\n<!>results,1\n<!>query,active=true\n"
		if buf.String() != want {
			t.Errorf("output = %q, want %q", buf.String(), want)
		}
	})

	t.Run("json", func(t *testing.T) {
		p := newFakeProxy(t, respond(200, results))
		var buf bytes.Buffer
		if err := runQuery(context.Background(), clientConfig(p.URL), "", "q", "json", false, &buf); err != nil || !strings.Contains(buf.String(), `"INC1"`) {
			t.Errorf("output = %q, %v", buf.String(), err)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		p := newFakeProxy(t, respond(200, results))
		if err := runQuery(context.Background(), clientConfig(deadURL(t), p.URL), "", "q", "csv", false, io.Discard); err != nil {
			t.Error(err)
		}
	})

	for name, h := range map[string]http.HandlerFunc{
		"unauthorised":  respond(401, `{"message":"missing or invalid token"}`),
		"server error":  respond(500, `{"message":"boom"}`),
		"empty object":  respond(200, `{}`),
		"foreign json":  respond(200, `{"status":"ok"}`),
		"not json":      func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "hello") },
		"truncated":     respond(200, `{"fields":["a"`),
		"no content":    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
		"redirect":      func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/x", http.StatusFound) },
		"disabled/404s": respond(404, `{"message":"Not Found"}`),
	} {
		t.Run(name, func(t *testing.T) {
			p := newFakeProxy(t, h)
			var buf bytes.Buffer
			if err := runQuery(context.Background(), clientConfig(p.URL), "", "q", "csv", false, &buf); err == nil {
				t.Errorf("runQuery() succeeded with output %q", buf.String())
			}
			if buf.Len() != 0 {
				t.Errorf("failure produced output %q", buf.String())
			}
		})
	}

	t.Run("no proxy", func(t *testing.T) {
		if err := runQuery(context.Background(), config.New(), "", "q", "csv", false, io.Discard); err == nil {
			t.Error("runQuery() succeeded with no proxy configured")
		}
	})
}
