package instance

import (
	"encoding/json"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const snapshotTestXPath = `//dataview[(@name="test")]`

func snapshotTestServer(t *testing.T, prefix string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var pings atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc(prefix+"/gatewayinfo/timezone", func(w http.ResponseWriter, _ *http.Request) {
		pings.Add(1)
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc(prefix+"/xpaths/match", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"xpaths": []string{snapshotTestXPath},
		})
	})
	return httptest.NewServer(mux), &pings
}

func TestSnapshotGatewayURLs(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		want         []string
		autoFallback bool
		wantError    bool
	}{
		{name: "HTTP URL", endpoint: "http://gateway.example:8080/proxy", want: []string{"http://gateway.example:8080/proxy"}},
		{name: "HTTPS URL", endpoint: "https://gateway.example/proxy", want: []string{"https://gateway.example/proxy"}},
		{name: "host port", endpoint: "gateway.example:8443", want: []string{"https://gateway.example:8443", "http://gateway.example:8443"}, autoFallback: true},
		{name: "IPv6 host port", endpoint: "[::1]:8443", want: []string{"https://[::1]:8443", "http://[::1]:8443"}, autoFallback: true},
		{name: "URL missing host", endpoint: "https:///proxy", wantError: true},
		{name: "unsupported URL scheme", endpoint: "ftp://gateway.example", wantError: true},
		{name: "host missing port", endpoint: "gateway.example", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			urls, fallback, err := snapshotGatewayURLs(nil, tc.endpoint)
			if (err != nil) != tc.wantError {
				t.Fatalf("snapshotGatewayURLs() error = %v, want error %v", err, tc.wantError)
			}
			if tc.wantError {
				return
			}
			if fallback != tc.autoFallback {
				t.Errorf("fallback = %v, want %v", fallback, tc.autoFallback)
			}
			if len(urls) != len(tc.want) {
				t.Fatalf("URLs = %v, want %v", urls, tc.want)
			}
			for n := range urls {
				if urls[n].String() != tc.want[n] {
					t.Errorf("URL %d = %q, want %q", n, urls[n], tc.want[n])
				}
			}
		})
	}
}

func TestSnapshotDirectEndpointUsesReverseProxyPath(t *testing.T) {
	server, pings := snapshotTestServer(t, "/geneos/rest")
	t.Cleanup(server.Close)

	values, err := SnapshotDataviews(
		nil,
		[]string{snapshotTestXPath, snapshotTestXPath},
		SnapshotEndpoint(server.URL+"/geneos/rest"),
		SnapshotXPathOnly(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 {
		t.Fatalf("got %d dataviews, want 2", len(values))
	}
	if got := pings.Load(); got != 1 {
		t.Errorf("Gateway was pinged %d times, want 1 connection test", got)
	}
}

func TestSnapshotHostPortFallsBackFromTLSOnlyForPlainHTTP(t *testing.T) {
	server, _ := snapshotTestServer(t, "/rest")
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")

	if _, err := SnapshotDataviews(
		nil,
		[]string{snapshotTestXPath},
		SnapshotEndpoint(address),
		SnapshotXPathOnly(true),
	); err != nil {
		t.Fatalf("HOST:PORT plain HTTP fallback failed: %v", err)
	}

	tlsServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	tlsServer.Config.ErrorLog = stdlog.New(io.Discard, "", 0)
	tlsServer.StartTLS()
	t.Cleanup(tlsServer.Close)

	_, err := SnapshotDataviews(
		nil,
		[]string{snapshotTestXPath},
		SnapshotEndpoint(strings.TrimPrefix(tlsServer.URL, "https://")),
		SnapshotXPathOnly(true),
	)
	if err == nil {
		t.Fatal("HOST:PORT TLS certificate error was ignored")
	}
	if !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("error = %v, want a certificate-validation error", err)
	}
}

func TestSnapshotHostPortDoesNotFallbackOnConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	_, err = SnapshotDataviews(
		nil,
		[]string{snapshotTestXPath},
		SnapshotEndpoint(address),
		SnapshotXPathOnly(true),
	)
	if err == nil {
		t.Fatal("snapshot unexpectedly connected")
	}
	if !strings.Contains(err.Error(), `gateway "https://`) || strings.Contains(err.Error(), `gateway "http://`) {
		t.Errorf("connection error = %v; fallback occurred for a non-TLS-protocol failure", err)
	}
}
