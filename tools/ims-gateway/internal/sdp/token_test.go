package sdp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// isolatedTokenDir points the user configuration directory, and the
// key file, at a temporary directory and returns it
func isolatedTokenDir(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses the XDG config directory")
	}

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	orig := DefaultUserKeyfile
	t.Cleanup(func() { DefaultUserKeyfile = orig })
	DefaultUserKeyfile = config.KeyFile(filepath.Join(dir, "keyfile.aes"))
	if _, _, err := DefaultUserKeyfile.ReadOrCreate(host.Localhost); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSaveTokenPermissions(t *testing.T) {
	dir := isolatedTokenDir(t)

	if err := saveToken(&oauth2.Token{AccessToken: "A", RefreshToken: "R", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	// saving again replaces the file, which must keep the permissions
	if err := saveToken(&oauth2.Token{AccessToken: "A2", RefreshToken: "R", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	confDir := filepath.Join(dir, ".config", "geneos")
	for path, want := range map[string]os.FileMode{confDir: 0o700, filepath.Join(confDir, "sdp.token.json"): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", path, got, want)
		}
	}

	// a file that was created too open is tightened
	if err := os.Chmod(filepath.Join(confDir, "sdp.token.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := saveToken(&oauth2.Token{AccessToken: "A3", RefreshToken: "R", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(filepath.Join(confDir, "sdp.token.json")); info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}
}

// a file in the working directory must never take precedence over, or
// be used instead of, the token that is saved
func TestLoadTokenIgnoresWorkingDirectory(t *testing.T) {
	isolatedTokenDir(t)

	work := t.TempDir()
	t.Chdir(work)
	stale := `{"token":{"access_token":"STALE","refresh_token":"STALE","expiry":"2000-01-01T00:00:00Z"}}`
	if err := os.WriteFile(filepath.Join(work, "sdp.token.json"), []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}

	if token, err := loadToken(); err == nil {
		t.Fatalf("loadToken() used the file in the working directory: %+v", token)
	}

	if err := saveToken(&oauth2.Token{AccessToken: "FRESH", RefreshToken: "REFRESH", Expiry: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	token, err := loadToken()
	if err != nil || token.RefreshToken != "REFRESH" {
		t.Errorf("loadToken() = %+v, %v, want the saved token", token, err)
	}
}

// all of the requests that find the token expired must share one refresh
func TestTokenRefreshIsSharedByRequests(t *testing.T) {
	var refreshes atomic.Int32
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer auth.Close()

	expired := &oauth2.Token{AccessToken: "old", RefreshToken: "refresh", Expiry: time.Now().Add(-time.Hour)}
	useToken(t, expired, nil)

	var saved atomic.Int32
	origSaver := tokenSaver
	tokenSaver = func(*oauth2.Token) error { saved.Add(1); return nil }
	t.Cleanup(func() { tokenSaver = origSaver })

	var authorizations sync.Map
	s := newSDPStub(t)
	s.list = func(w http.ResponseWriter, r *http.Request) {
		authorizations.Store(r.Header.Get("Authorization"), true)
		reply(200, noRequests)(w, r)
	}

	cf := testConfig(s.URL)
	config.Set(cf, "sdp.datacentres.t.auth", auth.URL)
	config.Set(cf, "sdp.timeout", "5s")

	var wg sync.WaitGroup
	var failures atomic.Int32
	for range 8 {
		wg.Go(func() {
			if rec, _ := call(get, cf, http.MethodGet, "/sdp", ""); rec.Code != http.StatusOK {
				failures.Add(1)
			}
		})
	}
	wg.Wait()

	if failures.Load() != 0 || refreshes.Load() != 1 || saved.Load() != 1 {
		t.Errorf("failures = %d, refreshes = %d, saves = %d, want 0, 1, 1", failures.Load(), refreshes.Load(), saved.Load())
	}
	if _, ok := authorizations.Load("Bearer new"); !ok {
		t.Error("requests did not use the refreshed token")
	}

	// the token is kept for later requests, without reading it again
	tokenLoader = func() (*oauth2.Token, error) { return nil, errNoToken }
	if rec, _ := call(get, cf, http.MethodGet, "/sdp", ""); rec.Code != http.StatusOK || refreshes.Load() != 1 {
		t.Errorf("status = %d, refreshes = %d", rec.Code, refreshes.Load())
	}
}
