package geneos

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestSafeFilename(t *testing.T) {
	for name, want := range map[string]string{
		"release.tar.gz":         "release.tar.gz",
		"../../etc/passwd":       "passwd",
		"/abs/path/release.zip":  "release.zip",
		`..\..\windows\evil.zip`: "evil.zip",
		"dir/":                   "",
		"..":                     "",
		".":                      "",
		"":                       "",
	} {
		if got := safeFilename(name); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestFilenameFromHTTPRespIsSanitised(t *testing.T) {
	u, _ := url.Parse("https://example.test/download/real-name.tar.gz")

	tests := []struct {
		disposition string
		want        string
	}{
		{`attachment; filename="geneos-gateway-7.1.0.tar.gz"`, "geneos-gateway-7.1.0.tar.gz"},
		{`attachment; filename="../../../.ssh/authorized_keys"`, "authorized_keys"},
		{`attachment; filename="/etc/cron.d/evil"`, "evil"},
		// an unusable name falls back to the URL
		{`attachment; filename=".."`, "real-name.tar.gz"},
		{``, "real-name.tar.gz"},
	}

	for _, tt := range tests {
		resp := &http.Response{Header: http.Header{}, Request: &http.Request{}}
		if tt.disposition != "" {
			resp.Header.Set("Content-Disposition", tt.disposition)
		}
		got, err := FilenameFromHTTPResp(resp, u)
		if err != nil || got != tt.want {
			t.Errorf("FilenameFromHTTPResp(%q) = %q, %v, want %q", tt.disposition, got, err, tt.want)
		}
	}
}

// TestOpenRemoteDefaultArchiveRetriesWithBody checks that each of the
// authenticated requests has a body, including the retry for a platform
// that is not available.
func TestOpenRemoteDefaultArchiveRetriesWithBody(t *testing.T) {
	setTestHosts(t, t.TempDir())

	var mutex sync.Mutex
	var posts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		mutex.Lock()
		posts = append(posts, string(body))
		mutex.Unlock()

		var auth downloadauth
		if err := json.Unmarshal(body, &auth); err != nil || auth.Username != "user" || auth.Password != "secret" {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		// the platform specific release does not exist
		if strings.Contains(r.URL.Query().Get("title"), "el9") {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, "archive")
	}))
	defer server.Close()

	previousURL := config.Get[string](config.Global(), "download.url")
	t.Cleanup(func() { config.Set(config.Global(), "download.url", previousURL) })
	config.Set(config.Global(), "download.url", server.URL+"/")

	ct := &Component{Name: "retrytest", DownloadBase: DownloadBases{Default: "retrytest"}}
	opts := evalOptions(
		Version("latest"),
		SetPlatformID("platform:el9"),
		Username("user"),
		Password(config.Secret("secret")),
	)

	_, resp, err := openRemoteDefaultArchive(ct, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %s, want the platform-less request to succeed", resp.Status)
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(posts) != 2 {
		t.Fatalf("server received %d authenticated requests, want 2: %q", len(posts), posts)
	}
}

func TestOpenRemoteDefaultArchiveNoSources(t *testing.T) {
	setTestHosts(t, t.TempDir())
	previousURL := config.Get[string](config.Global(), "download.url")
	t.Cleanup(func() { config.Set(config.Global(), "download.url", previousURL) })
	config.Set(config.Global(), "download.url", "http://127.0.0.1:0/")

	// a component with no download source must be an error and not a
	// nil response for the caller to dereference
	_, _, err := openRemoteArchive(&Component{Name: "nosource"}, Version("latest"))
	if err == nil {
		t.Fatal("openRemoteArchive() with no download source succeeded")
	}
}

// TestDownloadClientTimeouts checks that the download timeout applies
// to getting the response headers and to waiting for body data, but not
// to the time taken to read a large body.
func TestDownloadClientTimeouts(t *testing.T) {
	const timeout = 200 * time.Millisecond

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow-body":
			// a transfer that takes 3 times the timeout, but always has data
			w.WriteHeader(http.StatusOK)
			for range 12 {
				_, _ = io.WriteString(w, "data")
				w.(http.Flusher).Flush()
				time.Sleep(timeout / 4)
			}
		case "/stalled-body":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "partial")
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-release:
			}
		case "/slow-headers":
			time.Sleep(3 * timeout)
			_, _ = io.WriteString(w, "too late")
		}
	}))
	defer server.Close()
	defer close(release)

	client := newDownloadClient(timeout)

	start := time.Now()
	resp, err := client.Get(server.URL + "/slow-body")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if want := strings.Repeat("data", 12); err != nil || string(body) != want {
		t.Errorf("slow body read = %q, %v, want it to complete", body, err)
	}
	if time.Since(start) < 2*timeout {
		t.Errorf("slow body took %s, which is not longer than the timeout", time.Since(start))
	}

	t.Run("stalled body", func(t *testing.T) {
		start := time.Now()
		resp, err := client.Get(server.URL + "/stalled-body")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if string(body) != "partial" {
			t.Errorf("body read = %q, want the data before the stall", body)
		}
		if err == nil {
			t.Fatal("a stalled body did not fail")
		}
		if !errors.Is(err, os.ErrDeadlineExceeded) || !strings.Contains(err.Error(), "download stalled: no data received for 200ms") {
			t.Errorf("error = %q, want a clear stalled download error", err)
		}
		if elapsed := time.Since(start); elapsed > 5*timeout {
			t.Errorf("stalled body took %s to fail", elapsed)
		}
		if hinted := withDownloadTimeoutHint(err); !strings.Contains(hinted.Error(), "ITRS_DOWNLOAD_TIMEOUT") {
			t.Errorf("stalled error %q does not get the timeout hint", err)
		}
	})

	t.Run("time between reads is not idle time", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/slow-body")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		buf := make([]byte, 4)
		var got []byte
		for {
			n, err := resp.Body.Read(buf)
			got = append(got, buf[:n]...)
			if err != nil {
				if err != io.EOF {
					t.Fatalf("read error %v after %q", err, got)
				}
				break
			}
			// a slow consumer, longer than the timeout in total
			time.Sleep(timeout / 8)
		}
		if string(got) != strings.Repeat("data", 12) {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		resp, err := newDownloadClient(0).Get(server.URL + "/slow-body")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if _, ok := resp.Body.(*idleTimeoutBody); ok {
			t.Error("the body has an idle timeout when the timeout is zero")
		}
	})

	if resp, err := client.Get(server.URL + "/slow-headers"); err == nil {
		resp.Body.Close()
		t.Error("request with slow response headers did not time out")
	} else if hinted := withDownloadTimeoutHint(err); !strings.Contains(hinted.Error(), "ITRS_DOWNLOAD_TIMEOUT") {
		t.Errorf("header timeout error %q does not get the timeout hint", err)
	}
}

func TestDownloadTimeoutSetting(t *testing.T) {
	cf := config.New(config.WithEnvs("ITRS", "_"))
	if got := downloadTimeout(cf); got != 60*time.Second {
		t.Errorf("default download timeout = %s, want 60s", got)
	}
	t.Setenv("ITRS_DOWNLOAD_TIMEOUT", "5m")
	if got := downloadTimeout(cf); got != 5*time.Minute {
		t.Errorf("download timeout with ITRS_DOWNLOAD_TIMEOUT=5m = %s, want 5m", got)
	}
	config.Set(cf, cf.Join("download", "timeout"), "90s")
	if got := downloadTimeout(cf); got != 90*time.Second {
		t.Errorf("download timeout set to 90s = %s", got)
	}
}
