package ims

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/rest"
)

type testServer struct {
	*httptest.Server
	calls atomic.Int32
	path  atomic.Value
}

func newTestServer(t *testing.T, h http.HandlerFunc) *testServer {
	t.Helper()
	s := &testServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls.Add(1)
		s.path.Store(r.URL.Path)
		h(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func status(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}
}

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

func tryConfig(urls ...string) *config.Config {
	cf := config.New()
	config.Set(cf, "url", urls)
	config.Set(cf, "timeout", "300ms")
	return cf
}

// post is the typical use of Try, a request that must succeed
func post(cf *config.Config, suffix string) error {
	return Try(cf, suffix, func(rc *rest.Client) error {
		_, err := rc.Post(context.Background(), "x", "{}", nil)
		return err
	})
}

func TestTrySuccessStopsAtFirstServer(t *testing.T) {
	first := newTestServer(t, status(204, ""))
	second := newTestServer(t, status(204, ""))
	if err := post(tryConfig(first.URL, second.URL), ""); err != nil || first.calls.Load() != 1 || second.calls.Load() != 0 {
		t.Errorf("err = %v, calls %d, %d", err, first.calls.Load(), second.calls.Load())
	}
}

func TestTrySuffix(t *testing.T) {
	s := newTestServer(t, status(204, ""))
	if err := post(tryConfig(s.URL+"/"), "snow"); err != nil || s.path.Load() != "/snow/x" {
		t.Errorf("with suffix: err = %v, path %v", err, s.path.Load())
	}
	if err := post(tryConfig(s.URL+"/api/"), ""); err != nil || s.path.Load() != "/api/x" {
		t.Errorf("without suffix: err = %v, path %v", err, s.path.Load())
	}
}

func TestTryNoURL(t *testing.T) {
	for name, cf := range map[string]*config.Config{"unset": config.New(), "empty": tryConfig()} {
		if err := post(cf, ""); !errors.Is(err, ErrNoURL) {
			t.Errorf("%s: err = %v, want ErrNoURL", name, err)
		}
	}
}

func TestTryFallback(t *testing.T) {
	ok := func() *testServer { return newTestServer(t, status(204, "")) }

	t.Run("unreachable", func(t *testing.T) {
		good := ok()
		if err := post(tryConfig(deadURL(t), good.URL), ""); err != nil || good.calls.Load() != 1 {
			t.Errorf("err = %v, calls %d", err, good.calls.Load())
		}
	})

	t.Run("timeout", func(t *testing.T) {
		slow := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(600 * time.Millisecond) })
		good := ok()
		if err := post(tryConfig(slow.URL, good.URL), ""); err != nil || good.calls.Load() != 1 {
			t.Errorf("err = %v, calls %d", err, good.calls.Load())
		}
	})

	t.Run("unavailable statuses", func(t *testing.T) {
		for _, code := range []int{502, 503, 504} {
			bad := newTestServer(t, status(code, `{"error":"down"}`))
			good := ok()
			if err := post(tryConfig(bad.URL, good.URL), ""); err != nil || good.calls.Load() != 1 {
				t.Errorf("status %d: err = %v, calls %d", code, err, good.calls.Load())
			}
		}
	})

	t.Run("definitive answers are not retried", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 404, 413, 500} {
			first := newTestServer(t, status(code, `{"error":"rejected"}`))
			second := ok()
			if err := post(tryConfig(first.URL, second.URL), ""); err == nil || second.calls.Load() != 0 {
				t.Errorf("status %d: err = %v, second calls %d", code, err, second.calls.Load())
			}
		}
	})

	t.Run("errors from the function are not retried", func(t *testing.T) {
		first, second := ok(), ok()
		err := Try(tryConfig(first.URL, second.URL), "", func(rc *rest.Client) error { return errors.New("rejected response") })
		if err == nil || second.calls.Load() != 0 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("all fail", func(t *testing.T) {
		dead := deadURL(t)
		busy := newTestServer(t, status(503, `{"error":"busy"}`))
		err := post(tryConfig(dead, busy.URL), "")
		if err == nil || !strings.Contains(err.Error(), dead) || !strings.Contains(err.Error(), busy.URL) || !strings.Contains(err.Error(), "busy") {
			t.Errorf("err = %v, want both servers reported", err)
		}
	})

	t.Run("redirects are not followed", func(t *testing.T) {
		var target atomic.Int32
		dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { target.Add(1) }))
		defer dest.Close()
		s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dest.URL, http.StatusFound) })
		if err := post(tryConfig(s.URL), ""); err == nil || target.Load() != 0 {
			t.Errorf("err = %v, target hit %d times", err, target.Load())
		}
	})
}

func TestRetryable(t *testing.T) {
	for code, want := range map[int]bool{400: false, 401: false, 404: false, 500: false, 502: true, 503: true, 504: true} {
		if got := Retryable(&rest.HTTPError{StatusCode: code}); got != want {
			t.Errorf("Retryable(HTTP %d) = %t, want %t", code, got, want)
		}
	}
	if Retryable(errors.New("decode failure")) {
		t.Error("an error that is not a transport failure must not be retried")
	}
	_, err := rest.NewClient(rest.BaseURLString(deadURL(t))).Post(context.Background(), "x", "{}", nil)
	if !Retryable(err) {
		t.Errorf("Retryable(%v) = false, want true", err)
	}
}

func TestDescribeError(t *testing.T) {
	for name, body := range map[string]string{
		"gateway": `{"error":"the problem"}`,
		"proxy":   `{"_error":"the problem","result":"other"}`,
		"generic": `{"message":"the problem"}`,
	} {
		orig := &rest.HTTPError{StatusCode: 400, Status: "400 Bad Request", Body: []byte(body)}
		err := DescribeError(orig)
		if !strings.Contains(err.Error(), "the problem") || strings.Contains(err.Error(), "{") {
			t.Errorf("%s: DescribeError() = %q", name, err)
		}
		if he, ok := errors.AsType[*rest.HTTPError](err); !ok || he != orig {
			t.Errorf("%s: original error cannot be recovered", name)
		}
	}

	// bodies that cannot be described leave the error as it was
	for _, body := range []string{"", "Not Found", `{}`, `{"other":"x"}`, `{"error":`} {
		orig := &rest.HTTPError{StatusCode: 404, Status: "404 Not Found", Body: []byte(body)}
		if DescribeError(orig) != error(orig) {
			t.Errorf("body %q: error was changed", body)
		}
	}
	if plain := errors.New("x"); DescribeError(plain) != plain {
		t.Error("non-HTTP errors must be returned unchanged")
	}
}
