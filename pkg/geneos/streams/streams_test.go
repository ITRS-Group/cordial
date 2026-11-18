package streams

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRESTStreamWrite(t *testing.T) {
	var method, path, body atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		method.Store(r.Method)
		path.Store(r.URL.Path)
		body.Store(string(b))
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	s, err := NewRESTStream(u, "ent", "smp", "str")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	n, err := s.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	if method.Load() != "PUT" || path.Load() != "/managedEntity/ent/sampler/smp/stream/str" || body.Load() != "hello" {
		t.Errorf("request = %v %v %v", method.Load(), path.Load(), body.Load())
	}
}

func TestRESTStreamServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "no such stream")
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	s, _ := NewRESTStream(u, "e", "s", "x")
	n, err := s.Write([]byte("data"))
	if !errors.Is(err, ErrServerError) || !strings.Contains(err.Error(), "no such stream") || n != 0 {
		t.Errorf("n = %d, err = %v", n, err)
	}
}

// TLS certificates must be verified unless SkipVerify is given
func TestRESTStreamVerifiesTLSByDefault(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)

	s, _ := NewRESTStream(u, "e", "s", "x")
	if _, err := s.Write([]byte("data")); err == nil {
		t.Error("write to a server with an untrusted certificate succeeded without SkipVerify")
	}

	s, _ = NewRESTStream(u, "e", "s", "x", SkipVerify(true))
	if n, err := s.Write([]byte("data")); err != nil || n != 4 {
		t.Errorf("SkipVerify(true): n = %d, err = %v", n, err)
	}

	s, _ = NewRESTStream(u, "e", "s", "x", SkipVerify(false))
	if _, err := s.Write([]byte("data")); err == nil {
		t.Error("SkipVerify(false) must verify certificates")
	}
}

func TestRESTStreamTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	u, _ := url.Parse(srv.URL)
	s, _ := NewRESTStream(u, "e", "s", "x", RESTTimeout(100*time.Millisecond))
	start := time.Now()
	if _, err := s.Write([]byte("data")); err == nil {
		t.Fatal("expected a timeout")
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("write took %s", time.Since(start))
	}
}
