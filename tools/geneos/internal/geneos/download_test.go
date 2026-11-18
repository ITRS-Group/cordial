package geneos

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
// to getting the response headers but not to reading a large body.
func TestDownloadClientTimeouts(t *testing.T) {
	const timeout = 200 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow-body":
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			time.Sleep(3 * timeout)
			_, _ = io.WriteString(w, "complete")
		case "/slow-headers":
			time.Sleep(3 * timeout)
			_, _ = io.WriteString(w, "too late")
		}
	}))
	defer server.Close()

	client := newDownloadClient(timeout)

	resp, err := client.Get(server.URL + "/slow-body")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || string(body) != "complete" {
		t.Errorf("slow body read = %q, %v, want it to complete", body, err)
	}

	if resp, err := client.Get(server.URL + "/slow-headers"); err == nil {
		resp.Body.Close()
		t.Error("request with slow response headers did not time out")
	} else if hinted := withDownloadTimeoutHint(err); !strings.Contains(hinted.Error(), "ITRS_DOWNLOAD_TIMEOUT") {
		t.Errorf("header timeout error %q does not get the timeout hint", err)
	}
}
